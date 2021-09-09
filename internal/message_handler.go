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

	if topic_rcvd == "CheckVersion" {

	} else if topic_rcvd == "deploy" {

		jsonParsed, err := gabs.ParseJSON(payload)
		if err != nil {
			log.Error("Error on parsing message: ", err)
		} else {
			log.Debug("Parsed JSON >> ", jsonParsed)

			var thisManifest = model.Manifest{}

			thisManifest.Manifest = *jsonParsed

			// TODO: Error handling?
			deploy.DeployManifest(thisManifest)
		}

	} else if topic_rcvd == "StopService" {

		// expects JSON with field { name: serviceName }

		jsonParsed, err := gabs.ParseJSON(payload)
		if err != nil {
			log.Error("Error on parsing message: ", err)
		} else {
			log.Debug("Parsed JSON >> ", jsonParsed)

			serviceName := jsonParsed.Search("name").Data().(string)

			deploy.StopDataService(serviceName)
		}

	} else if topic_rcvd == "StartService" {

		// expects JSON with field { name: serviceName }

		jsonParsed, err := gabs.ParseJSON(payload)
		if err != nil {
			log.Error("Error on parsing message: ", err)
		} else {
			log.Debug("Parsed JSON >> ", jsonParsed)

			serviceName := jsonParsed.Search("name").Data().(string)

			deploy.StartDataService(serviceName)
		}

	}

	//TODO: Error return type?
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
