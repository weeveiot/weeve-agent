package dataservice

import (
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
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
	containerLogs, err := GetDataServiceLogs(manif, manif.LastLogReadTime, until)
	if err != nil {
		return msg, err
	}

	msg = com.EdgeAppLogMsg{
		ManifestID:    manif.ManifestID,
		ContainerLogs: containerLogs,
	}

	return msg, nil
}

func GetDataServiceLogs(manif model.ManifestStatus, since string, until string) ([]docker.ContainerLog, error) {
	var containerLogs []docker.ContainerLog

	appContainers, err := docker.ReadDataServiceContainers(manif.ManifestUniqueID)
	if err != nil {
		return nil, err
	}

	for _, container := range appContainers {
		logs, err := docker.ReadContainerLogs(container.ID, since, until)
		if err != nil {
			return nil, err
		}

		if len(logs.Log) > 0 {
			containerLogs = append(containerLogs, logs)
		}
	}

	return containerLogs, nil
}
