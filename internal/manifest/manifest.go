package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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

type connectionsType map[string][]string

const (
	Connected = "connected"
	Alarm     = "alarm"
	Running   = "running"
	Error     = "error"
	Paused    = "paused"
	Initiated = "initiated"
	Deleted   = "deleted"
)

// uncomment when all changes for v1 modules were done
// const (
// 	ModuleTypeInput      = "Input"
// 	ModuleTypeOutput     = "Output"
// 	ModuleTypeProcessing = "Processing"
// )

// kept for interoperability with pre-v1 modules, delete when the transition to v1 is complete
const (
	ModuleTypeInput      = "INGRESS"
	ModuleTypeOutput     = "EGRESS"
	ModuleTypeProcessing = "PROCESS"
)

func GetManifest(jsonParsed *gabs.Container) (Manifest, error) {
	manifestID := jsonParsed.Search("_id").Data().(string)
	manifestName := jsonParsed.Search("manifestName").Data().(string)
	versionName := jsonParsed.Search("versionName").Data().(string)
	versionNumber := jsonParsed.Search("versionNumber").Data().(float64)
	labels := map[string]string{
		"manifestID":    manifestID,
		"manifestName":  manifestName,
		"versionName":   versionName,
		"versionNumber": fmt.Sprint(versionNumber),
	}

	var containerConfigs []ContainerConfig

	// this map holds the directed connections from ingress towards egress key -> value
	var connections connectionsType

	err := json.Unmarshal(jsonParsed.Search("connections").Bytes(), &connections)
	if err != nil {
		return Manifest{}, err
	}

	// this map holds the reverted directed connections from egress towards ingress key <- value
	revertedConnections := make(connectionsType)

	for key, value := range connections {
		for _, v := range value {
			revertedConnections[v] = append(revertedConnections[v], key)
		}
	}

	modules := jsonParsed.Search("modules").Children()
	for index, module := range modules {
		var containerConfig ContainerConfig

		containerConfig.ImageName = module.Search("image").Search("name").Data().(string)
		containerConfig.ImageTag = module.Search("image").Search("tag").Data().(string)
		containerConfig.Labels = labels

		imageName := containerConfig.ImageName
		if containerConfig.ImageTag != "" {
			imageName = imageName + ":" + containerConfig.ImageTag
		}

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

		if revertedConnections[fmt.Sprint(index+1)] == nil {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", ModuleTypeInput))
		} else if connections[fmt.Sprint(index+1)] == nil {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", ModuleTypeOutput))
		} else {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", ModuleTypeProcessing))
		}

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
		ManifestUniqueID: model.ManifestUniqueID{ManifestName: manifestName, VersionName: versionName},
		VersionNumber:    versionNumber,
		Modules:          containerConfigs,
		Labels:           labels,
	}

	return manifest, nil
}

func GetCommand(jsonParsed *gabs.Container) (string, error) {
	if !jsonParsed.Exists("command") {
		return "", errors.New("command not found in manifest")
	}

	command := jsonParsed.Search("command").Data().(string)
	return command, nil
}

func GetEdgeAppUniqueID(parsedJson *gabs.Container) (model.ManifestUniqueID, error) {
	manifestName := parsedJson.Search("manifestName").Data().(string)
	versionName := parsedJson.Search("versionName").Data().(string)
	if manifestName == "" || versionName == "" {
		return model.ManifestUniqueID{}, errors.New("unique ID fields are missing in given manifest")
	}

	return model.ManifestUniqueID{ManifestName: manifestName, VersionName: versionName}, nil
}

func (m Manifest) UpdateManifest(networkName string) {
	for i, module := range m.Modules {
		m.Modules[i].NetworkName = networkName
		m.Modules[i].ContainerName = makeContainerName(networkName, module.ImageName, module.ImageTag, i)

		m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", m.Modules[i].ContainerName))
		if i > 0 {
			// following egressing convention 2: http://host:80/
			egressUrlArg := fmt.Sprintf("%v=http://%v:80/", "EGRESS_URL", m.Modules[i].ContainerName)
			m.Modules[i-1].EnvArgs = append(m.Modules[i-1].EnvArgs, egressUrlArg)
			if i == len(m.Modules)-1 { // last module is alsways an EGRESS module
				// need to pass anything as EGRESS_URL for module's validation script
				m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "EGRESS_URL", "None"))
			}
		}

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
			CgroupPermissions: "r",
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

func ValidateManifest(jsonParsed *gabs.Container) error {
	var errorList []string

	id := jsonParsed.Search("_id").Data()
	if id == nil {
		errorList = append(errorList, "Please provide manifest id")
	}
	manifestName := jsonParsed.Search("manifestName").Data()
	if manifestName == nil {
		errorList = append(errorList, "Please provide manifest manifestName")
	}
	versionName := jsonParsed.Search("versionName").Data()
	if versionName == nil {
		errorList = append(errorList, "Please provide manifest versionName")
	}
	command := jsonParsed.Search("command").Data()
	if command == nil {
		errorList = append(errorList, "Please provide manifest command")
	}
	modules := jsonParsed.Search("modules").Children()
	// Check if manifest contains services
	if modules == nil || len(modules) < 1 {
		errorList = append(errorList, "Please provide at least one service")
	} else {
		for _, module := range modules {
			moduleID := module.Search("moduleID").Data()
			if moduleID == nil {
				errorList = append(errorList, "Please provide moduleId for all services")
			}
			moduleName := module.Search("moduleName").Data()
			if moduleName == nil {
				errorList = append(errorList, "Please provide module name for all services")
			} else {
				imageName := module.Search("image").Search("name").Data()
				if imageName == nil {
					errorList = append(errorList, "Please provide image name for all services")
				}
				imageTag := module.Search("image").Search("tag").Data()
				if imageTag == nil {
					errorList = append(errorList, "Please provide image tags for all services")
				}
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

	// Expected JSON: {"manifestName": "Manifest name", "versionName": "Manifest version name"}

	var errorList []string
	manifestName := jsonParsed.Search("manifestName").Data()
	if manifestName == nil {
		errorList = append(errorList, "Expected manifest name in JSON, but not found.")
	}
	versionName := jsonParsed.Search("versionName").Data()
	if versionName == nil {
		errorList = append(errorList, "Expected manifest version name in JSON, but not found.")
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], " "))
	} else {
		return nil
	}
}
