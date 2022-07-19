package manifest_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

const mvpManifest = "../../testdata/manifest/mvp-manifest.json"
const sampleManifestBytesMVP = "../../testdata/manifest/test_manifest_3broker.json"

// Utility function to run ValidateManifest tests
func utilTestValidateManifest(filePath string, errMsg error, pass bool) error {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		return err
	}

	err = manifest.ValidateManifest(jsonParsed)
	if (err == nil && !pass) ||
		(err != nil && pass) ||
		(!strings.Contains(err.Error(), errMsg.Error()) && !pass) {
		return fmt.Errorf("Expected error %s, but recieved %s", errMsg, err.Error())
	}

	return nil
}

func TestValidateManifest_EmptyManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failEmptyManifestID.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

// func TestValidateManifest_MissingModules(t *testing.T) {
// 	errMsg := "Modules should not be empty"
// 	jsonParsed, err := parseJson("../../testdata/unittests/failMissingModules.json")
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	err = manifest.ValidateManifest(jsonParsed)
// 	if err == nil {
// 		t.Errorf("Expected error %s, but recieved %s", errMsg, "nil")
// 	} else if !strings.Contains(err.Error(), errMsg) {
// 		t.Errorf("Expected error %s, but recieved %s", errMsg, err.Error())
// 	}
// }

// func TestValidateManifest_EmptyModules(t *testing.T) {
// 	filePath = "../../testdata/unittests/failEmptyModules.json"
// 	errMsg = "Should throw validation error: Please provide at least one service"
// 	ExecuteFailTest(t)
// }

// func TestValidateManifest_MissingModuleId(t *testing.T) {
// 	filePath = "../../testdata/unittests/failMissingModuleId.json"
// 	errMsg = "Should throw validation error: Please provide module id for service"
// 	ExecuteFailTest(t)
// }

// func TestValidateManifest_EmptyModuleId(t *testing.T) {
// 	filePath = "../../testdata/unittests/failEmptyModuleID.json"
// 	errMsg = "Should throw validation error: Please provide module id for service"
// 	ExecuteFailTest(t)
// }

// func TestEmptyServiceName(t *testing.T) {
// 	filePath = "testdata/pipeline_unit/failMissingServiceName.json"
// 	errMsg = "Should throw validation error: Please provide name for service"
// 	ExecuteFailTest(t)
// }

// func TestMissingImage(t *testing.T) {
// 	filePath = "testdata/pipeline_unit/failMissingImage.json"
// 	errMsg = "Should throw validation error: Please provide image details"
// 	ExecuteFailTest(t)
// }

// func TestMissingImageName(t *testing.T) {
// 	filePath = "testdata/pipeline_unit/failMissingImageName.json"
// 	errMsg = "Should throw validation error: Please provide image name"
// 	ExecuteFailTest(t)
// }

// func TestWorkingManifest(t *testing.T) {
// 	filePath = "testdata/pipeline_unit/workingMVP.json"
// 	errMsg = "Should not throw any error"
// 	ExecutePassTest(t)
// }

func TestLoad(t *testing.T) {
	fmt.Println("Load the sample manifest")
	json, err := ioutil.ReadFile(mvpManifest)
	if err != nil {
		t.Error(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	manifest, _ := manifest.GetManifest(jsonParsed)

	ContainerConfigs := manifest.Modules

	fmt.Println("Container details:")
	for i, ContainerConf := range ContainerConfigs {
		fmt.Println(i, ContainerConf)
	}

	fmt.Print(ContainerConfigs[0].MountConfigs)
}

// The simple -p "1883:1883" in a docker run command
// Expands to multiple complex objects, basic assertions are done in this unittest
func TestStartOptionsComplex(t *testing.T) {
	json, err := ioutil.ReadFile(sampleManifestBytesMVP)
	if err != nil {
		t.Error(err)
	}
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	manifest, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		panic(err)
	}
	startCommands := manifest.Modules
	flgMosquitto := false
	for _, command := range startCommands {
		// fmt.Println("Start", i, command)
		// PrintStartCommand(command)
		// fmt.Println("Options:", command.Options)
		if command.ImageName == "eclipse-mosquitto" {
			flgMosquitto = true
			assert.Equal(t, nat.PortSet{
				nat.Port("1883/tcp"): struct{}{},
			}, command.ExposedPorts, "Exposed Ports do not match")
			assert.Equal(t,
				nat.PortMap{
					nat.Port("1883/tcp"): []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "1883",
						},
					},
				},
				command.PortBinding,
				"Port binding does not match")
		}
		// if command.ImageName == "weevenetwork/go-mqtt-gobot" {
		// 	assert.Equal(t, container.NetworkMode("host"), command.NetworkMode)
		// }
	}
	assert.True(t, flgMosquitto, "The manifest MUST include the mosquitto image definition with ports!")
}
