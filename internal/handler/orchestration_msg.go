package handler

import (
	"errors"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/edgeapp"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

var OrchestrationHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := ProcessOrchestrationMessage(msg.Payload())
	if err != nil {
		log.Error("Failed to process orchestration message! CAUSE --> ", err)
	}
}

func ProcessOrchestrationMessage(payload []byte) error {
	operation, err := manifest.GetCommand(payload)
	if err != nil {
		return traceutility.Wrap(err)
	}
	log.Infoln("Processing the", operation, "message")

	switch operation {
	case edgeapp.CMDDeploy:
		manifest, err := manifest.Parse(payload)
		if err != nil {
			return traceutility.Wrap(err)
		}
		err = edgeapp.DeployEdgeApp(manifest)
		if err != nil {
			return traceutility.Wrap(err)
		}
		log.Info("Deployment done!")

	case edgeapp.CMDStopService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return traceutility.Wrap(err)
		}
		err = edgeapp.StopEdgeApp(manifestUniqueID)
		if err != nil {
			return traceutility.Wrap(err)
		}
		log.Info("Service stopped!")

	case edgeapp.CMDResumeService:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return traceutility.Wrap(err)
		}
		err = edgeapp.ResumeEdgeApp(manifestUniqueID)
		if err != nil {
			return traceutility.Wrap(err)
		}
		log.Info("Service resumed!")

	case edgeapp.CMDUndeploy:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return traceutility.Wrap(err)
		}
		err = edgeapp.UndeployEdgeApp(manifestUniqueID)
		if err != nil {
			return traceutility.Wrap(err)
		}
		log.Info("Undeployment done!")

	case edgeapp.CMDRemove:
		manifestUniqueID, err := manifest.GetEdgeAppUniqueID(payload)
		if err != nil {
			return traceutility.Wrap(err)
		}
		err = edgeapp.RemoveEdgeApp(manifestUniqueID, nil)
		if err != nil {
			return traceutility.Wrap(err)
		}
		log.Info("Full removal done!")

	default:
		return errors.New("received message with unknown command")
	}

	return nil
}
