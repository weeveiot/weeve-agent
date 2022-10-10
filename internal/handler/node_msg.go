package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
)

var NodeDeleteHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := dataservice.UndeployAll()
	if err != nil {
		log.Error(err)
	}
}
