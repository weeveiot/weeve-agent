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
			// TODO: Error handling?
			deploy.DeployManifest(thisManifest, topic_rcvd)

		} else if topic_rcvd == "redeploy" {

			var thisManifest = model.Manifest{}
			thisManifest.Manifest = *jsonParsed
			// TODO: Error handling?
			deploy.DeployManifest(thisManifest, topic_rcvd)

		} else if topic_rcvd == "stopservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				deploy.StopDataService(serviceId, serviceName)
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "startservice" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				deploy.StartDataService(serviceId, serviceName)
			} else {
				log.Error(err)
			}

		} else if topic_rcvd == "undeploy" {

			err := model.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceName := jsonParsed.Search("name").Data().(string)

				deploy.UndeployDataService(serviceId, serviceName)
			} else {
				log.Error(err)
			}

		}
	}
}

func GetStatusMessage(nodeId string) model.StatusMessage {
	manifests := jsonlines.Read(constants.ManifestFile, "", "", nil, false)

	var mani []model.ManifestStatus
	var deviceParams = model.DeviceParams{"10", "10", "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		log.Info("Record on manifests >> ", rec)
		mani = append(mani, model.ManifestStatus{rec["id"].(string), rec["version"].(string), rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if "SUCCESS" == rec["status"].(string) {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	return model.StatusMessage{nodeId, millis, "Available", actv_cnt, serv_cnt, mani, deviceParams}

}
