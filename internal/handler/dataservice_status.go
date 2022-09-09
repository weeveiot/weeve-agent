package handler

import (
	"strings"

	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

type edgeApplicationLog struct {
	ManifestID    string                `json:"manifestID"`
	Status        string                `json:"status"`
	ContainerLogs []docker.ContainerLog `json:"containerLog"`
}

type edgeApplications struct {
	ManifestID string      `json:"manifestID"`
	Status     string      `json:"status"`
	Containers []container `json:"containers"`
}

type container struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func GetDataServiceStatus() ([]edgeApplications, error) {
	edgeApps := []edgeApplications{}
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		edgeApplication := edgeApplications{ManifestID: manif.ManifestID, Status: manif.Status}
		containersStat := []container{}

		appContainers, err := docker.ReadDataServiceContainers(manif.ManifestUniqueID)
		if err != nil {
			return edgeApps, err
		}

		if !manif.InTransition && (manif.Status == model.EdgeAppRunning || manif.Status == model.EdgeAppPaused) && len(appContainers) != manif.ContainerCount {
			edgeApplication.Status = model.EdgeAppError
		}

		for _, con := range appContainers {
			// The Status of each container is (assumed to be): Running, Paused, Restarting, Created, Exited
			container := container{Name: strings.Join(con.Names, ", "), Status: ioutility.FirstToUpper(con.State)}
			containersStat = append(containersStat, container)

			if !manif.InTransition && edgeApplication.Status != model.EdgeAppError {
				if manif.Status == model.EdgeAppRunning && ioutility.FirstToUpper(con.State) != model.ModuleRunning {
					edgeApplication.Status = model.EdgeAppError
				}
				if manif.Status == model.EdgeAppPaused && ioutility.FirstToUpper(con.State) != model.ModulePaused {
					edgeApplication.Status = model.EdgeAppError
				}
			}
		}
		edgeApplication.Containers = containersStat
		edgeApps = append(edgeApps, edgeApplication)
	}
	return edgeApps, nil
}

func CompareDataServiceStatus(edgeApps []edgeApplications) ([]edgeApplications, bool, error) {
	statusChange := false

	latestEdgeApps, err := GetDataServiceStatus()
	if err != nil {
		return nil, false, err
	}
	if len(edgeApps) == len(latestEdgeApps) {
		for index, edgeApp := range edgeApps {
			if edgeApp.Status != latestEdgeApps[index].Status {
				statusChange = true
			}
		}
	} else {
		statusChange = true
	}
	return latestEdgeApps, statusChange, nil
}

func GetDataServiceLogs(edgeApps []edgeApplications) ([]edgeApplicationLog, error) {
	var edgeAppLogs []edgeApplicationLog
	for _, edgeApp := range edgeApps {
		edgeAppLog := edgeApplicationLog{ManifestID: edgeApp.ManifestID}

		for _, container := range edgeApp.Containers {
			logs, err := docker.ReadContainerLogs(container.Name, "", "")
			if err != nil {
				return []edgeApplicationLog{}, err
			}

			edgeAppLog.ContainerLogs = append(edgeAppLog.ContainerLogs, logs)
		}

		edgeAppLogs = append(edgeAppLogs, edgeAppLog)
	}

	return edgeAppLogs, nil
}
