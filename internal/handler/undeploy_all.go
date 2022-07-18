package handler

import (
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

func UndeployAll() error {
	knownManifests := manifest.GetKnownManifests()
	log.Info(knownManifests)

	if len(knownManifests) > 0 {
		for _, manif := range knownManifests {
			err := dataservice.UndeployDataService(manif.ManifestUniqueID, dataservice.CMDRemove)
			if err != nil {
				return err
			}
		}
		log.Info("All the edge applications have been undeployed")
	} else {
		log.Info("No edge application to undeploy")
	}
	return nil
}
