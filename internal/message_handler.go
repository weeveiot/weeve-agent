package internal

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/deploy"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util/jsonlines"
)

func ProcessMessage(operation string, payload []byte, retry bool) {
	// flag for exception handling
	exception := true
	defer func() {
		if exception && retry {
			// on exception sleep 5s and try again
			time.Sleep(5 * time.Second)
			ProcessMessage(operation, payload, false)
		}
	}()

	log.Info("Processing the message >> ", operation)

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message : ", err)
	} else {
		log.Debug("Parsed JSON >> ", jsonParsed)

		if operation == "CheckVersion" {

		} else if operation == "deploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			err := deploy.DeployDataService(thisManifest, operation, false)
			if err != nil {
				log.Info("Deployment failed! CAUSE --> ", err, "!")
			} else {
				log.Info("Deployment done!")

			}

		} else if operation == "redeploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			err := deploy.DeployDataService(thisManifest, operation, false)
			if err != nil {
				log.Info("Redeployment failed! CAUSE --> ", err, "!")
			} else {
				log.Info("Redeployment done!")

			}

		} else if operation == "stopservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := deploy.StopDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Service stopped!")
				}
			}

		} else if operation == "startservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := deploy.StartDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Service started!")
				}
			}

		} else if operation == "undeploy" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := deploy.UndeployDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Undeployment done!")
				}
			}
		}
	}

	exception = false
}

func GetStatusMessage(nodeId string) model.StatusMessage {
	manifests, err := jsonlines.Read(deploy.ManifestFile, nil, false)

	if err != nil {
		return model.StatusMessage{}
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

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	return model.StatusMessage{Id: nodeId, Timestamp: millis, Status: "Available", ActiveServiceCount: actv_cnt, ServiceCount: serv_cnt, ServicesStatus: mani, DeviceParams: deviceParams}
}

func GetRegistrationMessage(nodeId string, nodeName string) model.RegistrationMessage {
	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000

	return model.RegistrationMessage{Id: nodeId, Timestamp: millis, Status: "Registering", Operation: "Registration", Name: nodeName}
}
