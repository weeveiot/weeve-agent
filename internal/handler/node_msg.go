package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var NodeDeleteHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))
	DeleteNode()
}

func DeleteNode() {
	err := dataservice.UndeployAll()
	if err != nil {
		log.Error(err)
	}

	dataservice.SetNodeStatus(model.NodeDeleted)
	dataservice.SendStatus()
}
