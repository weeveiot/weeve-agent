package manifest_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

const mvpManifest = "../../testdata/manifest/mvp-manifest.json"

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

func TestValidateManifest_MissingManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failMissingManifestID.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failEmptyManifestID.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestName(t *testing.T) {
	errMsg := "Please provide manifest manifestName"
	filePath := "../../testdata/unittests/failMissingManifestName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestName(t *testing.T) {
	errMsg := "Please provide manifest manifestName"
	filePath := "../../testdata/unittests/failEmptyManifestName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestVersionName(t *testing.T) {
	errMsg := "Please provide manifest versionName"
	filePath := "../../testdata/unittests/failMissingManifestVersionName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestVersionName(t *testing.T) {
	errMsg := "Please provide manifest versionName"
	filePath := "../../testdata/unittests/failEmptyManifestVersionName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestVersionNumber(t *testing.T) {
	errMsg := "Please provide manifest versionNumber"
	filePath := "../../testdata/unittests/failMissingManifestVersionNumber.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failMissingManifestCommand.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failEmptyManifestCommand.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failMissingManifestModules.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failEmptyManifestModules.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failMissingManifestImageName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failEmptyManifestImageName.json"
	err := utilTestValidateManifest(filePath, fmt.Errorf(errMsg), false)
	if err != nil {
		t.Error(err)
	}
}

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
