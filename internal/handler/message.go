package handler

import (
	"errors"
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var disconnect bool

type statusMessage struct {
	Status           string             `json:"status"`
	EdgeApplications []edgeApplications `json:"edgeApplications"`
	DeviceParams     deviceParams       `json:"deviceParams"`
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

func SetDisconnected(disconnectParam bool) {
	disconnect = disconnectParam
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

		manifestUniqueID := manifest.GetEdgeAppUniqueID(jsonParsed)
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
		manifestUniqueID := manifest.GetEdgeAppUniqueID(jsonParsed)
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

		manifestUniqueID := manifest.GetEdgeAppUniqueID(jsonParsed)
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		log.Info("Undeployment done!")

	case dataservice.CMDRemove:

		err := manifest.ValidateUniqueIDExist(jsonParsed)
		if err != nil {
			return err
		}

		manifestUniqueID := manifest.GetEdgeAppUniqueID(jsonParsed)
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		log.Info("Full removal done!")
	default:
		return errors.New("Received message with unknown command")
	}

	return nil
}

func GetStatusMessage() (statusMessage, error) {
	edgeApps, err := GetDataServiceStatus()

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
	nodeStatus := model.NodeAlarm
	if config.GetRegistered() {
		nodeStatus = model.NodeConnected
	}

	if disconnect {
		nodeStatus = model.NodeDisconnected
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
