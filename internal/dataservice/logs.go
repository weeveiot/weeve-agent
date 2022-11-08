package dataservice

import (
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

func SendEdgeAppLogs(manif manifest.ManifestStatus, until string) error {
	msg, err := GetEdgeAppLogsMsg(manif, until)
	if err != nil {
		return traceutility.Wrap(err)
	}
	err = com.SendEdgeAppLogs(msg)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return manifest.SetLastLogRead(manif.Manifest.ManifestUniqueID, until)
}

func GetEdgeAppLogsMsg(manif manifest.ManifestStatus, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	containerLogs, err := GetDataServiceLogs(manif, manif.LastLogReadTime, until)
	if err != nil {
		return msg, traceutility.Wrap(err)
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
		return nil, traceutility.Wrap(err)
	}

	for _, container := range appContainers {
		logs, err := docker.ReadContainerLogs(container.ID, since, until)
		if err != nil {
			return nil, traceutility.Wrap(err)
		}

		if len(logs.Log) > 0 {
			containerLogs = append(containerLogs, logs)
		}
	}

	return containerLogs, nil
}
