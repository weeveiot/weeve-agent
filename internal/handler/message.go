package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

type statusMessage struct {
	Status           string             `json:"status"`
	EdgeApplications []edgeApplications `json:"edgeApplications"`
	DeviceParams     deviceParams       `json:"deviceParams"`
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

		err := manifest.ValidateUniqueIDExist(jsonParsed)
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

		err := manifest.ValidateUniqueIDExist(jsonParsed)
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

		err := manifest.ValidateUniqueIDExist(jsonParsed)
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
	default:
		return errors.New("Received message with unknown command")
	}

	return nil
}

func GetStatusMessage() (statusMessage, error) {
	edgeApps := []edgeApplications{}
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		edgeApplication := edgeApplications{ManifestID: manif.ManifestID, Status: manif.Status}
		containersStat := []container{}

		appContainers, err := docker.ReadDataServiceContainers(manif.ManifestUniqueID)
		if err != nil {
			return statusMessage{}, err
		}

		if !manif.InTransition && (manif.Status == manifest.Running || manif.Status == manifest.Paused) && len(appContainers) != manif.ContainerCount {
			edgeApplication.Status = manifest.Error
		}

		for _, con := range appContainers {
			container := container{Name: strings.Join(con.Names, ", "), Status: con.State}
			containersStat = append(containersStat, container)

			if !manif.InTransition && edgeApplication.Status != manifest.Error {
				if manif.Status == manifest.Running && con.State != manifest.Running {
					edgeApplication.Status = manifest.Error
				}
				if manif.Status == manifest.Paused && con.State != manifest.Paused {
					edgeApplication.Status = manifest.Error
				}
			}
		}

		edgeApplication.Containers = containersStat
		edgeApps = append(edgeApps, edgeApplication)
	}

	deviceParams := deviceParams{}
	uptime, err := host.Uptime()
	if err != nil {
		return statusMessage{}, err
	}
	if uptime > 0 {
		deviceParams.SystemUpTime = float64((uptime / 60) / 24)
	}

	deviceParams.SystemLoad = 0
	cpu, err := cpu.Percent(0, false)
	if err != nil {
		return statusMessage{}, err
	}
	if len(cpu) > 0 {
		for _, c := range cpu {
			deviceParams.SystemLoad = deviceParams.SystemLoad + c
		}
		deviceParams.SystemLoad = deviceParams.SystemLoad / float64(len(cpu))
	}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return statusMessage{}, err
	}
	deviceParams.StorageFree = diskStat.Free

	verMem, err := mem.VirtualMemory()
	if err != nil {
		return statusMessage{}, err
	}
	deviceParams.RamFree = verMem.Free

	// TODO: Do proper check for node status
	nodeStatus := manifest.Alarm
	if config.GetRegistered() {
		nodeStatus = manifest.Connected
	}

	msg := statusMessage{
		Status:           nodeStatus,
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
