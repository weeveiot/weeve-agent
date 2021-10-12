package model

import (
	"encoding/json"
	"fmt"
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

// func (m Manifest) CountNumModules() int {
// 	// t.Logf("%d", i)
// 	// t.Logf("COUNT")
// 	fmt.Println("COUNT")

// 	// return len(m.Manifest.Search("compose")
// 	// return len(m.Manifest.Search("Modules").Children())
// }

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

	containerName = strings.ReplaceAll(containerName, "/", "_")
	containerName = strings.ReplaceAll(containerName, ":", "_")

	return strings.ReplaceAll(containerName, " ", "")
}

// Based on an existing Manifest object, build a new object
// The new object is used to start a container
// The new object has all information required to execute 'docker run':
// 		- Bridge Network information
// 		- Arguments to pass into entrypoint
func (m Manifest) GetContainerStart(networkName string) []ContainerConfig {
	var startCommands []ContainerConfig
	var cntr = 0
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
		}

		//Populate Environment variables
		log.Debug("Processing environments arguments")
		var envArgs = ParseArguments(mod.Search("environments").Children(), false)
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "SERVICE_ID", m.Manifest.Search("id").Data().(string)))

		if cntr > 0 {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "PREV_CONTAINER_NAME", prev_container_name))

			var next_arg = fmt.Sprintf("%v=%v", "NEXT_CONTAINER_NAME", thisStartCommand.ContainerName)
			startCommands[cntr-1].EnvArgs = append(startCommands[cntr-1].EnvArgs, next_arg)

			var temp_arg = fmt.Sprintf("%v=http://%v", "EGRESS_API_HOST", thisStartCommand.ContainerName)
			startCommands[cntr-1].EnvArgs = append(startCommands[cntr-1].EnvArgs, temp_arg)
		}
		prev_container_name = thisStartCommand.ContainerName

		for _, thisArg := range envArgs {
			log.Debug(fmt.Sprintf("%v %T", thisArg, thisArg))
		}

		thisStartCommand.EnvArgs = envArgs

		log.Debug("Processing cmd arguments")
		var cmdArgs = ParseArguments(mod.Search("commands").Children(), true)
		thisStartCommand.EntryPointArgs = cmdArgs

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
					panic("Need to define HostPort in options!")
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

			// Define Network config (why isn't PORT in here...?:
			// https://godoc.org/github.com/docker/docker/api/types/network#NetworkingConfig
			networkConfig := &network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{},
			}

			// gatewayConfig := &network.EndpointSettings{
			// 	Gateway: "gatewayname",
			// }
			// networkConfig.EndpointsConfig["bridge"] = gatewayConfig

			thisStartCommand.NetworkConfig = *networkConfig
		}

		startCommands = append(startCommands, thisStartCommand)
		cntr += 1
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
