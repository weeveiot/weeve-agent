package docker

import (
	"fmt"
	"os"
	"testing"

	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util"
)

var manifestBytesMVP []byte

func TestMain(m *testing.M) {

	fullManifestPath := "/testdata/pipeline_integration_public/workingMVP.json"
	manifestBytesMVP = util.LoadJsonBytes(fullManifestPath)
	code := m.Run()

	os.Exit(code)
}

// Unit function to validate negative tests
func TestImageExists(t *testing.T) {
	thisFilePath := "/testdata/pipeline_integration_public/failEmptyServices.json"
	json := util.LoadJsonBytes(thisFilePath)
	m, err := model.ParseJSONManifest(json)
	if err != nil {
		t.Error("Json parsing failed")
	}

	for _, srv := range m.Manifest.Search("services").Children() {
		moduleID := srv.Search("moduelId").Data()
		serviceName := srv.Search("name").Data()
		imageName := srv.Search("image").Search("name").Data()
		imageTag := srv.Search("image").Search("tag").Data()

		fmt.Println("Service:", moduleID, serviceName, imageName, imageTag)
	}
}
