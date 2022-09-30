package handler

import (
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func GetEdgeAppLogsMsg(manif model.ManifestStatus, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	if manif.Status != model.EdgeAppUndeployed {
		containerLogs, err := dataservice.GetDataServiceLogs(manif, manif.LastLogReadTime, until)
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
