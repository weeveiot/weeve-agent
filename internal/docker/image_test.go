package docker_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

// Unit function to validate negative tests
func TestImageExists(t *testing.T) {
	thisFilePath := "../../testdata/pipeline_integration_public/failEmptyServices.json"
	json, err := ioutil.ReadFile(thisFilePath)
	if err != nil {
		t.Error(err)
	}

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
