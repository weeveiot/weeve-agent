package manifest

import (
	"encoding/json"
	"io"
	"os"

	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

type ManifestRecord struct {
	Manifest        Manifest
	Status          string
	LastLogReadTime string
}

var knownManifests = make(map[model.ManifestUniqueID]*ManifestRecord)

const ManifestFile = "known_manifests.jsonl"

func GetKnownManifests() map[model.ManifestUniqueID]*ManifestRecord {
	return knownManifests
}

func GetKnownManifest(manifestUniqueID model.ManifestUniqueID) *ManifestRecord {
	return knownManifests[manifestUniqueID]
}

func GetUsedImages(uniqueID model.ManifestUniqueID) ([]string, error) {
	var images []string
	manifest, manifestKnown := knownManifests[uniqueID]
	if !manifestKnown {
		return nil, errors.New("could not get the images. the edge app is not known")
	}
	for _, module := range manifest.Manifest.Modules {
		images = append(images, module.ImageNameFull)
	}
	return images, nil
}

func AddKnownManifest(man Manifest) {
	manCopy := clearSecretValues(man) // remove some fields so that secret values never touch the hard disk
	knownManifests[man.UniqueID] = &ManifestRecord{
		Manifest: manCopy,
		Status:   model.EdgeAppInitiated,
	}
}

func DeleteKnownManifest(manifestUniqueID model.ManifestUniqueID) {
	delete(knownManifests, manifestUniqueID)

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal("Failed to write known manifest to file! CAUSE --> ", err)
	}
}

func SetStatus(manifestUniqueID model.ManifestUniqueID, status string) error {
	log.Debugln("Setting status", status, "to edge app", manifestUniqueID)

	manifest, manifestKnown := knownManifests[manifestUniqueID]
	if !manifestKnown {
		return errors.New("could not set the status. the edge app is not known (deployed)")
	}
	manifest.Status = status

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal("Failed to write known manifest to file! CAUSE --> ", err)
	}
	return nil
}

func SetLastLogRead(manifestUniqueID model.ManifestUniqueID, lastLogReadTime string) error {
	log.Debugln("Setting last log read time", lastLogReadTime, "to edge app", manifestUniqueID)

	manifest, manifestKnown := knownManifests[manifestUniqueID]
	if !manifestKnown {
		return errors.New("could not set the status. the edge app is not known (deployed)")
	}
	manifest.LastLogReadTime = lastLogReadTime

	err := writeKnownManifestsToFile()
	if err != nil {
		log.Fatal("Failed to write known manifest to file! CAUSE --> ", err)
	}
	return nil
}

func InitKnownManifests() error {
	log.Debug("Initializing known manifests...")

	jsonFile, err := os.Open(ManifestFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return json.Unmarshal(byteValue, &knownManifests)
}

func GetEdgeAppStatus(manifestUniqueID model.ManifestUniqueID) (string, error) {
	manifest, manifestKnown := knownManifests[manifestUniqueID]
	if !manifestKnown || manifest == nil {
		return "", errors.New("could not get the status. the edge app " + manifestUniqueID.String() + " is not known")
	}
	return manifest.Status, nil
}

func writeKnownManifestsToFile() error {
	encodedJson, err := json.MarshalIndent(knownManifests, "", " ")
	if err != nil {
		return traceutility.Wrap(err)
	}

	err = os.WriteFile(ManifestFile, encodedJson, 0644)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return nil
}
