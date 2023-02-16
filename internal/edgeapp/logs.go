package edgeapp

import (
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

func SendEdgeAppLogs(manif manifest.ManifestRecord, until string) error {
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

func GetEdgeAppLogsMsg(manif manifest.ManifestRecord, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	containerLogs, err := GetEdgeAppLogs(manif.Manifest.ManifestUniqueID, manif.LastLogReadTime, until)
	if err != nil {
		return msg, traceutility.Wrap(err)
	}

	msg = com.EdgeAppLogMsg{
		ManifestID:    manif.Manifest.ID,
		ContainerLogs: containerLogs,
	}

	return msg, nil
}

func GetEdgeAppLogs(uniqueID model.ManifestUniqueID, since string, until string) ([]docker.ContainerLog, error) {
	var containerLogs []docker.ContainerLog

	appContainers, err := docker.ReadEdgeAppContainers(uniqueID)
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
