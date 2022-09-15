package manifest

import (
	"encoding/json"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var knownManifests []model.ManifestStatus

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() []model.ManifestStatus {
	return knownManifests
}

func SetStatus(manifestID string, containerCount int, manifestUniqueID model.ManifestUniqueID, status string, inTransition bool, lastLogReadTime string) {
	log.Debugln("Setting status", status, "to data service", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)
	manifestKnown := false
	for i, manifest := range knownManifests {
		if manifest.ManifestUniqueID == manifestUniqueID {
			knownManifests[i].Status = status
			knownManifests[i].InTransition = inTransition
			if lastLogReadTime != "" {
				knownManifests[i].LastLogReadTime = lastLogReadTime
			}
			manifestKnown = true
			break
		}
	}
	if !manifestKnown {
		knownManifests = append(knownManifests, model.ManifestStatus{
			ManifestID:       manifestID,
			ManifestUniqueID: manifestUniqueID,
			Status:           status,
			ContainerCount:   containerCount,
			InTransition:     inTransition,
			LastLogReadTime:  lastLogReadTime,
		})
	}

	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(ManifestFile, encodedJson, 0644)
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

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &knownManifests)
	if err != nil {
		log.Fatal(err)
	}
}
