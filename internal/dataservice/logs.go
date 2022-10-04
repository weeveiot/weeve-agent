package dataservice

import (
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func SendEdgeAppLogs(manif model.ManifestStatus, until string) error {
	msg, err := GetEdgeAppLogsMsg(manif, until)
	if err != nil {
		return err
	}
	err = com.SendEdgeAppLogs(msg)
	if err != nil {
		return err
	}
	manifest.SetLastLogRead(manif.ManifestUniqueID, until)
	return nil
}

func GetEdgeAppLogsMsg(manif model.ManifestStatus, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	if manif.Status != model.EdgeAppUndeployed {
		containerLogs, err := GetDataServiceLogs(manif, manif.LastLogReadTime, until)
		if err != nil {
			return msg, err
		}

		msg = com.EdgeAppLogMsg{
			ManifestID:    manif.ManifestID,
			ContainerLogs: containerLogs,
		}
	}

	return msg, nil
}
