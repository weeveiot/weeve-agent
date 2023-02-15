package manifest_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"

	"github.com/weeveiot/weeve-agent/internal/manifest"
)

var manifestUniqueID struct {
	ManifestName string `json:"manifestName"`
	UpdatedAt    string `json:"updatedAt"`
}

func TestGetManifest(t *testing.T) {
	assert := assert.New(t)

	json, err := os.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Fatal(err)
	}

	manifest, _ := manifest.Parse(json)

	assert.NotNil(manifest)
	assert.Equal("kunbus-demo-manifest", manifest.ManifestUniqueID.ManifestName)
	assert.Equal("2023-01-01T00:00:00Z", manifest.ManifestUniqueID.UpdatedAt)
	assert.Equal(3, len(manifest.Connections))
	assert.Equal(4, len(manifest.Modules))

	if len(manifest.Modules) == 4 {
		assert.Equal(3, len(manifest.Modules[0].Labels))
		assert.Equal("weevenetwork/mqtt-ingress:V1", manifest.Modules[0].ImageName)
		assert.Equal(11, len(manifest.Modules[0].EnvArgs))
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
		assert.Equal(13, len(manifest.Modules[0].EnvArgs))
		if (len(manifest.Modules[0].EnvArgs)) == 12 {
			assert.Equal("INGRESS_HOST=kunbus-demo-manifest_1d.weevenetwork_mqtt-ingress_V1.0", manifest.Modules[0].EnvArgs[10])
			assert.Equal("EGRESS_URLS=http://kunbus-demo-manifest_1d.weevenetwork_fluctuation-filter_V1.1:80/", manifest.Modules[0].EnvArgs[11])
		}
	}
}

func TestGetEdgeAppUniqueID(t *testing.T) {
	assert := assert.New(t)

	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.UpdatedAt = "2023-01-01T00:00:00Z"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	man, err := manifest.GetEdgeAppUniqueID(json)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(manifestUniqueID.ManifestName, man.ManifestName)
	assert.Equal(manifestUniqueID.UpdatedAt, man.UpdatedAt)
}

func TestGetCommand_MissingCommand(t *testing.T) {
	assert := assert.New(t)
	errMsg := "Key: 'commandMsg.Command' Error:Field validation for 'Command' failed on the 'required' tag"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := manifest.GetCommand(json)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), errMsg)
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

	cmd, err := manifest.GetCommand(json)
	assert.Nil(err)
	if err == nil {
		assert.Equal(commandJson.Command, cmd)
	}
}

// Utility function to run ValidateManifest fail tests
func utilFailTestValidateManifest(t *testing.T, filePath string, errMsg string) {
	assert := assert.New(t)

	json, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manifest.Parse(json)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), errMsg)
	}
}

func TestValidateManifest_MissingManifestID(t *testing.T) {
	errMsg := "Key: 'manifestMsg.ID' Error:Field validation for 'ID' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestID.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_EmptyManifestID(t *testing.T) {
	errMsg := "Key: 'manifestMsg.ID' Error:Field validation for 'ID' failed on the 'notblank' tag"
	filePath := "../../testdata/unittests/failEmptyManifestID.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_MissingManifestName(t *testing.T) {
	errMsg := "Key: 'manifestMsg.ManifestName' Error:Field validation for 'ManifestName' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestName.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_EmptyManifestName(t *testing.T) {
	errMsg := "Key: 'manifestMsg.ManifestName' Error:Field validation for 'ManifestName' failed on the 'notblank' tag"
	filePath := "../../testdata/unittests/failEmptyManifestName.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_MissingManifestUpdatedAt(t *testing.T) {
	errMsg := "Key: 'manifestMsg.UpdatedAt' Error:Field validation for 'UpdatedAt' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestUpdatedAt.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_MissingManifestCommand(t *testing.T) {
	errMsg := "Key: 'manifestMsg.Command' Error:Field validation for 'Command' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestCommand.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_EmptyManifestCommand(t *testing.T) {
	errMsg := "Key: 'manifestMsg.Command' Error:Field validation for 'Command' failed on the 'notblank' tag"
	filePath := "../../testdata/unittests/failEmptyManifestCommand.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_MissingManifestModules(t *testing.T) {
	errMsg := "Key: 'manifestMsg.Modules' Error:Field validation for 'Modules' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestModules.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_EmptyManifestModules(t *testing.T) {
	errMsg := "Key: 'manifestMsg.Modules' Error:Field validation for 'Modules' failed on the 'notblank' tag"
	filePath := "../../testdata/unittests/failEmptyManifestModules.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_MissingManifestImageName(t *testing.T) {
	errMsg := "Key: 'moduleMsg.Image.Name' Error:Field validation for 'Name' failed on the 'required' tag"
	filePath := "../../testdata/unittests/failMissingManifestImageName.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest_EmptyManifestImageName(t *testing.T) {
	errMsg := "Key: 'moduleMsg.Image.Name' Error:Field validation for 'Name' failed on the 'notblank' tag"
	filePath := "../../testdata/unittests/failEmptyManifestImageName.json"
	utilFailTestValidateManifest(t, filePath, errMsg)
}

func TestValidateManifest(t *testing.T) {
	json, err := os.ReadFile("../../testdata/unittests/mvpManifest.json")
	if err != nil {
		t.Fatal(err)
	}

	_, err = manifest.Parse(json)
	assert.Nil(t, err)
}

func TestValidateUniqueIDExist_EmptyManifestName(t *testing.T) {
	assert := assert.New(t)
	manifestUniqueID.ManifestName = " "
	manifestUniqueID.UpdatedAt = "2021-05-11T12:00:00Z"
	errMsg := "Key: 'uniqueIDmsg.ManifestName' Error:Field validation for 'ManifestName' failed on the 'notblank' tag"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manifest.GetEdgeAppUniqueID(json)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), errMsg)
	}
}

func TestValidateUniqueIDExist(t *testing.T) {
	manifestUniqueID.ManifestName = "kunbus-demo-manifest"
	manifestUniqueID.UpdatedAt = "2021-05-11T12:00:00Z"

	json, err := json.Marshal(manifestUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manifest.GetEdgeAppUniqueID(json)
	assert.Nil(t, err)
}
