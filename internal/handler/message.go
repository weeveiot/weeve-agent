package handler

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func ProcessMessage(operation string, payload []byte) error {
	log.Info("Processing the message >> ", operation)

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		return err
	}
	log.Debug("Parsed JSON >> ", jsonParsed)

	switch operation {
	case "deploy":
		var err = model.ValidateManifest(jsonParsed)
		if err != nil {
			return err
		}

		manifest, err := model.GetManifest(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Deployment done!")

	case "redeploy":
		var err = model.ValidateManifest(jsonParsed)
		if err != nil {
			return err
		}

		manifest, err := model.GetManifest(jsonParsed)
		if err != nil {
			return err
		}
		err = dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Redeployment done!")

	case "stopservice":

		err := model.ValidateStartStopJSON(jsonParsed)
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

	case "startservice":

		err := model.ValidateStartStopJSON(jsonParsed)
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

	case "undeploy":

		err := model.ValidateStartStopJSON(jsonParsed)
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

func GetStatusMessage(nodeId string) model.StatusMessage {
	knownManifests := model.GetKnownManifests()
	deviceParams := model.DeviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := len(knownManifests)
	for _, manifest := range knownManifests {
		if manifest.Status == "SUCCESS" {
			actv_cnt++
		}
	}

	msg := model.StatusMessage{
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

func GetRegistrationMessage(nodeId string, nodeName string) model.RegistrationMessage {
	msg := model.RegistrationMessage{
		Id:        nodeId,
		Timestamp: time.Now().UnixMilli(),
		Status:    "Registering",
		Operation: "Registration",
		Name:      nodeName,
	}
	return msg
}
