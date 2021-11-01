package internal

import (
	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/deploy"
)

func ProcessMessage(topic_rcvd string, payload []byte, retry bool) {
	// flag for exception handling
	exception := true
	defer func() {
		if exception && retry {
			// on exception sleep 5s and try again
			time.Sleep(5 * time.Second)
			ProcessMessage(topic_rcvd, payload, false)
		}
	}()

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
			if status {
				log.Info("Manifest deployed successfully")
			}

		} else if topic_rcvd == "redeploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			status := deploy.DeployManifest(thisManifest, topic_rcvd)
			if status {
				log.Info("Manifest redeployed successfully")
			}

		} else if topic_rcvd == "stopservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				status := deploy.StopDataService(serviceId, serviceVersion)
				if status {
					log.Info("Service stopped!")
				}
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "startservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				status := deploy.StartDataService(serviceId, serviceVersion)
				if status {
					log.Info("Service started!")
				}
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "undeploy" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				status := deploy.UndeployDataService(serviceId, serviceVersion)
				if status {
					log.Info("Service undeployed!")
				}
			} else {
				log.Error(err)
			}
		}
	}

	exception = false
}
