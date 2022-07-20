package manifest_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

var manifestUniqueID struct {
	ManifestName  string  `json:"manifestName"`
	VersionNumber float64 `json:"versionNumber"`
}

func TestGetManifest(t *testing.T) {
	json, err := ioutil.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Error(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	manifest, _ := manifest.GetManifest(jsonParsed)

	assert.NotNil(t, manifest)
	assert.Equal(t, "kunbus-demo-manifest", manifest.ManifestUniqueID.ManifestName)
	assert.Equal(t, float64(1), manifest.VersionNumber)
	assert.Equal(t, 3, len(manifest.Connections))
	assert.Equal(t, 4, len(manifest.Modules))

	if len(manifest.Modules) == 4 {
		assert.Equal(t, 4, len(manifest.Modules[0].Labels))
		assert.Equal(t, "weevenetwork/mqtt-ingress", manifest.Modules[0].ImageName)
		assert.Equal(t, "V1", manifest.Modules[0].ImageTag)
		assert.Equal(t, 10, len(manifest.Modules[0].EnvArgs))
		if (len(manifest.Modules[0].EnvArgs)) == 10 {
			assert.Equal(t, "MQTT_BROKER=mqtt://mapi-dev.weeve.engineering", manifest.Modules[0].EnvArgs[0])
			assert.Equal(t, "PORT=1883", manifest.Modules[0].EnvArgs[1])
			assert.Equal(t, "PROTOCOL=mqtt", manifest.Modules[0].EnvArgs[2])
			assert.Equal(t, "TOPIC=revpi_I14", manifest.Modules[0].EnvArgs[3])
			assert.Equal(t, "QOS=0", manifest.Modules[0].EnvArgs[4])
			assert.Equal(t, "SERVICE_ID=62bef68d664ed72f8ecdd690", manifest.Modules[0].EnvArgs[5])
			assert.Equal(t, "MODULE_NAME=weevenetwork/mqtt-ingress", manifest.Modules[0].EnvArgs[6])
			assert.Equal(t, "INGRESS_PORT=80", manifest.Modules[0].EnvArgs[7])
			assert.Equal(t, "INGRESS_PATH=/", manifest.Modules[0].EnvArgs[8])
			assert.Equal(t, "MODULE_TYPE=Input", manifest.Modules[0].EnvArgs[9])
		}

		assert.Equal(t, struct{}{}, manifest.Modules[0].ExposedPorts[nat.Port("1883")])
		assert.Equal(t, []nat.PortBinding{{HostPort: "1883"}}, manifest.Modules[0].PortBinding[nat.Port("1883")])

		assert.Equal(t, 1, len(manifest.Modules[0].MountConfigs))
		if (len(manifest.Modules[0].MountConfigs)) == 1 {
			assert.Equal(t,
				mount.Mount{Type: "bind",
					Source:      "/data/host",
					Target:      "/data",
					ReadOnly:    false,
					Consistency: "default",
					BindOptions: &mount.BindOptions{Propagation: "rprivate", NonRecursive: true}},
				manifest.Modules[0].MountConfigs[0])
		}

		assert.Equal(t, 1, len(manifest.Modules[0].Resources.Devices))
		if (len(manifest.Modules[0].MountConfigs)) == 1 {
			assert.Equal(t,
				container.DeviceMapping{
					PathOnHost:        "/dev/ttyUSB0/host",
					PathInContainer:   "/dev/ttyUSB0",
					CgroupPermissions: "r",
				},
				manifest.Modules[0].Resources.Devices[0])
		}
	}
}

func TestGetEdgeAppUniqueID(t *testing.T) {
	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.VersionNumber = 1

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	man := manifest.GetEdgeAppUniqueID(jsonParsed)
	assert.Equal(t, man.ManifestName, manifestUniqueID.ManifestName)
	assert.Equal(t, man.VersionNumber, fmt.Sprintf("%g", manifestUniqueID.VersionNumber))
}

func TestGetCommand_MissingCommand(t *testing.T) {
	errMsg := "command not found in manifest"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	cmd, err := manifest.GetCommand(jsonParsed)
	if err == nil {
		t.Errorf("Expected %s, but got: %s", errMsg, err)
	} else {
		assert.Equal(t, errMsg, err.Error())
		assert.Equal(t, "", cmd)
	}
}

func TestGetCommand(t *testing.T) {
	var commandJson struct {
		Command string `json:"command"`
	}
	commandJson.Command = "DEPLOY"

	json, err := json.Marshal(commandJson)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	cmd, err := manifest.GetCommand(jsonParsed)
	if err != nil {
		t.Error(err.Error())
	} else {
		assert.Equal(t, "DEPLOY", cmd)
	}
}

// Utility function to run ValidateManifest fail tests
func utilFailTestValidateManifest(filePath string, errMsg error) error {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		return err
	}

	err = manifest.ValidateManifest(jsonParsed)
	if err == nil || err.Error() != errMsg.Error() {
		return fmt.Errorf("Expected %s, but got: %s", errMsg, err)
	}

	return nil
}

func TestValidateManifest_MissingManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failMissingManifestID.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failEmptyManifestID.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestName(t *testing.T) {
	errMsg := "Please provide manifestName"
	filePath := "../../testdata/unittests/failMissingManifestName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestName(t *testing.T) {
	errMsg := "Please provide manifestName"
	filePath := "../../testdata/unittests/failEmptyManifestName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestVersionName(t *testing.T) {
	errMsg := "Please provide manifest versionName"
	filePath := "../../testdata/unittests/failMissingManifestVersionName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestVersionName(t *testing.T) {
	errMsg := "Please provide manifest versionName"
	filePath := "../../testdata/unittests/failEmptyManifestVersionName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestVersionNumber(t *testing.T) {
	errMsg := "Please provide manifest versionNumber"
	filePath := "../../testdata/unittests/failMissingManifestVersionNumber.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failMissingManifestCommand.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failEmptyManifestCommand.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failMissingManifestModules.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failEmptyManifestModules.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_MissingManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failMissingManifestImageName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest_EmptyManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failEmptyManifestImageName.json"
	err := utilFailTestValidateManifest(filePath, fmt.Errorf(errMsg))
	if err != nil {
		t.Error(err)
	}
}

func TestValidateManifest(t *testing.T) {
	json, err := ioutil.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Error(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	err = manifest.ValidateManifest(jsonParsed)
	assert.Nil(t, err)
}

func TestValidateUniqueIDExist_EmptyManifestName(t *testing.T) {
	manifestUniqueID.ManifestName = " "
	manifestUniqueID.VersionNumber = 1
	errMsg := "Please provide manifestName"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	err = manifest.ValidateUniqueIDExist(jsonParsed)
	if err == nil {
		t.Errorf("Expected %s, but got: %s", errMsg, err)
	} else {
		assert.Equal(t, errMsg, err.Error())
	}
}

func TestValidateUniqueIDExist(t *testing.T) {
	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.VersionNumber = 1

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}

	err = manifest.ValidateUniqueIDExist(jsonParsed)
	assert.Nil(t, err)
}
