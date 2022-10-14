package manifest

import (
	"encoding/json"
	"io"
	"os"

	linq "github.com/ahmetb/go-linq/v3"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var knownManifests []model.ManifestStatus

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() []model.ManifestStatus {
	return knownManifests
}

func DeleteKnownManifest(manifestUniqueID model.ManifestUniqueID) {
	var filteredKnownManifests []model.ManifestStatus

	linq.From(knownManifests).Where(func(c interface{}) bool {
		return c.(model.ManifestStatus).ManifestUniqueID != manifestUniqueID
	}).ToSlice(&filteredKnownManifests)

	knownManifests = filteredKnownManifests

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
}

func SetStatus(manifestID string, containerCount int, manifestUniqueID model.ManifestUniqueID, status string, inTransition bool) {
	if status != "" {
		log.Debugln("Setting status", status, "to data service", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)
	}

	manifestKnown := false
	for i, manifest := range knownManifests {
		if manifest.ManifestUniqueID == manifestUniqueID {
			if status != "" {
				knownManifests[i].Status = status
			}
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

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
}

func SetLastLogRead(manifestUniqueID model.ManifestUniqueID, lastLogReadTime string) {
	log.Debugln("Setting last log read time", lastLogReadTime, "to data service", manifestUniqueID)

	for i, manifest := range knownManifests {
		if manifest.ManifestUniqueID == manifestUniqueID {
			if lastLogReadTime != "" {
				knownManifests[i].LastLogReadTime = lastLogReadTime
			}
			break
		}
	}

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
}

func InitKnownManifests() error {
	jsonFile, err := os.Open(ManifestFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(byteValue, &knownManifests)
}

func GetEdgeAppStatus(manif model.ManifestUniqueID) string {
	for _, manifest := range knownManifests {
		if manif.ManifestName == manifest.ManifestUniqueID.ManifestName && manif.VersionNumber == manifest.ManifestUniqueID.VersionNumber {
			return manifest.Status
		}
	}

	return ""
}

func writeKnownManifestsToFile() error {
	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(ManifestFile, encodedJson, 0644)
	if err != nil {
		return err
	}

	return nil
}
