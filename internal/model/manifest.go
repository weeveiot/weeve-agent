package model

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"
)

type Manifest struct {
	data        []byte
	Manifest    gabs.Container
	ID          string
	NetworkName string
	NumModules  int
}

type ContainerConfig struct {
	PipelineName   string
	ContainerName  string
	ImageName      string
	ImageTag       string
	EntryPointArgs []string
	Options        []OptionKeyVal
	NetworkName    string
	ExposedPorts   nat.PortSet // This must be set for the container create
	PortBinding    nat.PortMap // This must be set for the containerStart
	NetworkMode    container.NetworkMode
	NetworkConfig  network.NetworkingConfig
}

type OptionKeyVal struct {
	key string
	val string
}

func PrintStartCommand(sc ContainerConfig) {
	empJSON, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("StartCommand:\n %s\n", string(empJSON))
}

// Create a Manifest type
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
	thisManifest.ID = thisManifest.Manifest.Search("ID").Data().(string)
	thisManifest.NetworkName = thisManifest.Manifest.Search("Name").Data().(string)
	thisManifest.NumModules = thisManifest.CountNumModules()
	if thisManifest.NumModules == 0 {
		msg := "No modules found in manifest"
		log.Error(msg)
		return Manifest{}, errors.New(msg)
	}
	return thisManifest, nil
}

func (m Manifest) CountNumModules() int {
	return len(m.Manifest.Search("Modules").Children())
}

func (m Manifest) ImageNamesList() []string {
	var imageNamesList []string
	for _, mod := range m.Manifest.Search("Modules").Children() {
		imageNamesList = append(imageNamesList, mod.Search("ImageName").Data().(string)+":"+mod.Search("Tag").Data().(string))
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

func (m Manifest) ContainerNamesList() []string {
	var containerNamesList []string
	for _, mod := range m.Manifest.Search("Modules").Children() {
		_ = mod
		containerName := GetContainerName(m.Manifest.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		containerNamesList = append(containerNamesList, containerName)
	}
	return containerNamesList
}

// GetContainerName is a simple utility to return a standard container name
// This function appends the pipelineID and containerName with _
func GetContainerName(pipelineID string, containerName string) string {
	return pipelineID + "_" + containerName
}

// Return a list of container start objects
func (m Manifest) GetContainerStart() []ContainerConfig {
	var startCommands []ContainerConfig
	for _, mod := range m.Manifest.Search("Modules").Children() {
		var thisStartCommand ContainerConfig
		thisStartCommand.PipelineName = m.Manifest.Search("Name").Data().(string)
		thisStartCommand.NetworkName = m.Manifest.Search("Name").Data().(string)
		thisStartCommand.NetworkMode = "" // This is the default setting

		thisStartCommand.ContainerName = GetContainerName(m.Manifest.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		thisStartCommand.ImageName = mod.Search("ImageName").Data().(string)
		thisStartCommand.ImageTag = mod.Search("Tag").Data().(string)

		var theseOptions []OptionKeyVal
		for _, opt := range mod.Search("options").Children() {
			// log.Debug(opt)
			var thisOption OptionKeyVal
			thisOption.key = opt.Search("opt").Data().(string)
			thisOption.val = opt.Search("val").Data().(string)
			theseOptions = append(theseOptions, thisOption)
			// fmt.Println(thisOption)
		}
		thisStartCommand.Options = theseOptions

		var strArgs []string
		for _, arg := range mod.Search("arguments").Children() {
			// strArgs = append(strArgs, "-"+arg.Search("arg").Data().(string)+" "+arg.Search("val").Data().(string))
			strArgs = append(strArgs, arg.Search("arg").Data().(string)+" "+arg.Search("val").Data().(string))
		}

		thisStartCommand.EntryPointArgs = strArgs

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
	}

	return startCommands
}
