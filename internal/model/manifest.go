package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types/container"
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
	Options        []OptionKeyVal
	NetworkName    string
	NetworkDriver  string
	ExposedPorts   nat.PortSet // This must be set for the container create
	PortBinding    nat.PortMap // This must be set for the containerStart
	NetworkMode    container.NetworkMode
	NetworkConfig  network.NetworkingConfig
	Volumes        map[string]struct{}
	MountConfigs   []mount.Mount
	Labels         map[string]string
}

type OptionKeyVal struct {
	key string
	val string
}

type RegistryDetails struct {
	ImageName string
	UserName  string
	Password  string
}

func PrintStartCommand(sc ContainerConfig) {
	empJSON, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("StartCommand:\n %s\n", string(empJSON))
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

func (m Manifest) ImageNamesList() []string {
	var imageNamesList []string
	for _, mod := range m.Manifest.Search("services").Children() {
		imageName := mod.Search("image").Search("name").Data().(string)
		if mod.Search("image").Search("tag").Data() != nil {
			imageName = imageName + ":" + mod.Search("image").Search("tag").Data().(string)
		}

		imageNamesList = append(imageNamesList, imageName)
	}
	return imageNamesList
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

func (m Manifest) PrintManifest() {
	for _, mod := range m.Manifest.Search("Modules").Children() {
		log.Debug(fmt.Sprintf("\t***** index: %v, name: %v", mod.Search("Index").Data(), mod.Search("Name").Data()))
		log.Debug(fmt.Sprintf("\timage %v:%v", mod.Search("ImageName").Data(), mod.Search("Tag").Data()))
		log.Debug("\toptions:")
		for _, opt := range mod.Search("options").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v = %v", opt.Search("opt").Data(), opt.Search("val").Data()))
		}
		log.Debug("\targuments:")
		for _, arg := range mod.Search("arguments").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))
		}
	}
}

func (m Manifest) SpewManifest() {
	spew.Dump(m)
	// spew.Printf("%v", m)
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
func (m Manifest) GetContainerStart(networkName string) []ContainerConfig {
	var startCommands []ContainerConfig
	var prev_container_name = ""

	for index, mod := range m.Manifest.Search("services").Children() {
		var thisStartCommand ContainerConfig

		thisStartCommand.NetworkName = networkName
		thisStartCommand.NetworkMode = "" // This is the default setting
		thisStartCommand.ImageName = mod.Search("image").Search("name").Data().(string)
		thisStartCommand.ImageTag = mod.Search("image").Search("tag").Data().(string)
		thisStartCommand.ContainerName = GetContainerName(networkName, thisStartCommand.ImageName, thisStartCommand.ImageTag, index)
		thisStartCommand.Labels = m.GetLabels()
		thisStartCommand.NetworkDriver = m.Manifest.Search("networks").Search("driver").Data().(string)

		var doc_data = mod.Search("document").Data()
		if doc_data != nil {
			ParseDocumentTag(mod.Search("document").Data(), &thisStartCommand)

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
					thisStartCommand.ExposedPorts = nat.PortSet{
						nat.Port("80/tcp"): struct{}{},
					}

					thisStartCommand.PortBinding = nat.PortMap{
						nat.Port("80/tcp"): []nat.PortBinding{
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
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", thisStartCommand.ContainerName))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))

		// since there is no cmd in module's dockerfile, need to move commands to environments
		for _, cmd := range mod.Search("commands").Children() {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", cmd.Search("key").Data().(string), cmd.Search("value").Data().(string)))
		}

		if index > 0 {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "PREV_CONTAINER_NAME", prev_container_name))

			var next_arg = fmt.Sprintf("%v=%v", "NEXT_CONTAINER_NAME", thisStartCommand.ContainerName)
			startCommands[index-1].EnvArgs = append(startCommands[index-1].EnvArgs, next_arg)

			// following egressing convention 2: http://host:80/
			var temp_arg = fmt.Sprintf("%v=http://%v:80/", "EGRESS_URL", thisStartCommand.ContainerName)
			startCommands[index-1].EnvArgs = append(startCommands[index-1].EnvArgs, temp_arg)
		}
		prev_container_name = thisStartCommand.ContainerName

		for _, thisArg := range envArgs {
			log.Debug(fmt.Sprintf("%v %T", thisArg, thisArg))
		}

		thisStartCommand.EnvArgs = envArgs

		log.Debug("Processing cmd arguments")
		var cmdArgs = ParseArguments(mod.Search("commands").Children(), true)
		thisStartCommand.EntryPointArgs = cmdArgs

		/*
			// Handle the options
			var ExposedPorts string
			for _, option := range thisStartCommand.Options {
				// ExposedPorts is a simple option, just apply it to the struct
				if option.key == "ExposedPorts" {
					ExposedPorts = option.val
					thisStartCommand.ExposedPorts = nat.PortSet{
						nat.Port(option.val): struct{}{},
					}
				}
				// HostIP is always found with HostPort
				// TODO: Refactor!
				if option.key == "HostIP" {
					HostIP := option.val
					HostPort := ""
					for _, subOpt := range thisStartCommand.Options {
						if subOpt.key == "HostPort" {
							HostPort = subOpt.val
						}
					}
					// Make sure HostPort was seen in the options!
					if HostPort == "" {
						// Set default HostPort as in Modules and Intercontainer Communication Spec 1.0.0
						HostPort = "80"
					}

					// Finally, build the PortBindings struct
					thisStartCommand.PortBinding = nat.PortMap{
						nat.Port(ExposedPorts): []nat.PortBinding{
							{
								HostIP:   HostIP,
								HostPort: HostPort,
							},
						},
					}
				}

				if option.key == "network" {
					thisStartCommand.NetworkMode = container.NetworkMode(option.val)
				}

				networkConfig := &network.NetworkingConfig{
					EndpointsConfig: map[string]*network.EndpointSettings{},
				}

				thisStartCommand.NetworkConfig = *networkConfig
			}
		*/

		startCommands = append(startCommands, thisStartCommand)
	}

	return startCommands
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

func ParseDocumentTag(doc_data interface{}, thisStartCommand *ContainerConfig) {
	var vol_maps []map[string]struct{}
	vol_map := make(map[string]struct{})

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
		// log.Info("Mounts: %v", mounts)
		log.Info("Mounts:", mounts)
	} else {
		mounts = nil
	}

	for _, vols := range man_doc.Search("volumes").Children() {
		vol_maps = append(vol_maps, map[string]struct{}{
			vols.Search("container").Data().(string): {},
		})
	}

	if len(vol_maps) >= 0 {
		for _, vol := range vol_maps {
			for k, v := range vol {
				vol_map[k] = v
			}
		}
		thisStartCommand.Volumes = vol_map
		thisStartCommand.MountConfigs = mounts
	}
}

func (man Manifest) GetLabels() map[string]string {
	labels := make(map[string]string)
	labels["manifestID"] = man.Manifest.Search("id").Data().(string)
	labels["version"] = man.Manifest.Search("version").Data().(string)
	labels["name"] = man.Manifest.Search("name").Data().(string)

	return labels
}
