package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/secret"
)

var OrgPrivKeyHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := secret.ProcessOrgPrivKeyMessage(msg.Payload())
	if err != nil {
		log.Error(err)
	}
}
