package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	// "gitlab.com/weeve/poc-festo/poc-festo-mqtts-ethereum-gateway/internal/parser"

	"github.com/stretchr/testify/assert"
	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)

var manifestBytes []byte

func TestMain(m *testing.M){

	wd, _ := os.Getwd()
	fmt.Println()
	manifestPath := path.Join(wd, "testdata", "test_manifest1.json")
	fmt.Println("Loading manifest from ", manifestPath)

	var err error = nil
	manifestBytes, err = ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	os.Exit(code)

	// manifest := ParseJSONManifest(manifestBytes)
	// fmt.Println(manifest
}

func TestManifestCreate(t *testing.T) {

	manifest := ParseJSONManifest(manifestBytes)
	fmt.Println("Manifest created, ID: ", manifest.ID)
	assert.Equal(t, manifest.ID, "test-manifest-1")
}

func TestGetImageNamesList(t *testing.T) {
	// M	ImageNamesList
	manifest := ParseJSONManifest(manifestBytes)
	// fmt.Println(manifest.manifest)
	_ = manifest.Manifest
	imgNameList := manifest.ImageNamesList()
	for i, img := range(imgNameList) {
		fmt.Println("Image", i, img)
	}
}