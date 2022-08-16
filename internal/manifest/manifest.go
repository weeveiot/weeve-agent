package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/go-connections/nat"
	"github.com/weeveiot/weeve-agent/internal/model"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"
)

type Manifest struct {
	ID               string
	ManifestUniqueID model.ManifestUniqueID
	VersionNumber    float64
	Modules          []ContainerConfig
	Labels           map[string]string
	Connections      connectionsInt
}

// This struct holds information for starting a container
type ContainerConfig struct {
	ContainerName  string
	ImageName      string
	ImageTag       string
	EntryPointArgs []string
	EnvArgs        []string
	NetworkName    string
	NetworkDriver  string
	ExposedPorts   nat.PortSet // This must be set for the container create
	PortBinding    nat.PortMap // This must be set for the containerStart
	NetworkConfig  network.NetworkingConfig
	MountConfigs   []mount.Mount
	Labels         map[string]string
	Registry       RegistryDetails
	Resources      container.Resources
}

type RegistryDetails struct {
	Url       string
	ImageName string
	UserName  string
	Password  string
}

type connectionsInt map[int][]int
type connectionsString map[string][]string

func GetManifest(jsonParsed *gabs.Container) (Manifest, error) {
	manifestID := jsonParsed.Search("_id").Data().(string)
	manifestName := jsonParsed.Search("manifestName").Data().(string)
	versionNumber := jsonParsed.Search("versionNumber").Data().(float64)
	labels := map[string]string{
		"manifestID":    manifestID,
		"manifestName":  manifestName,
		"versionNumber": fmt.Sprint(versionNumber),
	}

	connections, err := getConnections(jsonParsed)
	if err != nil {
		return Manifest{}, err
	}

	var containerConfigs []ContainerConfig

	modules := jsonParsed.Search("modules").Children()
	for _, module := range modules {
		var containerConfig ContainerConfig

		containerConfig.ImageName = module.Search("image").Search("name").Data().(string)
		imageTag := module.Search("image").Search("tag").Data()
		if imageTag != nil {
			containerConfig.ImageTag = imageTag.(string)
		}
		containerConfig.Labels = labels

		imageName := containerConfig.ImageName
		if containerConfig.ImageTag != "" {
			imageName = imageName + ":" + containerConfig.ImageTag
		}

		moduleType := module.Search("type").Data().(string)

		var url string
		var userName string
		var password string
		if data := module.Search("image").Search("registry").Search("url").Data(); data != nil {
			url = data.(string)
		}
		if data := module.Search("image").Search("registry").Search("userName").Data(); data != nil {
			userName = data.(string)
		}
		if data := module.Search("image").Search("registry").Search("password").Data(); data != nil {
			password = data.(string)
		}
		containerConfig.Registry = RegistryDetails{url, imageName, userName, password}

		envArgs := parseArguments(module.Search("envs").Children())

		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "SERVICE_ID", manifestID))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_NAME", containerConfig.ImageName))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", moduleType))

		containerConfig.EnvArgs = envArgs
		containerConfig.MountConfigs, err = getMounts(module)
		if err != nil {
			return Manifest{}, err
		}

		devices, err := getDevices(module)
		if err != nil {
			return Manifest{}, err
		}
		containerConfig.Resources = container.Resources{Devices: devices}

		containerConfig.ExposedPorts, containerConfig.PortBinding = getPorts(module)
		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := Manifest{
		ID:               manifestID,
		ManifestUniqueID: model.ManifestUniqueID{ManifestName: manifestName, VersionNumber: fmt.Sprint(versionNumber)},
		VersionNumber:    versionNumber,
		Modules:          containerConfigs,
		Labels:           labels,
		Connections:      connections,
	}

	return manifest, nil
}

func GetCommand(jsonParsed *gabs.Container) (string, error) {
	if !jsonParsed.Exists("command") {
		return "", fmt.Errorf("command not found in manifest")
	}

	command := jsonParsed.Search("command").Data().(string)
	return command, nil
}

func GetEdgeAppUniqueID(parsedJson *gabs.Container) model.ManifestUniqueID {
	manifestName := parsedJson.Search("manifestName").Data().(string)
	versionNumber := parsedJson.Search("versionNumber").Data().(float64)

	return model.ManifestUniqueID{ManifestName: manifestName, VersionNumber: fmt.Sprint(versionNumber)}
}

func (m Manifest) UpdateManifest(networkName string) {
	for i, module := range m.Modules {
		m.Modules[i].NetworkName = networkName
		m.Modules[i].ContainerName = makeContainerName(networkName, module.ImageName, module.ImageTag, i)

		m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", m.Modules[i].ContainerName))
	}

	for start, ends := range m.Connections {
		var endpointStrings []string
		for _, end := range ends {
			endpointStrings = append(endpointStrings, fmt.Sprintf("http://%v:80/", m.Modules[end].ContainerName))
		}
		m.Modules[start].EnvArgs = append(m.Modules[start].EnvArgs, fmt.Sprintf("%v=%v", "EGRESS_URLS", strings.Join(endpointStrings, ",")))
	}
}

// makeContainerName is a simple utility to return a standard container name
// This function appends the pipelineID and containerName with _
func makeContainerName(networkName string, imageName string, tag string, index int) string {
	containerName := fmt.Sprint(networkName, ".", imageName, "_", tag, ".", index)

	// create regular expression for all alphanumeric characters and _ . -
	reg, err := regexp.Compile("[^A-Za-z0-9_.-]+")
	if err != nil {
		log.Fatal(err)
	}

	containerName = strings.ReplaceAll(containerName, " ", "")
	containerName = reg.ReplaceAllString(containerName, "_")

	return containerName
}

func parseArguments(options []*gabs.Container) []string {
	log.Debug("Processing environments arguments")

	var args []string
	for _, arg := range options {
		key := arg.Search("key").Data().(string)
		val := arg.Search("value").Data()

		if key != "" {
			args = append(args, fmt.Sprintf("%v=%v", key, val))
		}
	}
	return args
}

func getMounts(parsedJson *gabs.Container) ([]mount.Mount, error) {
	mounts := []mount.Mount{}

	for _, mnt := range parsedJson.Search("mounts").Children() {
		mount := mount.Mount{
			Type:        "bind",
			Source:      mnt.Search("host").Data().(string),
			Target:      mnt.Search("container").Data().(string),
			ReadOnly:    false,
			Consistency: "default",
			BindOptions: &mount.BindOptions{Propagation: "rprivate", NonRecursive: true},
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func getDevices(parsedJson *gabs.Container) ([]container.DeviceMapping, error) {
	devices := []container.DeviceMapping{}

	for _, mnt := range parsedJson.Search("devices").Children() {
		device := container.DeviceMapping{
			PathOnHost:        mnt.Search("host").Data().(string),
			PathInContainer:   mnt.Search("container").Data().(string),
			CgroupPermissions: "rw",
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func getPorts(parsedJson *gabs.Container) (nat.PortSet, nat.PortMap) {
	exposedPorts := nat.PortSet{}
	portBinding := nat.PortMap{}
	for _, port := range parsedJson.Search("ports").Children() {
		hostPort := port.Search("host").Data().(string)
		containerPort := port.Search("container").Data().(string)
		exposedPorts[nat.Port(containerPort)] = struct{}{}
		portBinding[nat.Port(containerPort)] = []nat.PortBinding{{HostPort: hostPort}}
	}

	return exposedPorts, portBinding
}

func getConnections(parsedJson *gabs.Container) (map[int][]int, error) {
	var connectionsStringMap connectionsString
	connectionsIntMap := make(connectionsInt)

	err := json.Unmarshal(parsedJson.Search("connections").Bytes(), &connectionsStringMap)
	if err != nil {
		return nil, err
	}

	for key, values := range connectionsStringMap {
		var valuesInt []int
		for _, value := range values {
			valueInt, err := strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
			valuesInt = append(valuesInt, valueInt)
		}
		keyInt, err := strconv.Atoi(key)
		if err != nil {
			return nil, err
		}
		connectionsIntMap[keyInt] = valuesInt
	}

	return connectionsIntMap, nil
}

func ValidateManifest(jsonParsed *gabs.Container) error {
	var errorList []string

	id := jsonParsed.Search("_id").Data()
	if id == nil || (strings.TrimSpace(id.(string)) == "") {
		errorList = append(errorList, "Please provide manifest id")
	}
	manifestName := jsonParsed.Search("manifestName").Data()
	if manifestName == nil || (strings.TrimSpace(manifestName.(string)) == "") {
		errorList = append(errorList, "Please provide manifestName")
	}
	versionNumber := jsonParsed.Search("versionNumber").Data()
	if versionNumber == nil {
		errorList = append(errorList, "Please provide manifest versionNumber")
	}
	command := jsonParsed.Search("command").Data()
	if command == nil || (strings.TrimSpace(command.(string)) == "") {
		errorList = append(errorList, "Please provide manifest command")
	}
	modules := jsonParsed.Search("modules").Children()
	// Check if manifest contains modules
	if modules == nil || len(modules) < 1 {
		errorList = append(errorList, "Please provide manifest module/s")
	} else {
		for _, module := range modules {
			imageName := module.Search("image").Search("name").Data()
			if imageName == nil || (strings.TrimSpace(imageName.(string)) == "") {
				errorList = append(errorList, "Please provide image name for all modules")
			}

			imageTag := module.Search("image").Search("tag").Data()
			if imageTag == nil {
				errorList = append(errorList, "Please provide image tag for all modules")
			}
		}
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], ","))
	} else {
		return nil
	}
}

func ValidateUniqueIDExist(jsonParsed *gabs.Container) error {
	// Expected JSON: {"manifestName": "Manifest name", "versionNumber": "Manifest version number"}
	var errorList []string
	manifestName := jsonParsed.Search("manifestName").Data()
	if manifestName == nil || (strings.TrimSpace(manifestName.(string)) == "") {
		errorList = append(errorList, "Please provide manifestName")
	}
	versionNumber := jsonParsed.Search("versionNumber").Data()
	if versionNumber == nil {
		errorList = append(errorList, "Please provide manifest versionNumber")
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], " "))
	} else {
		return nil
	}
}
