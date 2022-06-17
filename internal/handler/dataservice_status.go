package handler

import (
	"strings"

	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const (
	Connected  string = "connected"
	Running    string = "running"
	Alarm      string = "alarm"
	Restarting string = "restarting"
	Exited     string = "exited"
)

func GetDataServiceStatus() ([]model.EdgeApplications, error) {
	edgeApps := []model.EdgeApplications{}
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		edgeApplication := model.EdgeApplications{ManifestID: manif.ManifestId}
		containersStat := []model.Container{}

		if manif.Status == "DEPLOYED" {
			edgeApplication.Status = Connected

			appContainers, err := docker.ReadDataServiceContainers(manif.ManifestId, manif.ManifestVersion)
			if err != nil {
				return edgeApps, err
			}

			edgeApplication.Status = Running

			for _, con := range appContainers {
				container := model.Container{Name: strings.Join(con.Names, ", "), Status: con.State}
				containersStat = append(containersStat, container)
			}
			for _, con := range appContainers {
				if con.State == Exited {
					edgeApplication.Status = Alarm
					break
				}
				if con.State == Restarting {
					edgeApplication.Status = Restarting
				}
			}
		} else {
			edgeApplication.Status = manif.Status
		}

		edgeApplication.Containers = containersStat

		edgeApps = append(edgeApps, edgeApplication)
	}
	return edgeApps, nil
}

func CompareDataServiceStatus(edgeApps []model.EdgeApplications) ([]model.EdgeApplications, bool, error) {
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
