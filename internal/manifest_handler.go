package internal

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

const ManifestFile = "Manifest.json"

var ManifestPath string

func ReadManifest() map[string]interface{} {
	jsonFile, err := os.Open(ManifestPath)
	if err != nil {
		log.Fatalf("Unable to open Manifest file: %v", err)
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Unable to read node manifest file: %v", err)
	}

	var manifest map[string]interface{}

	json.Unmarshal(byteValue, &manifest)
0.....
	return manifest
}
