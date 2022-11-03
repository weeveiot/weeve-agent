package dataservice

import (
	"io"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/manifest"
)

// Read local manifest file provided and deploy it
func ReadDeployManifestLocal(manifestPath string) error {
	log.Info("Reading local manifest to deploy...")

	jsonFile, err := os.Open(manifestPath)
	if err != nil {
		return errors.Wrap(err, "ReadDeployManifestLocal")
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return errors.Wrap(err, "ReadDeployManifestLocal")
	}

	thisManifest, err := manifest.Parse(byteValue)
	if err != nil {
		return errors.Wrap(err, "ReadDeployManifestLocal")
	}

	err = UndeployDataService(thisManifest.ManifestUniqueID)
	if err != nil {
		return errors.Wrap(err, "ReadDeployManifestLocal")
	}

	err = DeployDataService(thisManifest)
	if err != nil {
		return errors.Wrap(err, "ReadDeployManifestLocal")
	}

	return nil
}
