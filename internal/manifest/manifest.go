package manifest

import (
	"encoding/json"
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
	ID              string
	VersionName     string
	VersionNumber   string
	ApplicationName string
	Modules         []ContainerConfig
	Labels          map[string]string
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
	applicationID := jsonParsed.Search("applicationID").Data().(string)
	versionName := jsonParsed.Search("versionName").Data().(string)
	versionNumber := jsonParsed.Search("versionNumber").Data().(string)
	labels := map[string]string{
		"manifestID":    manifestID,
		"applicationID": applicationID,
		"versionName":   versionName,
		"versionNumber": versionNumber,
	}

	var containerConfigs []ContainerConfig

	for _, module := range jsonParsed.Search("modules").Children() {
		var containerConfig ContainerConfig

		containerConfig.ImageName = module.Search("image").Search("name").Data().(string)
		containerConfig.ImageTag = module.Search("image").Search("tag").Data().(string)
		containerConfig.Labels = labels
		containerConfig.NetworkDriver = jsonParsed.Search("networks").Search("driver").Data().(string)

		imageName := containerConfig.ImageName
		if containerConfig.ImageTag != "" {
			imageName = imageName + ":" + containerConfig.ImageTag
		}
		var userName string
		var password string
		if data := module.Search("registry").Search("userName").Data(); data != nil {
			userName = data.(string)
		}
		if data := module.Search("registry").Search("password").Data(); data != nil {
			password = data.(string)
		}
		containerConfig.Registry = RegistryDetails{imageName, userName, password}

		envJson := module.Search("environments").Children()
		var envArgs = parseArguments(envJson, false)

		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "SERVICE_ID", manifestID))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_NAME", containerConfig.ImageName))
		typesMap := map[string]string{
			"input":   "INGRESS",
			"process": "PROCESS",
			"output":  "EGRESS",
		}
		moduleType := typesMap[module.Search("type").Data().(string)]
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", moduleType))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))

		// since there is no cmd in module's dockerfile, need to move commands to environments
		for _, cmd := range module.Search("commands").Children() {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", cmd.Search("key").Data().(string), cmd.Search("value").Data().(string)))
		}

		containerConfig.EnvArgs = envArgs

		if docData := module.Search("document").Data(); docData != nil {
			document := strings.ReplaceAll(docData.(string), "'", "\"")
			parsedDoc, err := gabs.ParseJSON([]byte(document))
			if err != nil {
				return Manifest{}, err
			}

			containerConfig.MountConfigs, err = getMounts(parsedDoc)
			if err != nil {
				return Manifest{}, err
			}

			exposedPorts, portBinding := getPorts(parsedDoc, envJson)
			containerConfig.ExposedPorts = exposedPorts
			containerConfig.PortBinding = portBinding
		}

		containerConfig.EntryPointArgs = parseArguments(module.Search("commands").Children(), true)

		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := Manifest{
		ID:              manifestID,
		ApplicationName: applicationID,
		VersionName:     versionName,
		VersionNumber:   versionNumber,
		Modules:         containerConfigs,
		Labels:          labels,
	}

	return manifest, nil
}

func GetCommand(jsonParsed *gabs.Container) (string, error) {
	if !jsonParsed.Exists("command") {
		return "", errors.New("command not found in manifest")
	}

	command := jsonParsed.Search("Command").Data().(string)
	return command, nil
}

func (m Manifest) UpdateManifest(networkName string) {
	for i, module := range m.Modules {
		m.Modules[i].NetworkName = networkName
		m.Modules[i].ContainerName = makeContainerName(networkName, module.ImageName, module.ImageTag, i)

		m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", m.Modules[i].ContainerName))

		var prevContainerName = ""
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
	m, ok := parsedJson.Search("mounts").Data().([]interface{})
	if ok && m != nil {
		strMounts, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(strMounts), &mounts)
		log.Info("Mounts:", mounts)
	} else {
		mounts = nil
	}

	return mounts, nil
}

func getPorts(document *gabs.Container, envs []*gabs.Container) (nat.PortSet, nat.PortMap) {
	/* BELOW IS A TEMPORARY SOLUTION TO PORT BINDINGS - NEEDS TO BE REFACTORED */
	// Read which environmental variables are for ports binding
	ports_values_map := document.Search("ports").ChildrenMap()
	if len(ports_values_map) == 0 {
		return nat.PortSet{}, nat.PortMap{}
	}

	hostIPtag := ports_values_map["HostIP"].Data().(string)
	hostPorttag := ports_values_map["HostPort"].Data().(string)

	hostIP := ""
	hostPort := ""
	for _, env := range envs {
		if env.Search("key").Data().(string) == hostIPtag {
			hostIP = env.Search("value").Data().(string)
		}
		if env.Search("key").Data().(string) == hostPorttag {
			hostPort = env.Search("value").Data().(string)
		}
	}

	// Handle Ports Binding
	if hostIP == "" || hostPort == "" {
		log.Error("Failed ports binding - module environments passed in manifest document ports section do not exist.")
	}
	// expose 80/tcp as weeve default port in containers
	exposedPorts := nat.PortSet{
		nat.Port("80/tcp"): struct{}{},
	}

	portBinding := nat.PortMap{
		nat.Port("80/tcp"): []nat.PortBinding{
			{
				HostIP:   hostIP,
				HostPort: hostPort,
			},
		},
	}
	return exposedPorts, portBinding
}

func ValidateManifest(jsonParsed *gabs.Container) error {
	var errorList []string

	id := jsonParsed.Search("id").Data()
	if id == nil {
		errorList = append(errorList, "Please provide data service id")
	}
	version := jsonParsed.Search("version").Data()
	if version == nil {
		errorList = append(errorList, "Please provide data service version")
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
	serviceVersion := jsonParsed.Search("version").Data()
	if serviceVersion == nil {
		errorList = append(errorList, "Expected Data Service Version 'version' in JSON, but not found.")
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], " "))
	} else {
		return nil
	}
}
