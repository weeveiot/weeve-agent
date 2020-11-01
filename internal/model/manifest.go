package model

import (
	"fmt"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

type Manifest struct {
	data []byte;
	Manifest gabs.Container;
	ID string;
}

type StartCommand struct {
	containerName string;
	imageName string
	imageTag string;
	entryPointArgs []string;
}

// Create a Manifest type
func ParseJSONManifest(data []byte) Manifest {
	log.Debug("Parsing data into arbitrary JSON")
	var thisManifest = Manifest{}
	thisManifest.data = data
	jsonParsed, err := gabs.ParseJSON(thisManifest.data )
	if err != nil {
		panic(err)
	}

	thisManifest.Manifest = *jsonParsed
	thisManifest.ID = thisManifest.Manifest.Search("ID").Data().(string)
	return thisManifest
}

func (m Manifest) ImageNamesList()  []string {
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

func (m Manifest) GetContainerStart() []StartCommand {
	var startCommands []StartCommand
	for _, mod := range m.Manifest.Search("Modules").Children() {
		var thisStartCommand StartCommand;

		thisStartCommand.containerName = GetContainerName(m.Manifest.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		thisStartCommand.imageName = mod.Search("ImageName").Data().(string)
		thisStartCommand.imageTag = mod.Search("Tag").Data().(string)

		var strArgs []string
		for _, arg := range mod.Search("arguments").Children() {
			strArgs = append(strArgs, "--"+arg.Search("arg").Data().(string)+"="+arg.Search("val").Data().(string))
		}

		thisStartCommand.entryPointArgs = strArgs
		startCommands = append(startCommands, thisStartCommand)
	}
	return startCommands
}