package dataservice

import (
	"github.com/pkg/errors"
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

func SendEdgeAppLogs(manif manifest.ManifestStatus, until string) error {
	msg, err := GetEdgeAppLogsMsg(manif, until)
	if err != nil {
		return errors.Wrap(err, "SendEdgeAppLogs")
	}
	err = com.SendEdgeAppLogs(msg)
	if err != nil {
		return errors.Wrap(err, "SendEdgeAppLogs")
	}

	return manifest.SetLastLogRead(manif.Manifest.ManifestUniqueID, until)
}

func GetEdgeAppLogsMsg(manif manifest.ManifestStatus, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	containerLogs, err := GetDataServiceLogs(manif, manif.LastLogReadTime, until)
	if err != nil {
		return msg, errors.Wrap(err, "GetEdgeAppLogsMsg")
	}

	msg = com.EdgeAppLogMsg{
		ManifestID:    manif.Manifest.ID,
		ContainerLogs: containerLogs,
	}

	return msg, nil
}

func GetDataServiceLogs(manif manifest.ManifestStatus, since string, until string) ([]docker.ContainerLog, error) {
	var containerLogs []docker.ContainerLog

	appContainers, err := docker.ReadDataServiceContainers(manif.Manifest.ManifestUniqueID)
	if err != nil {
		return nil, errors.Wrap(err, "GetDataServiceLogs")
	}

	for _, container := range appContainers {
		logs, err := docker.ReadContainerLogs(container.ID, since, until)
		if err != nil {
			return nil, errors.Wrap(err, "GetDataServiceLogs")
		}

		if len(logs.Log) > 0 {
			containerLogs = append(containerLogs, logs)
		}
	}

	return containerLogs, nil
}
