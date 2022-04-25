package internal

import (
	"io/ioutil"
	"os"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/deploy"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const ManifestFile = "Manifest.json"

var ManifestPath string

func ReadManifest() []byte {
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
	return byteValue
}
func DeploManifestLocal(topic_rcvd string, payload []byte, retry bool) bool {
	var DeploymentStatus = false
	log.Info("Processing the message >> ", topic_rcvd)
	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message : ", err)
	} else {
		log.Debug("Parsed JSON >> ", jsonParsed)
		var thisManifest = model.Manifest{}
		thisManifest.Manifest = *jsonParsed
		err := deploy.DeployDataService(thisManifest, topic_rcvd, true)
		if err == nil {
			DeploymentStatus = true
		}
	}
	return DeploymentStatus
}
