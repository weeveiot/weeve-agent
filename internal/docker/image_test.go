package docker

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	// _ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)


var manifestBytesMVP []byte
var filePath string
var errMsg string

func TestMain(m *testing.M) {

	fullManifestPath, err := os.Getwd() + "./testdata/newFormat020/workingMVP.json"
	t.Error(fullManifestPath)
	manifestBytesMVP = LoadJsonBytes(fullManifestPath)
	// manifestBytesMVP = LoadJsonBytes("./testdata/newFormat020/workingMVP.json")
	code := m.Run()

	os.Exit(code)

	// manifest := ParseJSONManifest(manifestBytes)
	// fmt.Println(manifest
}

func LoadJsonBytes(manName string) []byte {
	wd, _ := os.Getwd()
	fmt.Println()
	manifestPath := path.Join(wd, "testdata", manName)
	// fmt.Println("Loading manifest from ", manifestPath)

	var err error = nil
	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}

// Unit function to validate negative tests
func TestImageExists(t *testing.T) {
	thisFilePath := "newFormat020/failEmptyServices.json"
	json := LoadJsonBytes(thisFilePath)
	m, err := model.ParseJSONManifest(json)
	if err != nil {
		t.Error("Json parsing failed")
	}

	for _, srv := range m.Manifest.Search("compose").Search("services").Children() {
		moduleID := srv.Search("moduelId").Data()
		serviceName := srv.Search("name").Data()
		imageName := srv.Search("image").Search("name").Data()
		imageTag := srv.Search("image").Search("tag").Data()

		fmt.Println("Service:", moduleID, serviceName, imageName, imageTag)
	}


	// ImageExists()
	// err = ValidateManifest(m)
	// if err == nil {
	// 	t.Error(err.Error())
	// 	t.Error(errMsg)
	// }
}