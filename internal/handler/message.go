package handler

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var disconnect bool

type statusMessage struct {
	Status           string              `json:"status"`
	EdgeApplications []edgeApplications  `json:"edgeApplications"`
	DeviceParams     deviceParamsMessage `json:"deviceParams"`
}

type deviceParamsMessage struct {
	SystemUpTime uint64  `json:"systemUpTime"`
	SystemLoad   float64 `json:"systemLoad"`
	StorageFree  float64 `json:"storageFree"`
	RamFree      float64 `json:"ramFree"`
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
	log.Infoln("Processing the", operation, "message")

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
		log.Info("Deployment done!")

	case dataservice.CMDReDeploy:
		manifest, err := manifest.Parse(payload)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Redeployment done!")

	case dataservice.CMDStopService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.StopDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Service stopped!")

	case dataservice.CMDStartService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.StartDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Service started!")

	case dataservice.CMDUndeploy:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		log.Info("Undeployment done!")

	case dataservice.CMDRemove:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID, operation)
		if err != nil {
			return err
		}
		log.Info("Full removal done!")
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

	deviceParams, err := getDeviceParams()
	if err != nil {
		return statusMessage{}, err
	}

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

func getDeviceParams() (deviceParamsMessage, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return deviceParamsMessage{}, err
	}

	cpu, err := cpu.Percent(0, false)
	if err != nil {
		return deviceParamsMessage{}, err
	}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return deviceParamsMessage{}, err
	}

	verMem, err := mem.VirtualMemory()
	if err != nil {
		return deviceParamsMessage{}, err
	}

	params := deviceParamsMessage{
		SystemUpTime: uptime,
		SystemLoad:   cpu[0],
		StorageFree:  100.0 - diskStat.UsedPercent,
		RamFree:      float64(verMem.Available) / float64(verMem.Total),
	}
	return params, nil
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
