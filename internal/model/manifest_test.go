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
var filePath string
var errMsg string

func TestMain(m *testing.M) {

	manifestBytesMVP = LoadJsonBytes("newFormat020/workingMVP.json")
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

// Unit function to validate negative tests
func ExecuteFailTest(t *testing.T) {
	json := LoadJsonBytes(filePath)
	m, err := ParseJSONManifest(json)
	if err != nil {
		t.Error("Json parsing failed")
	}

	err = ValidateManifest(m)
	if err == nil {
		t.Error(err.Error())
		t.Error(errMsg)
	}
}

// Unit function to validate positive tests
func ExecutePassTest(t *testing.T) {
	json := LoadJsonBytes(filePath)
	m, err := ParseJSONManifest(json)
	if err != nil {
		t.Error("Json parsing failed")
	}

	err = ValidateManifest(m)
	if err != nil {
		t.Error(err.Error())
		t.Error(errMsg)
	}
}

func TestInvalidJson(t *testing.T) {
	json := LoadJsonBytes("newFormat020/failInvalidJSON.json")
	_, err := ParseJSONManifest(json)
	if err == nil {
		t.Error("Json parsing should fail")
	}
}

func TestEmptyServices(t *testing.T) {
	filePath = "newFormat020/failEmptyServices.json"
	errMsg = "Should throw validation error: Please provide at least one service"
	ExecuteFailTest(t)
}

func TestMissingImageName(t *testing.T) {
	filePath = "newFormat020/failMissingImageName.json"
	errMsg = "Should throw validation error: Please provide image name"
	ExecuteFailTest(t)
}

func TestWorkingManifest(t *testing.T) {
	filePath = "newFormat020/workingMVP.json"
	errMsg = "Should not throw any error"
	ExecutePassTest(t)
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
