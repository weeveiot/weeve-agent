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

var manifestBytesSimple []byte
var manifestBytesNoModules []byte
var manifestBytes3nodesBroker []byte

func TestMain(m *testing.M) {

	manifestBytesSimple = LoadJsonBytes("test_manifest1.json")
	manifestBytesNoModules = LoadJsonBytes("test_manifest_no_modules.json")
	manifestBytes3nodesBroker = LoadJsonBytes("test_manifest_3broker.json")
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
func TestManifestCreate(t *testing.T) {

	manifest, err := ParseJSONManifest(manifestBytesSimple)
	if err != nil {
		panic(err)
	}
	fmt.Println("Manifest", manifest.ID, "with", manifest.NumModules, "modules")

	assert.Equal(t, manifest.ID, "test-manifest-1")
}

func TestManifestFailNoModules(t *testing.T) {
	_, err := ParseJSONManifest(manifestBytesNoModules)
	assert.Error(t, err)
}

func TestGetImageNamesList(t *testing.T) {
	manifest, err := ParseJSONManifest(manifestBytesSimple)
	if err != nil {
		panic(err)
	}
	imgNameList := manifest.ImageNamesList()
	for i, img := range imgNameList {
		fmt.Println("Image", i, img)
	}
}

func TestGetContainerNamesList(t *testing.T) {
	manifest, err := ParseJSONManifest(manifestBytesSimple)
	if err != nil {
		panic(err)
	}
	conNameList := manifest.ContainerNamesList()
	for i, img := range conNameList {
		fmt.Println("Container", i, img)
	}
}

func TestGetStartCommands(t *testing.T) {
	manifest, err := ParseJSONManifest(manifestBytesSimple)
	if err != nil {
		panic(err)
	}
	startCommands := manifest.GetContainerStart()
	for i, command := range startCommands {
		fmt.Println("Start", i, command)
	}
}

func TestStartOptionsComplex(t *testing.T) {
	manifest, err := ParseJSONManifest(manifestBytes3nodesBroker)
	if err != nil {
		panic(err)
	}
	startCommands := manifest.GetContainerStart()
	for i, command := range startCommands {
		fmt.Println("Start", i, command)
		PrintStartCommand(command)
		fmt.Println("Options:", command.Options)
	}
}
