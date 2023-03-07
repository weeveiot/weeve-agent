package edgeapp

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/manifest"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

// Read local manifest file provided and deploy it
func ReadDeployManifestLocal(manifestPath string) error {
	log.Info("Reading local manifest to deploy...")

	jsonFile, err := os.Open(manifestPath)
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return traceutility.Wrap(err)
	}

	thisManifest, err := manifest.Parse(byteValue)
	if err != nil {
		return traceutility.Wrap(err)
	}

	err = UndeployEdgeApp(thisManifest.UniqueID)
	if err != nil {
		return traceutility.Wrap(err)
	}

	err = DeployEdgeApp(thisManifest)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return nil
}
