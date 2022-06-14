package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"
)

type Manifest struct {
	ID            string
	VersionName   string
	VersionNumber float64
	ManifestName  string
	Modules       []ContainerConfig
	Labels        map[string]string
}

type ManifestUniqueID struct {
	VersionName  string
	ManifestName string
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
}

type RegistryDetails struct {
	Url       string
	ImageName string
	UserName  string
	Password  string
}

// Create a Manifest type
/* The manifest type holds the parsed JSON of a manifest file, as well as
several convenience attributes.

The manifest JSON object itself is parsed into a golang 'gabs' object.
(see https://github.com/Jeffail/gabs)
*/
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

	for _, module := range jsonParsed.Search("modules").Children() {
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

		envJson := module.Search("envs").Children()
		var envArgs = parseArguments(envJson, false)

		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "SERVICE_ID", manifestID))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_NAME", containerConfig.ImageName))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))

		containerConfig.EnvArgs = envArgs
		var err error
		containerConfig.MountConfigs, err = getMounts(module)
		if err != nil {
			return Manifest{}, err
		}

		exposedPorts, portBinding := getPorts(module, envJson)
		containerConfig.ExposedPorts = exposedPorts
		containerConfig.PortBinding = portBinding

		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := Manifest{
		ID:            manifestID,
		ManifestName:  manifestName,
		VersionName:   versionName,
		VersionNumber: versionNumber,
		Modules:       containerConfigs,
		Labels:        labels,
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

func GetEdgeAppUniqueID(parsedJson *gabs.Container) (ManifestUniqueID, error) {
	manifestName := parsedJson.Search("manifestName").Data().(string)
	versionName := parsedJson.Search("versionName").Data().(string)
	if manifestName == "" || versionName == "" {
		return ManifestUniqueID{}, errors.New("unique ID fields are missing in given manifest")
	}

	return ManifestUniqueID{ManifestName: manifestName, VersionName: versionName}, nil
}

func (m Manifest) UpdateManifest(networkName string) {
	var prevContainerName = ""
	for i, module := range m.Modules {
		m.Modules[i].NetworkName = networkName
		m.Modules[i].ContainerName = makeContainerName(networkName, module.ImageName, module.ImageTag, i)

		m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", m.Modules[i].ContainerName))
		if i > 0 {
			m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "PREV_CONTAINER_NAME", prevContainerName))

			nextContainerNameArg := fmt.Sprintf("%v=%v", "NEXT_CONTAINER_NAME", m.Modules[i].ContainerName)
			m.Modules[i-1].EnvArgs = append(m.Modules[i-1].EnvArgs, nextContainerNameArg)

			// following egressing convention 2: http://host:80/
			egressUrlArg := fmt.Sprintf("%v=http://%v:80/", "EGRESS_URL", m.Modules[i].ContainerName)
			m.Modules[i-1].EnvArgs = append(m.Modules[i-1].EnvArgs, egressUrlArg)
			if i == len(m.Modules)-1 { // last module is alsways an EGRESS module
				// need to pass anything as EGRESS_URL for module's validation script
				m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "EGRESS_URL", "None"))
			}
		}
		prevContainerName = m.Modules[i].ContainerName

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

func parseArguments(options []*gabs.Container, cmdArgs bool) []string {
	if cmdArgs {
		log.Debug("Processing CLI arguments")
	} else {
		log.Debug("Processing environments arguments")
	}
	var args []string
	for _, arg := range options {
		key := arg.Search("key").Data().(string)
		val := arg.Search("value").Data().(string)

		if key != "" && val != "" {
			if cmdArgs { // CLI arguments
				args = append(args, fmt.Sprintf("--%v", key))
				args = append(args, fmt.Sprintf("%v", val))
			} else { // env varialbes
				args = append(args, fmt.Sprintf("%v=%v", key, val))
			}
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
			ReadOnly:    true,
			Consistency: "default",
			BindOptions: &mount.BindOptions{Propagation: "rprivate", NonRecursive: true},
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func getPorts(document *gabs.Container, envs []*gabs.Container) (nat.PortSet, nat.PortMap) {
	binding := []nat.PortBinding{}
	for _, port := range document.Search("ports").Children() {
		hostPort := port.Search("host").Data().(string)
		binding = append(binding, nat.PortBinding{HostPort: hostPort})
	}

	portBinding := nat.PortMap{nat.Port("80/tcp"): binding}
	exposedPorts := nat.PortSet{
		nat.Port("80/tcp"): struct{}{},
	}

	return exposedPorts, portBinding
}

func ValidateManifest(jsonParsed *gabs.Container) error {
	var errorList []string

	id := jsonParsed.Search("id").Data()
	if id == nil {
		errorList = append(errorList, "Please provide data service id")
	}
	versionName := jsonParsed.Search("versionName").Data()
	if versionName == nil {
		errorList = append(errorList, "Please provide data service versionName")
	}
	name := jsonParsed.Search("name").Data()
	if name == nil {
		errorList = append(errorList, "Please provide data service name")
	}
	services := jsonParsed.Search("services").Children()
	// Check if manifest contains services
	if services == nil || len(services) < 1 {
		errorList = append(errorList, "Please provide at least one service")
	} else {
		for _, srv := range services {
			moduleID := srv.Search("id").Data()
			if moduleID == nil {
				errorList = append(errorList, "Please provide moduleId for all services")
			}
			serviceName := srv.Search("name").Data()
			if serviceName == nil {
				errorList = append(errorList, "Please provide service name for all services")
			} else {
				imageName := srv.Search("image").Search("name").Data()
				if imageName == nil {
					errorList = append(errorList, "Please provide image name for all services")
				}
				imageTag := srv.Search("image").Search("tag").Data()
				if imageTag == nil {
					errorList = append(errorList, "Please provide image tags for all services")
				}
			}
		}
	}
	network := jsonParsed.Search("networks").Data()
	if network == nil {
		errorList = append(errorList, "Please provide data service network")
	} else {
		networkName := jsonParsed.Search("networks").Search("driver").Data()
		if networkName == nil {
			errorList = append(errorList, "Please provide data service network driver")
		}
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], ","))
	} else {
		return nil
	}
}

func ValidateStartStopJSON(jsonParsed *gabs.Container) error {

	// Expected JSON: {"id": dataServiceID, "version": dataServiceVesion}

	var errorList []string
	serviceID := jsonParsed.Search("id").Data()
	if serviceID == nil {
		errorList = append(errorList, "Expected Data Service ID 'id' in JSON, but not found.")
	}
	versionName := jsonParsed.Search("versionName").Data()
	if versionName == nil {
		errorList = append(errorList, "Expected Data Service VersionName in JSON, but not found.")
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], " "))
	} else {
		return nil
	}
}
