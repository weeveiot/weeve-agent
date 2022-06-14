package handler

import (
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

type statusMessage struct {
	Status           string             `json:"status"`
	EdgeApplications []edgeApplications `json:"edgeApplications"`
	DeviceParams     deviceParams       `json:"deviceParams"`
}

const (
	Connected    string = "connected"
	Disconnected string = "disconnected"
	Running      string = "running"
	Alarm        string = "alarm"
	Restarting   string = "restarting"
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

type deviceParams struct {
	SystemUpTime float64 `json:"systemUpTime"`
	SystemLoad   float64 `json:"systemLoad"`
	StorageFree  uint64  `json:"storageFree"`
	RamFree      uint64  `json:"ramFree"`
}

type registrationMessage struct {
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Name      string `json:"name"`
}

func ProcessMessage(payload []byte) error {
	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		return err
	}
	log.Debug("Parsed JSON >> ", jsonParsed)

	operation, err := manifest.GetCommand(jsonParsed)
	if err != nil {
		return err
	}
	log.Info("Processing the message >> ", operation)

	switch operation {
	case dataservice.CMDDeploy:
		var err = manifest.ValidateManifest(jsonParsed)
		if err != nil {
			return err
		}

		manifest, err := manifest.GetManifest(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Deployment done!")

	case dataservice.CMDReDeploy:
		var err = manifest.ValidateManifest(jsonParsed)
		if err != nil {
			return err
		}

		manifest, err := manifest.GetManifest(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Redeployment done!")

	case dataservice.CMDStopService:

		err := manifest.ValidateStartStopJSON(jsonParsed)
		if err != nil {
			return err

		}

		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.StopDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Service stopped!")

	case dataservice.CMDStartService:

		err := manifest.ValidateStartStopJSON(jsonParsed)
		if err != nil {
			return err
		}
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.StartDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Service started!")

	case dataservice.CMDUndeploy:

		err := manifest.ValidateStartStopJSON(jsonParsed)
		if err != nil {
			return err
		}

		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Undeployment done!")
	}

	return nil
}

func GetStatusMessage(nodeId string) (statusMessage, error) {
	edgeApps := []edgeApplications{}
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		edgeApplication := edgeApplications{}
		containersStat := []container{}

		if manif.Status == "SUCCESS" {
			edgeApplication.Status = Connected

			appContainers, err := docker.ReadDataServiceContainers(manifest.ManifestUniqueID{ManifestName: manif.ManifestName, VersionName: manif.VersionName})
			if err != nil {
				return statusMessage{}, err
			}

			edgeApplication.Status = Running

			for _, con := range appContainers {
				container := container{Name: strings.Join(con.Names, ", "), Status: con.Status}
				containersStat = append(containersStat, container)

				if con.Status != Running {
					edgeApplication.Status = Alarm
					if con.Status == Restarting {
						edgeApplication.Status = Restarting
					}
				}
			}
		} else {
			edgeApplication.Status = manif.Status
		}

		edgeApplication.Containers = containersStat

		edgeApps = append(edgeApps, edgeApplication)
	}

	deviceParams := deviceParams{}
	if uptime, err := host.Uptime(); err == nil && uptime > 0 {
		deviceParams.SystemUpTime = float64((uptime / 60) / 24)
	}

	var per float64 = 0
	if cpu, err := cpu.Percent(0, false); err == nil {
		for _, c := range cpu {
			per = per + c
		}
		if len(cpu) > 0 {
			per = per / float64(len(cpu))
		}
	}
	deviceParams.SystemLoad = per

	if diskStat, err := disk.Usage("/"); err == nil {
		deviceParams.StorageFree = diskStat.Free
	}
	if verMem, err := mem.VirtualMemory(); err == nil {
		deviceParams.RamFree = verMem.Free
	}

	msg := statusMessage{
		Status:           "Available",
		EdgeApplications: edgeApps,
		DeviceParams:     deviceParams,
	}

	return msg, nil
}

func GetRegistrationMessage(nodeId string, nodeName string) registrationMessage {
	msg := registrationMessage{
		Id:        nodeId,
		Timestamp: time.Now().UnixMilli(),
		Status:    "Registering",
		Operation: "Registration",
		Name:      nodeName,
	}
	return msg
}
