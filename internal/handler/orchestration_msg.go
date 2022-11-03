package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

var OrchestrationHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := ProcessOrchestrationMessage(msg.Payload())
	if err != nil {
		log.Error("Failed to process orchestration message failed! CAUSE --> ", err)
	}
}

func ProcessOrchestrationMessage(payload []byte) error {
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
		err = dataservice.DeployDataService(manifest)
		if err != nil {
			return err
		}
		log.Info("Deployment done!")

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

	case dataservice.CMDResumeService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.ResumeDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Service resumed!")

	case dataservice.CMDUndeploy:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.UndeployDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Undeployment done!")

	case dataservice.CMDRemove:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return err
		}
		err = dataservice.RemoveDataService(manifestUniqueID)
		if err != nil {
			return err
		}
		log.Info("Full removal done!")

	default:
		return errors.New("received message with unknown command")
	}

	return nil
}
