package manifest

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

type ManifestStatus struct {
	Manifest        Manifest
	Status          string
	LastLogReadTime string
}

var knownManifests = make(map[model.ManifestUniqueID]*ManifestStatus)

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() map[model.ManifestUniqueID]*ManifestStatus {
	return knownManifests
}

func GetUsedImages(uniqueID model.ManifestUniqueID) ([]string, error) {
	var images []string
	manifest, manifestKnown := knownManifests[uniqueID]
	if !manifestKnown {
		return nil, errors.New("could not get the images. the edge app is not known")
	}
	for _, module := range manifest.Manifest.Modules {
		images = append(images, module.ImageName)
	}
	return images, nil
}

func AddKnownManifest(man Manifest) {
	manCopy := clearSecretValues(man) // remove some fields so that secret values never touch the hard disk
	knownManifests[man.ManifestUniqueID] = &ManifestStatus{
		Manifest: manCopy,
		Status:   model.EdgeAppInitiated,
	}
}

func DeleteKnownManifest(manifestUniqueID model.ManifestUniqueID) {
	delete(knownManifests, manifestUniqueID)

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
}

func SetStatus(manifestUniqueID model.ManifestUniqueID, status string) error {
	log.Debugln("Setting status", status, "to data service", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)

	manifest, manifestKnown := knownManifests[manifestUniqueID]
	if !manifestKnown {
		return errors.New("could not set the status. the edge app is not known (deployed)")
	}
	manifest.Status = status

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func SetLastLogRead(manifestUniqueID model.ManifestUniqueID, lastLogReadTime string) error {
	log.Debugln("Setting last log read time", lastLogReadTime, "to data service", manifestUniqueID)

	manifest, manifestKnown := knownManifests[manifestUniqueID]
	if !manifestKnown {
		return errors.New("could not set the status. the edge app is not known (deployed)")
	}
	manifest.LastLogReadTime = lastLogReadTime

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal(err)
	}
	return nil
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

func GetEdgeAppStatus(manifestUniqueID model.ManifestUniqueID) string {
	return knownManifests[manifestUniqueID].Status
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
