package docker

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	// _ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)

var manifestBytesMVP []byte

const RootPath = "/"

func TestMain(m *testing.M) {

	fullManifestPath := "/testdata/pipeline_integration_public/workingMVP.json"
	manifestBytesMVP = LoadJsonBytes(fullManifestPath)
	// manifestBytesMVP = LoadJsonBytes("./testdata/pipeline_integration_public/workingMVP.json")
	code := m.Run()

	os.Exit(code)

	// manifest := ParseJSONManifest(manifestBytes)
	// fmt.Println(manifest
}

func LoadJsonBytes(filePath string) []byte {
	dir := filepath.Join(filepath.Dir(os.Args[1]) + RootPath)
	Root, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	manifestPath := path.Join(Root, filePath)
	//fmt.Println("Loading manifest from ", manifestPath)

	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}

// Unit function to validate negative tests
func TestImageExists(t *testing.T) {
	thisFilePath := "/testdata/pipeline_integration_public/failEmptyServices.json"
	json := LoadJsonBytes(thisFilePath)
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
