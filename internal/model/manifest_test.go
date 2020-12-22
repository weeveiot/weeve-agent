package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	// "gitlab.com/weeve/poc-festo/poc-festo-mqtts-ethereum-gateway/internal/parser"

	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)

var manifestBytesMVP []byte

func TestMain(m *testing.M) {

	manifestBytesMVP = LoadJsonBytes("testdata/data-service-compose-NEW.json")
	// manifestBytesSimple = LoadJsonBytes("test_manifest1.json")
	// manifestBytesNoModules = LoadJsonBytes("test_manifest_no_modules.json")
	// manifestBytes3nodesBroker = LoadJsonBytes("test_manifest_3broker.json")
	code := m.Run()

	os.Exit(code)

	// manifest := ParseJSONManifest(manifestBytes)
	// fmt.Println(manifest
}

func LoadJsonBytes(manName string) []byte {
	wd, _ := os.Getwd()
	fmt.Println()
	manifestPath := path.Join(wd, "testdata", manName)
	// fmt.Println("Loading manifest from ", manifestPath)

	var err error = nil
	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}
func TestManifestCreate(t *testing.T) {
	manifest, err := ParseJSONManifest(manifestBytesMVP)
	if err != nil {
		panic(err)
	}
	// fmt.Println("Manifest", manifest.ID, "with", manifest.NumModules, "modules")
	fmt.Println("Manifest", manifest.Name)

	// assert.Equal(t, manifest.ID, "test-manifest-1")
}

func TestEmptyServices(t *testing.T) {
	// Parse the JSON, returns error
	// Assert failure
}

func TestInvalidJSON(t *testing.T) {
	// Parse the JSON, returns error
	// Assert failure
}

func TestMissingImageName(t *testing.T) {
	// Parse the JSON, returns error
	// Assert failure
}

// func TestManifestFailNoModules(t *testing.T) {
// 	_, err := ParseJSONManifest(manifestBytesNoModules)
// 	assert.Error(t, err)
// }

// func TestGetImageNamesList(t *testing.T) {
// 	manifest, err := ParseJSONManifest(manifestBytesSimple)
// 	if err != nil {
// 		panic(err)
// 	}
// 	imgNameList := manifest.ImageNamesList()
// 	for i, img := range imgNameList {
// 		fmt.Println("Image", i, img)
// 	}
// }

// func TestGetContainerNamesList(t *testing.T) {
// 	manifest, err := ParseJSONManifest(manifestBytesSimple)
// 	if err != nil {
// 		panic(err)
// 	}
// 	conNameList := manifest.ContainerNamesList()
// 	for i, img := range conNameList {
// 		fmt.Println("Container", i, img)
// 	}
// }

// func TestGetStartCommands(t *testing.T) {
// 	manifest, err := ParseJSONManifest(manifestBytesSimple)
// 	if err != nil {
// 		panic(err)
// 	}
// 	startCommands := manifest.GetContainerStart()
// 	for i, command := range startCommands {
// 		fmt.Println("Start", i, command)
// 	}
// }

// // The simple -p "1883:1883" in a docker run command
// // Expands to multiple complex objects, basic assertions are done in this unittest
// func TestStartOptionsComplex(t *testing.T) {
// 	manifest, err := ParseJSONManifest(manifestBytes3nodesBroker)
// 	if err != nil {
// 		panic(err)
// 	}
// 	startCommands := manifest.GetContainerStart()
// 	flgMosquitto := false
// 	for _, command := range startCommands {
// 		// fmt.Println("Start", i, command)
// 		// PrintStartCommand(command)
// 		// fmt.Println("Options:", command.Options)
// 		if command.ImageName == "eclipse-mosquitto" {
// 			flgMosquitto = true
// 			assert.Equal(t, nat.PortSet{
// 				nat.Port("1883/tcp"): struct{}{},
// 			}, command.ExposedPorts, "Exposed Ports do not match")
// 			assert.Equal(t,
// 				nat.PortMap{
// 					nat.Port("1883/tcp"): []nat.PortBinding{
// 						{
// 							HostIP: "0.0.0.0",
// 							HostPort: "1883",
// 						},
// 					},
// 				},
// 				command.PortBinding,
// 				"Port binding does not match")
// 		}
// 		if command.ImageName == "weevenetwork/go-mqtt-gobot" {
// 			assert.Equal(t, container.NetworkMode("host"), command.NetworkMode)
// 		}
// 	}
// 	assert.True(t, flgMosquitto, "The manifest MUST include the mosquitto image definition with ports!")
// }
