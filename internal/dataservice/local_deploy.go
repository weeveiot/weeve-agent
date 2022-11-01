package dataservice

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

// Read local manifest file provided and deploy it
func ReadDeployManifestLocal(manifestPath string) error {
	log.Info("Reading local manifest to deploy...")

	jsonFile, err := os.Open(manifestPath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	thisManifest, err := manifest.Parse(byteValue)
	if err != nil {
		return err
	}

	err = UndeployDataService(thisManifest.ManifestUniqueID)
	if err != nil {
		return err
	}

	err = DeployDataService(thisManifest)
	if err != nil {
		return err
	}

	return nil
}
