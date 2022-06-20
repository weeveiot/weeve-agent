package handler

import (
	"strings"

	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

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

		if !manif.InTransition && (manif.Status == manifest.Running || manif.Status == manifest.Paused) && len(appContainers) != manif.ContainerCount {
			edgeApplication.Status = manifest.Error
		}

		for _, con := range appContainers {
			container := container{Name: strings.Join(con.Names, ", "), Status: con.State}
			containersStat = append(containersStat, container)

			if !manif.InTransition && edgeApplication.Status != manifest.Error {
				// if manif.Status == manifest.Running && con.State != manifest.Running {
				// 	edgeApplication.Status = manifest.Error
				// }
				if manif.Status == manifest.Paused && con.State != manifest.Paused {
					edgeApplication.Status = manifest.Error
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
