package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var NodeDeleteHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Infoln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))
	DeleteNode(model.NodeDeleted)
}

func DeleteNode(nodeStatus string) {
	log.Debug("Deleting node...")

	err := dataservice.RemoveAll()
	if err != nil {
		log.Error("Deletion of node failed! CAUSE --> ", err)
	}

	dataservice.SetNodeStatus(nodeStatus)
	dataservice.SendStatus()
}
