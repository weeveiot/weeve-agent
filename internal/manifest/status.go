package manifest

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var knownManifests []model.ManifestStatus

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() []model.ManifestStatus {
	return knownManifests
}

func SetStatus(manifestUniqueID ManifestUniqueID, status string) {
	log.Debugln("Setting status", status, "to data service", manifestUniqueID.ApplicationID, manifestUniqueID.VersionName)
	manifestKnown := false
	for i, manifest := range knownManifests {
		if manifest.ApplicationId == manifestUniqueID.ApplicationID && manifest.VersionName == manifestUniqueID.VersionName {
			knownManifests[i].Status = status
			manifestKnown = true
			break
		}
	}
	if !manifestKnown {
		knownManifests = append(knownManifests, model.ManifestStatus{
			ApplicationId: manifestUniqueID.ApplicationID,
			VersionName:   manifestUniqueID.VersionName,
			Status:        status,
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
