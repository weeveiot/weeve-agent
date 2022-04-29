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
	data        []byte
	Manifest    gabs.Container
	Name        string
	NetworkName string
	NumModules  int
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

Additionally,
*/
func ParseJSONManifest(data []byte) (Manifest, error) {
	log.Debug("Parsing data into arbitrary JSON")
	var thisManifest = Manifest{}
	thisManifest.data = data
	jsonParsed, err := gabs.ParseJSON(thisManifest.data)
	if err != nil {
		log.Error(err)
		return Manifest{}, err
	}

	thisManifest.Manifest = *jsonParsed

	return thisManifest, nil
}

func (m Manifest) ImageNamesWithRegList() []RegistryDetails {
	var imageNamesList []RegistryDetails
	for _, mod := range m.Manifest.Search("services").Children() {
		imageName := mod.Search("image").Search("name").Data().(string)
		if mod.Search("image").Search("tag").Data() != nil {
			imageName = imageName + ":" + mod.Search("image").Search("tag").Data().(string)
		}

		var userName string
		var password string
		if mod.Search("registry").Search("userName").Data() != nil && mod.Search("registry").Search("password").Data() != nil {
			userName = mod.Search("registry").Search("userName").Data().(string)
			password = mod.Search("registry").Search("password").Data().(string)
		}

		imageNamesList = append(imageNamesList, RegistryDetails{imageName, userName, password})
	}

	return imageNamesList
}

func (m Manifest) ContainerNamesList(networkName string) []string {
	var containerNamesList []string
	for index, mod := range m.Manifest.Search("services").Children() {
		containerName := "/" + GetContainerName(networkName, mod.Search("image").Search("name").Data().(string), mod.Search("image").Search("tag").Data().(string), index)
		containerNamesList = append(containerNamesList, containerName)
	}
	return containerNamesList
}

// GetContainerName is a simple utility to return a standard container name
// This function appends the pipelineID and containerName with _
func GetContainerName(networkName string, imageName string, tag string, index int) string {
	containerName := fmt.Sprint(networkName, ".", imageName, "_", tag, ".", index)

	// create regular expression for all alphanumeric characters and _ . -
	reg, err := regexp.Compile("[^A-Za-z0-9_.-]+")
	if err != nil {
		log.Error(err)
	}

	containerName = strings.ReplaceAll(containerName, " ", "")
	containerName = reg.ReplaceAllString(containerName, "_")

	return containerName
}

// Based on an existing Manifest object, build a new object
// The new object is used to start a container
// The new object has all information required to execute 'docker run':
// 		- Bridge Network information
// 		- Arguments to pass into entrypoint
func (m Manifest) GetContainerConfig(networkName string) []ContainerConfig {
	const defaultTcpPort = "80/tcp"
	var containerConfigs []ContainerConfig
	var prev_container_name = ""

	for index, mod := range m.Manifest.Search("services").Children() {
		var containerConfig ContainerConfig

		containerConfig.NetworkName = networkName
		containerConfig.ImageName = mod.Search("image").Search("name").Data().(string)
		containerConfig.ImageTag = mod.Search("image").Search("tag").Data().(string)
		containerConfig.ContainerName = GetContainerName(networkName, containerConfig.ImageName, containerConfig.ImageTag, index)
		containerConfig.Labels = m.GetLabels()
		containerConfig.NetworkDriver = m.Manifest.Search("networks").Search("driver").Data().(string)

		var doc_data = mod.Search("document").Data()
		if doc_data != nil {
			ParseDocumentTag(mod.Search("document").Data(), &containerConfig)

			/* BELOW IS A TEMPORARY SOLUTION TO PORT BINDINGS - NEEDS TO BE REFACTORED */
			// Read which environmental variables are for ports binding
			var document = doc_data.(string)
			document = strings.ReplaceAll(document, "'", "\"")
			man_doc, err := gabs.ParseJSON([]byte(document))
			if err != nil {
				log.Error("Error on parsing document tag ", err)
			}
			ports_values_map := man_doc.Search("ports").ChildrenMap()
			if len(ports_values_map) != 0 {
				// Set placeholders for ports binding values HostIP and HostPort
				var hostIP = ""
				var hostPort = ""
				hostIP_tag := ports_values_map["HostIP"].Data().(string)
				hostPort_tag := ports_values_map["HostPort"].Data().(string)

				for _, env := range mod.Search("environments").Children() {
					if env.Search("key").Data().(string) == hostIP_tag {
						hostIP = env.Search("value").Data().(string)
					}
					if env.Search("key").Data().(string) == hostPort_tag {
						hostPort = env.Search("value").Data().(string)
					}
				}

				// Handle Ports Binding
				if hostIP != "" && hostPort != "" {
					// expose 80/tcp as weeve default port in containers
					containerConfig.ExposedPorts = nat.PortSet{
						nat.Port(defaultTcpPort): struct{}{},
					}

					containerConfig.PortBinding = nat.PortMap{
						nat.Port(defaultTcpPort): []nat.PortBinding{
							{
								HostIP:   hostIP,
								HostPort: hostPort,
							},
						},
					}
				} else {
					log.Error("Failed ports binding - module environments passed in manifest document ports section do not exist.")
				}
			}
		}

		//Populate Environment variables
		log.Debug("Processing environments arguments")
		var envArgs = ParseArguments(mod.Search("environments").Children(), false)

		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "SERVICE_ID", m.Manifest.Search("id").Data().(string)))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_NAME", mod.Search("name").Data().(string)))
		types_mapping := map[string]string{"input": "INGRESS", "process": "PROCESS", "output": "EGRESS"}
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", types_mapping[mod.Search("type").Data().(string)]))
		if mod.Search("type").Data().(string) == "output" {
			// need to pass anything as EGRESS_URL for module's validation script
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "EGRESS_URL", "None"))
		}
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", containerConfig.ContainerName))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))

		// since there is no cmd in module's dockerfile, need to move commands to environments
		for _, cmd := range mod.Search("commands").Children() {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", cmd.Search("key").Data().(string), cmd.Search("value").Data().(string)))
		}

		if index > 0 {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "PREV_CONTAINER_NAME", prev_container_name))

			var next_arg = fmt.Sprintf("%v=%v", "NEXT_CONTAINER_NAME", containerConfig.ContainerName)
			containerConfigs[index-1].EnvArgs = append(containerConfigs[index-1].EnvArgs, next_arg)

			// following egressing convention 2: http://host:80/
			var temp_arg = fmt.Sprintf("%v=http://%v:80/", "EGRESS_URL", containerConfig.ContainerName)
			containerConfigs[index-1].EnvArgs = append(containerConfigs[index-1].EnvArgs, temp_arg)
		}
		prev_container_name = containerConfig.ContainerName

		for _, thisArg := range envArgs {
			log.Debug(fmt.Sprintf("%v %T", thisArg, thisArg))
		}

		containerConfig.EnvArgs = envArgs

		log.Debug("Processing cmd arguments")
		var cmdArgs = ParseArguments(mod.Search("commands").Children(), true)
		containerConfig.EntryPointArgs = cmdArgs

		containerConfigs = append(containerConfigs, containerConfig)
	}

	return containerConfigs
}

func ParseArguments(options []*gabs.Container, cmdArgs bool) []string {
	var args []string
	for _, arg := range options {
		var key = ""
		var val = ""
		if arg.Search("key").Data().(string) != "" {
			key = arg.Search("key").Data().(string)
		}

		if arg.Search("value").Data().(string) != "" {
			val = arg.Search("value").Data().(string)
		}

		if key != "" && val != "" {
			if cmdArgs {
				args = append(args, fmt.Sprintf("--%v", key))
				args = append(args, fmt.Sprintf("%v", val))
			} else {
				args = append(args, fmt.Sprintf("%v=%v", key, val))
			}

		}
	}
	return args
}

func ParseDocumentTag(doc_data interface{}, containerConfig *ContainerConfig) {
	var document = doc_data.(string)
	document = strings.ReplaceAll(document, "'", "\"")

	man_doc, err := gabs.ParseJSON([]byte(document))
	if err != nil {
		log.Error("Error on parsing document tag ", err)
		return
	}

	log.Info("man_doc ", document, man_doc)

	mounts := []mount.Mount{}
	m, ok := man_doc.Search("mounts").Data().([]interface{})
	if ok && m != nil {
		strMounts, err := json.Marshal(m)
		if err != nil {
			log.Error("Error on parsing mounts ", err)
			return
		}
		json.Unmarshal([]byte(strMounts), &mounts)
		log.Info("Mounts:", mounts)
	} else {
		mounts = nil
	}

	containerConfig.MountConfigs = mounts
}

func (man Manifest) GetLabels() map[string]string {
	labels := make(map[string]string)
	labels["manifestID"] = man.Manifest.Search("id").Data().(string)
	labels["version"] = man.Manifest.Search("version").Data().(string)
	labels["name"] = man.Manifest.Search("name").Data().(string)

	return labels
}

// ValidateManifest function to validate manifest JSON
func ValidateManifest(m Manifest) error {
	var errorList []string

	id := m.Manifest.Search("id").Data()
	if id == nil {
		errorList = append(errorList, "Please provide data service id")
	}
	version := m.Manifest.Search("version").Data()
	if version == nil {
		errorList = append(errorList, "Please provide data service version")
	}
	name := m.Manifest.Search("name").Data()
	if name == nil {
		errorList = append(errorList, "Please provide data service name")
	}
	services := m.Manifest.Search("services").Children()
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
	network := m.Manifest.Search("networks").Data()
	if network == nil {
		errorList = append(errorList, "Please provide data service network")
	} else {
		networkName := m.Manifest.Search("networks").Search("driver").Data()
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
