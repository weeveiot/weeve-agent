package model

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

var knownManifests []ManifestStatus

const ManifestFile = "known_manifests.json"

func GetKnownManifests() []ManifestStatus {
	return knownManifests
}

func SetStatus(id, version, status string) {
	manifestKnown := false
	for _, manifest := range knownManifests {
		if manifest.ManifestId == id && manifest.ManifestVersion == version {
			manifest.Status = status
			manifestKnown = true
			break
		}
	}
	if !manifestKnown {
		knownManifests = append(knownManifests, ManifestStatus{
			ManifestId:      id,
			ManifestVersion: version,
			Status:          status,
		})
	}

	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(ManifestFile, encodedJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func InitKnownManifests() {
	jsonFile, err := os.Open(ManifestFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &knownManifests)
	if err != nil {
		log.Fatal(err)
	}
}
