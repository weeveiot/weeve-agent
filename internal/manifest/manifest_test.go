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
	assert := assert.New(t)

	json, err := ioutil.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}
	manifest, _ := manifest.GetManifest(jsonParsed)

	assert.NotNil(manifest)
	assert.Equal("kunbus-demo-manifest", manifest.ManifestUniqueID.ManifestName)
	assert.Equal(float64(1), manifest.VersionNumber)
	assert.Equal(3, len(manifest.Connections))
	assert.Equal(4, len(manifest.Modules))

	if len(manifest.Modules) == 4 {
		assert.Equal(4, len(manifest.Modules[0].Labels))
		assert.Equal("weevenetwork/mqtt-ingress", manifest.Modules[0].ImageName)
		assert.Equal("V1", manifest.Modules[0].ImageTag)
		assert.Equal(10, len(manifest.Modules[0].EnvArgs))
		if (len(manifest.Modules[0].EnvArgs)) == 10 {
			assert.Equal("MQTT_BROKER=mqtt://mapi-dev.weeve.engineering", manifest.Modules[0].EnvArgs[0])
			assert.Equal("PORT=1883", manifest.Modules[0].EnvArgs[1])
			assert.Equal("PROTOCOL=mqtt", manifest.Modules[0].EnvArgs[2])
			assert.Equal("TOPIC=revpi_I14", manifest.Modules[0].EnvArgs[3])
			assert.Equal("QOS=0", manifest.Modules[0].EnvArgs[4])
			assert.Equal("SERVICE_ID=62bef68d664ed72f8ecdd690", manifest.Modules[0].EnvArgs[5])
			assert.Equal("MODULE_NAME=weevenetwork/mqtt-ingress", manifest.Modules[0].EnvArgs[6])
			assert.Equal("INGRESS_PORT=80", manifest.Modules[0].EnvArgs[7])
			assert.Equal("INGRESS_PATH=/", manifest.Modules[0].EnvArgs[8])
			assert.Equal("MODULE_TYPE=Input", manifest.Modules[0].EnvArgs[9])
		}

		assert.Equal(struct{}{}, manifest.Modules[0].ExposedPorts[nat.Port("1883")])
		assert.Equal([]nat.PortBinding{{HostPort: "1883"}}, manifest.Modules[0].PortBinding[nat.Port("1883")])

		assert.Equal(1, len(manifest.Modules[0].MountConfigs))
		if (len(manifest.Modules[0].MountConfigs)) == 1 {
			assert.Equal(mount.Mount{Type: "bind",
				Source:      "/data/host",
				Target:      "/data",
				ReadOnly:    false,
				Consistency: "default",
				BindOptions: &mount.BindOptions{Propagation: "rprivate", NonRecursive: true}},
				manifest.Modules[0].MountConfigs[0])
		}

		assert.Equal(1, len(manifest.Modules[0].Resources.Devices))
		if (len(manifest.Modules[0].MountConfigs)) == 1 {
			assert.Equal(container.DeviceMapping{
				PathOnHost:        "/dev/ttyUSB0/host",
				PathInContainer:   "/dev/ttyUSB0",
				CgroupPermissions: "rw",
			},
				manifest.Modules[0].Resources.Devices[0])
		}

		manifest.UpdateManifest("kunbus-demo-manifest_1d")
		assert.Equal(12, len(manifest.Modules[0].EnvArgs))
		if (len(manifest.Modules[0].EnvArgs)) == 12 {
			assert.Equal("INGRESS_HOST=kunbus-demo-manifest_1d.weevenetwork_mqtt-ingress_V1.0", manifest.Modules[0].EnvArgs[10])
			assert.Equal("EGRESS_URLS=http://kunbus-demo-manifest_1d.weevenetwork_fluctuation-filter_V1.1:80/", manifest.Modules[0].EnvArgs[11])
		}
	}
}

func TestGetEdgeAppUniqueID(t *testing.T) {
	assert := assert.New(t)

	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.VersionNumber = 1

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	man := manifest.GetEdgeAppUniqueID(jsonParsed)
	assert.Equal(manifestUniqueID.ManifestName, man.ManifestName)
	assert.Equal(fmt.Sprintf("%g", manifestUniqueID.VersionNumber), man.VersionNumber)
}

func TestGetCommand_MissingCommand(t *testing.T) {
	assert := assert.New(t)
	errMsg := "command not found in manifest"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := manifest.GetCommand(jsonParsed)
	assert.NotNil(err)
	if err != nil {
		assert.Equal(errMsg, err.Error())
		assert.Equal("", cmd)
	}
}

func TestGetCommand(t *testing.T) {
	assert := assert.New(t)
	var commandJson struct {
		Command string `json:"command"`
	}
	commandJson.Command = "DEPLOY"

	json, err := json.Marshal(commandJson)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := manifest.GetCommand(jsonParsed)
	assert.Nil(err)
	if err == nil {
		assert.Equal(commandJson.Command, cmd)
	}
}

// Utility function to run ValidateManifest fail tests
func utilFailTestValidateManifest(filePath string, errMsg string) error {
	json, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		return err
	}

	err = manifest.ValidateManifest(jsonParsed)
	if err == nil {
		return fmt.Errorf("Expected %s, but got: %s", errMsg, err)
	} else if err.Error() != errMsg {
		return fmt.Errorf("Expected %s, but got: %s", errMsg, err)
	}

	return nil
}

func TestValidateManifest_MissingManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failMissingManifestID.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_EmptyManifestID(t *testing.T) {
	errMsg := "Please provide manifest id"
	filePath := "../../testdata/unittests/failEmptyManifestID.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestName(t *testing.T) {
	errMsg := "Please provide manifestName"
	filePath := "../../testdata/unittests/failMissingManifestName.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_EmptyManifestName(t *testing.T) {
	errMsg := "Please provide manifestName"
	filePath := "../../testdata/unittests/failEmptyManifestName.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestVersionNumber(t *testing.T) {
	errMsg := "Please provide manifest versionNumber"
	filePath := "../../testdata/unittests/failMissingManifestVersionNumber.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failMissingManifestCommand.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_EmptyManifestCommand(t *testing.T) {
	errMsg := "Please provide manifest command"
	filePath := "../../testdata/unittests/failEmptyManifestCommand.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failMissingManifestModules.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_EmptyManifestModules(t *testing.T) {
	errMsg := "Please provide manifest module/s"
	filePath := "../../testdata/unittests/failEmptyManifestModules.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failMissingManifestImageName.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_MissingManifestImageTag(t *testing.T) {
	errMsg := "Please provide image tag for all modules"
	filePath := "../../testdata/unittests/failMissingManifestImageTag.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest_EmptyManifestImageName(t *testing.T) {
	errMsg := "Please provide image name for all modules"
	filePath := "../../testdata/unittests/failEmptyManifestImageName.json"
	err := utilFailTestValidateManifest(filePath, errMsg)
	assert.Nil(t, err)
}

func TestValidateManifest(t *testing.T) {
	json, err := ioutil.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}
	err = manifest.ValidateManifest(jsonParsed)
	assert.Nil(t, err)
}

func TestValidateUniqueIDExist_EmptyManifestName(t *testing.T) {
	assert := assert.New(t)
	manifestUniqueID.ManifestName = " "
	manifestUniqueID.VersionNumber = 1
	errMsg := "Please provide manifestName"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	err = manifest.ValidateUniqueIDExist(jsonParsed)
	assert.NotNil(err)
	if err != nil {
		assert.Equal(errMsg, err.Error())
	}
}

func TestValidateUniqueIDExist(t *testing.T) {
	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.VersionNumber = 1

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	err = manifest.ValidateUniqueIDExist(jsonParsed)
	assert.Nil(t, err)
}
