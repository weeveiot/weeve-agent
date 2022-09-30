package handler

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func GetEdgeAppLogsMsg() ([]com.EdgeAppLogMsg, error) {
	log.Debug("Check if new logs available for edge apps")
	knownManifests := manifest.GetKnownManifests()

	var msgs []com.EdgeAppLogMsg
	for _, manif := range knownManifests {
		if manif.Status != model.EdgeAppUndeployed {
			since := manif.LastLogReadTime
			until := time.Now().UTC().Format(time.RFC3339Nano)

			containerLogs, err := dataservice.GetDataServiceLogs(manif, since, until)
			if err != nil {
				return nil, err
			}
			manifest.SetLastLogRead(manif.ManifestUniqueID, until)

			msg := com.EdgeAppLogMsg{
				ManifestID:    manif.ManifestID,
				ContainerLogs: containerLogs,
			}
			msgs = append(msgs, msg)
		}
	}

	return msgs, nil
}
