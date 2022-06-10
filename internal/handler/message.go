package handler

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
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
)

type edgeApplications struct {
	ManifestID string     `json:"manifestID"`
	Status     string     `json:"status"`
	Containers containers `json:"containers"`
}

type containers struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type deviceParams struct {
	SystemUpTime uint64  `json:"systemUpTime"`
	SystemLoad   float64 `json:"systemLoad"`
	StorageFree  int     `json:"storageFree"`
	RamFree      int     `json:"ramFree"`
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

func GetStatusMessage(nodeId string) statusMessage {
	knownManifests := manifest.GetKnownManifests()

	for _, manifest := range knownManifests {
		if manifest.Status == "SUCCESS" {

		}
	}

	deviceParams := deviceParams{}
	if uptime, err := host.Uptime(); err == nil {
		deviceParams.SystemUpTime = uptime
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

	msg := statusMessage{
		Status:           "Available",
		EdgeApplications: nil,
		DeviceParams:     deviceParams,
	}

	return msg
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
