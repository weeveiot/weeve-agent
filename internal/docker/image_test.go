package docker_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

var manifestBytesMVP []byte

func TestMain(m *testing.M) {

	fullManifestPath := "/testdata/pipeline_integration_public/workingMVP.json"
	manifestBytesMVP = ioutility.LoadJsonBytes(fullManifestPath)
	code := m.Run()

	os.Exit(code)
}

// Unit function to validate negative tests
func TestImageExists(t *testing.T) {
	thisFilePath := "/testdata/pipeline_integration_public/failEmptyServices.json"
	json := ioutility.LoadJsonBytes(thisFilePath)
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error(err.Error())
	}
	m, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error("Json parsing failed")
	}

	for _, module := range m.Modules {
		fmt.Println("Service:", module.ImageName, module.ImageTag)
	}
}
