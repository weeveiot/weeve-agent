package manifest_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
	_ "github.com/weeveiot/weeve-agent/testing"
)

var manifestBytesMVP []byte
var filePath string
var errMsg string

func TestMain(m *testing.M) {

	manifestBytesMVP = ioutility.LoadJsonBytes("testdata/pipeline_unit/workingMVP.json")
	code := m.Run()

	os.Exit(code)
}

// Unit function to validate negative tests
func ExecuteFailTest(t *testing.T) {
	json := ioutility.LoadJsonBytes(filePath)
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	err = manifest.ValidateManifest(jsonParsed)
	if err == nil {
		t.Error(errMsg)
	}

	_, err = manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error("Json parsing failed")
	}
}

// Unit function to validate positive tests
func ExecutePassTest(t *testing.T) {
	json := ioutility.LoadJsonBytes(filePath)
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	err = manifest.ValidateManifest(jsonParsed)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error("Json parsing failed")
	}
}

func TestInvalidJson(t *testing.T) {
	json := ioutility.LoadJsonBytes("testdata/pipeline_unit/failInvalidJSON.json")
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = manifest.GetManifest(jsonParsed)
	if err == nil {
		t.Error("Json parsing should fail")
	}
}

func TestMissingCompose(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingCompose.json"
	errMsg = "Should throw validation error: Please provide compose"
	ExecuteFailTest(t)
}

func TestMissingNetwork(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingNetwork.json"
	errMsg = "Should throw validation error: Please provide network details"
	ExecuteFailTest(t)
}

func TestMissingNetworkName(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingNetworkName.json"
	errMsg = "Should throw validation error: Please provide network name"
	ExecuteFailTest(t)
}

func TestEmptyServices(t *testing.T) {
	filePath = "testdata/pipeline_unit/failEmptyServices.json"
	errMsg = "Should throw validation error: Please provide at least one service"
	ExecuteFailTest(t)
}

func TestEmptyServiceModuleId(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingModuleId.json"
	errMsg = "Should throw validation error: Please provide module id for service"
	ExecuteFailTest(t)
}

func TestEmptyServiceName(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingServiceName.json"
	errMsg = "Should throw validation error: Please provide name for service"
	ExecuteFailTest(t)
}

func TestMissingImage(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingImage.json"
	errMsg = "Should throw validation error: Please provide image details"
	ExecuteFailTest(t)
}

func TestMissingImageName(t *testing.T) {
	filePath = "testdata/pipeline_unit/failMissingImageName.json"
	errMsg = "Should throw validation error: Please provide image name"
	ExecuteFailTest(t)
}

func TestWorkingManifest(t *testing.T) {
	filePath = "testdata/pipeline_unit/workingMVP.json"
	errMsg = "Should not throw any error"
	ExecutePassTest(t)
}

func TestLoad(t *testing.T) {
	fmt.Println("Load the sample manifest")
	var sampleManifestBytesMVP []byte = ioutility.LoadJsonBytes("manifest/mvp-manifest.json")
	// fmt.Println(sampleManifestBytesMVP)
	jsonParsed, err := gabs.ParseJSON(sampleManifestBytesMVP)
	if err != nil {
		t.Error(err.Error())
	}
	manifest, _ := manifest.GetManifest(jsonParsed)
	// fmt.Print(res.ContainerNamesList())
	ContainerConfigs := manifest.Modules
	// fmt.Print(ContainerConfig.MountConfigs)
	fmt.Println("Container details:")
	for i, ContainerConf := range ContainerConfigs {
		fmt.Println(i, ContainerConf)
	}

	fmt.Print(ContainerConfigs[0].MountConfigs)
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
// 	startCommands := manifest.Modules
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
// 	startCommands := manifest.Modules
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
