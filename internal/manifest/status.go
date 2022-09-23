package manifest

import (
	"encoding/json"
	"io"
	"os"

	"github.com/weeveiot/weeve-agent/internal/logger"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var knownManifests []model.ManifestStatus

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() []model.ManifestStatus {
	return knownManifests
}

func SetStatus(manifestID string, containerCount int, manifestUniqueID model.ManifestUniqueID, status string, inTransition bool) {
	logger.Log.Debugln("Setting status", status, "to data service", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)
	manifestKnown := false
	for i, manifest := range knownManifests {
		if manifest.ManifestUniqueID == manifestUniqueID {
			knownManifests[i].Status = status
			knownManifests[i].InTransition = inTransition
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
		})
	}

	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		logger.Log.Fatal(err)
	}

	err = os.WriteFile(ManifestFile, encodedJson, 0644)
	if err != nil {
		logger.Log.Fatal(err)
	}
}

func SetLastLogRead(manifestUniqueID model.ManifestUniqueID, lastLogReadTime string) {
	logger.Log.Debugln("Setting last log read time", lastLogReadTime, "to data service", manifestUniqueID)

	for i, manifest := range knownManifests {
		if manifest.ManifestUniqueID == manifestUniqueID {
			if lastLogReadTime != "" {
				knownManifests[i].LastLogReadTime = lastLogReadTime
			}
			break
		}
	}

	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		logger.Log.Fatal(err)
	}

	err = os.WriteFile(ManifestFile, encodedJson, 0644)
	if err != nil {
		logger.Log.Fatal(err)
	}
}

func InitKnownManifests() {
	jsonFile, err := os.Open(ManifestFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		logger.Log.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &knownManifests)
	if err != nil {
		logger.Log.Fatal(err)
	}
}
