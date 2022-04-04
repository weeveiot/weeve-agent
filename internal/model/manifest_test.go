package model

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/weeveiot/weeve-agent/testing"
)

var manifestBytesMVP []byte
var filePath string
var errMsg string

func TestMain(m *testing.M) {

	manifestBytesMVP = LoadJsonBytes("pipeline_unit/workingMVP.json")
	code := m.Run()

	os.Exit(code)
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
	json := LoadJsonBytes("pipeline_unit/failInvalidJSON.json")
	_, err := ParseJSONManifest(json)
	if err == nil {
		t.Error("Json parsing should fail")
	}
}

func TestMissingCompose(t *testing.T) {
	filePath = "pipeline_unit/failMissingCompose.json"
	errMsg = "Should throw validation error: Please provide compose"
	ExecuteFailTest(t)
}

func TestMissingNetwork(t *testing.T) {
	filePath = "pipeline_unit/failMissingNetwork.json"
	errMsg = "Should throw validation error: Please provide network details"
	ExecuteFailTest(t)
}

func TestMissingNetworkName(t *testing.T) {
	filePath = "pipeline_unit/failMissingNetworkName.json"
	errMsg = "Should throw validation error: Please provide network name"
	ExecuteFailTest(t)
}

func TestEmptyServices(t *testing.T) {
	filePath = "pipeline_unit/failEmptyServices.json"
	errMsg = "Should throw validation error: Please provide at least one service"
	ExecuteFailTest(t)
}

func TestEmptyServiceModuleId(t *testing.T) {
	filePath = "pipeline_unit/failMissingModuleId.json"
	errMsg = "Should throw validation error: Please provide module id for service"
	ExecuteFailTest(t)
}

func TestEmptyServiceName(t *testing.T) {
	filePath = "pipeline_unit/failMissingServiceName.json"
	errMsg = "Should throw validation error: Please provide name for service"
	ExecuteFailTest(t)
}

func TestMissingImage(t *testing.T) {
	filePath = "pipeline_unit/failMissingImage.json"
	errMsg = "Should throw validation error: Please provide image details"
	ExecuteFailTest(t)
}

func TestMissingImageName(t *testing.T) {
	filePath = "pipeline_unit/failMissingImageName.json"
	errMsg = "Should throw validation error: Please provide image name"
	ExecuteFailTest(t)
}

func TestWorkingManifest(t *testing.T) {
	filePath = "pipeline_unit/workingMVP.json"
	errMsg = "Should not throw any error"
	ExecutePassTest(t)
}

func TestLoad(t *testing.T) {
	fmt.Println("Load the sample manifest")
	var sampleManifestBytesMVP []byte = LoadJsonBytes("manifest/mvp-manifest.json")
	// fmt.Println(sampleManifestBytesMVP)
	manifest, _ := ParseJSONManifest(sampleManifestBytesMVP)
	// fmt.Print(res.ContainerNamesList())
	ContainerConfigs := manifest.GetContainerStart("MVPDataServ_001")
	// fmt.Print(ContainerConfig.MountConfigs)
	fmt.Println("Container details:")
	for i, ContainerConf := range ContainerConfigs {
		fmt.Println(i, ContainerConf)
	}

	fmt.Print(ContainerConfigs[0].MountConfigs)
}
