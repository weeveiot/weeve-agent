package internal

import (
	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/deploy"
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
	}

	//TODO: Error return type?
}
