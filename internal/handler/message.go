package handler

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util/jsonlines"
)

func ProcessMessage(operation string, payload []byte) error {
	log.Info("Processing the message >> ", operation)

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		return err
	}
	log.Debug("Parsed JSON >> ", jsonParsed)

	if operation == "deploy" {

		var manifest = model.Manifest{}
		manifest.Manifest = *jsonParsed
		err := dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Deployment done!")

	} else if operation == "redeploy" {

		var manifest = model.Manifest{}
		manifest.Manifest = *jsonParsed
		err := dataservice.DeployDataService(manifest, operation)
		if err != nil {
			return err
		}
		log.Info("Redeployment done!")

	} else if operation == "stopservice" {

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

	} else if operation == "startservice" {

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

	} else if operation == "undeploy" {

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

func GetStatusMessage(nodeId string) (model.StatusMessage, error) {
	manifests, err := jsonlines.Read(dataservice.ManifestFile, nil, false)
	if err != nil {
		return model.StatusMessage{}, err
	}

	var mani []model.ManifestStatus
	var deviceParams = model.DeviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		mani = append(mani, model.ManifestStatus{ManifestId: rec["id"].(string), ManifestVersion: rec["version"].(string), Status: rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if rec["status"].(string) == "SUCCESS" {
			actv_cnt = actv_cnt + 1
		}
	}

	msg := model.StatusMessage{
		Id:                 nodeId,
		Timestamp:          time.Now().UnixMilli(),
		Status:             "Available",
		ActiveServiceCount: actv_cnt,
		ServiceCount:       serv_cnt,
		ServicesStatus:     mani,
		DeviceParams:       deviceParams,
	}
	return msg, nil
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
