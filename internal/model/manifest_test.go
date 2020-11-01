package model

import (
	"fmt"
	"os"
	"testing"

	// "gitlab.com/weeve/poc-festo/poc-festo-mqtts-ethereum-gateway/internal/parser"

	"github.com/stretchr/testify/assert"
	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)



func TestManifestCreate(t *testing.T) {
	wd, _ := os.Getwd()
	fmt.Println(wd)
	// manifestPath := path.Join(wd, "testdata", "test_manifest1.json")
	// file, err := os.Open(path.Join(wd, "testdata", "test_manifest1.json"))
	// manifestBytes, err := ioutil.ReadFile(manifestPath)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// manifest := manifest.ParseJSON(manifestBytes)
	// fmt.Println(file)
	// Parse the bytes into the 'gabs' json package
	// // jsonParsed, err := gabs
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(manifest)

	assert.True(t, true)
}