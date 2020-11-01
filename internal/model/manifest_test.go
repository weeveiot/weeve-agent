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
	// wd, _ := os.Getwd()
	// fmt.Println(wd)
	// manifestPath := path.Join(wd, "testdata", "test_manifest1.json")
	// // file, err := os.Open(path.Join(wd, "testdata", "test_manifest1.json"))
	// manifestBytes, err := ioutil.ReadFile(manifestPath)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	manifest := ParseJSONManifest(manifestBytes)
	fmt.Println(manifest.manifest)
	// manifest := manifest.ParseJSON(manifestBytes)
	// fmt.Println(file)
	// Parse the bytes into the 'gabs' json package
	// jsonParsed, err := gabs
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(manifest)

	assert.True(t, true)
}

// func TestManifestImageNames(t *testing.T) {
// 	M	ImageNamesList
// }