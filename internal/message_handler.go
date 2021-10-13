package internal

import (
	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"time"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/deploy"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

func ProcessMessage(topic_rcvd string, payload []byte) {
	log.Info(" ProcessMessage topic_rcvd ", topic_rcvd)

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message: ", err)
	} else {
		log.Debug("Parsed JSON >> ", jsonParsed)

		if topic_rcvd == "CheckVersion" {

		} else if topic_rcvd == "deploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			status := deploy.DeployManifest(thisManifest, topic_rcvd)
			if !status {
				log.Error("Error on deploy manifest")
			} else {
				log.Info("Manifest deployed successfully")
			}

		} else if topic_rcvd == "redeploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			status := deploy.DeployManifest(thisManifest, topic_rcvd)
			if !status {
				log.Error("Error on redeploy manifest")
			} else {
				log.Info("Manifest redeployed successfully")
			}

		} else if topic_rcvd == "stopservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				status := deploy.StopDataService(serviceId, serviceName)
				if !status {
					log.Error("Error on stop service")
				} else {
					log.Info("Service stopped!")
				}
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "startservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				status := deploy.StartDataService(serviceId, serviceName)
				if !status {
					log.Error("Error on start service")
				} else {
					log.Info("Service started!")
				}
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "undeploy" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				status := deploy.UndeployDataService(serviceId, serviceName)
				if !status {
					log.Error("Error on undeploy service")
				} else {
					log.Info("Service undeployed!")
				}
			} else {
				log.Error(err)
			}

		}
	}
}

func GetStatusMessage(nodeId string) model.StatusMessage {
	manifests := jsonlines.Read(constants.ManifestFile, "", "", nil, false)

	var mani []model.ManifestStatus
	var deviceParams = model.DeviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		log.Info("Record on manifests >> ", rec)
		mani = append(mani, model.ManifestStatus{ManifestId: rec["id"].(string), ManifestVersion: rec["version"].(string), Status: rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if rec["status"].(string) == "SUCCESS" {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	return model.StatusMessage{Id: nodeId, Timestamp: millis, Connectivity: "Available", ActiveServiceCount: actv_cnt, ServiceCount: serv_cnt, DeployStatus: mani, DeviceParams: deviceParams}

}
