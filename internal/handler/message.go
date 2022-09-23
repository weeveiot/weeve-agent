package handler

import (
	"errors"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/logger"
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
	operation, err := manifest.GetCommand(payload)
	if err != nil {
		return err
	}
	logger.Log.Infoln("Processing the", operation, "message")

	switch operation {
	case dataservice.CMDDeploy:
		manifest, err := manifest.Parse(payload)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		logger.Log.Info("Deployment done!")

	case dataservice.CMDReDeploy:
		manifest, err := manifest.Parse(payload)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		logger.Log.Info("Redeployment done!")

	case dataservice.CMDStopService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.StopDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		logger.Log.Info("Service stopped!")

	case dataservice.CMDStartService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.StartDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		logger.Log.Info("Service started!")

	case dataservice.CMDUndeploy:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		logger.Log.Info("Undeployment done!")

	case dataservice.CMDRemove:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		logger.Log.Info("Full removal done!")
	default:
		return errors.New("received message with unknown command")
	}

	return nil
}

func GetStatusMessage() (statusMessage, error) {
	edgeApps, err := GetDataServiceStatus()
	if err != nil {
		return statusMessage{}, err
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
