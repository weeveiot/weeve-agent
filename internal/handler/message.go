package handler

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

type statusMessage struct {
	Id                 string                 `json:"ID"`
	Timestamp          int64                  `json:"timestamp"`
	Status             string                 `json:"status"`
	ActiveServiceCount int                    `json:"activeServiceCount"`
	ServiceCount       int                    `json:"serviceCount"`
	ServicesStatus     []model.ManifestStatus `json:"servicesStatus"`
	DeviceParams       deviceParams           `json:"deviceParams"`
}

type deviceParams struct {
	Sensors string `json:"sensors"`
	Uptime  string `json:"uptime"`
	CpuTemp string `json:"cputemp"`
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
		serviceId := jsonParsed.Search("id").Data().(string)
		serviceVersion := jsonParsed.Search("version").Data().(string)

		err = dataservice.StopDataService(serviceId, serviceVersion)
		if err != nil {
			return err
		}
		log.Info("Service stopped!")

	case dataservice.CMDStartService:

		err := manifest.ValidateStartStopJSON(jsonParsed)
		if err != nil {
			return err
		}
		serviceId := jsonParsed.Search("id").Data().(string)
		serviceVersion := jsonParsed.Search("version").Data().(string)

		err = dataservice.StartDataService(serviceId, serviceVersion)
		if err != nil {
			return err
		}
		log.Info("Service started!")

	case dataservice.CMDUndeploy:

		err := manifest.ValidateStartStopJSON(jsonParsed)
		if err != nil {
			return err
		}
		serviceId := jsonParsed.Search("id").Data().(string)
		serviceVersion := jsonParsed.Search("version").Data().(string)

		err = dataservice.UndeployDataService(serviceId, serviceVersion)
		if err != nil {
			return err
		}
		log.Info("Undeployment done!")
	}

	return nil
}

func GetStatusMessage(nodeId string) statusMessage {
	knownManifests := manifest.GetKnownManifests()
	deviceParams := deviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := len(knownManifests)
	for _, manifest := range knownManifests {
		if manifest.Status == "SUCCESS" {
			actv_cnt++
		}
	}

	msg := statusMessage{
		Id:                 nodeId,
		Timestamp:          time.Now().UnixMilli(),
		Status:             "Available",
		ActiveServiceCount: actv_cnt,
		ServiceCount:       serv_cnt,
		ServicesStatus:     knownManifests,
		DeviceParams:       deviceParams,
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
