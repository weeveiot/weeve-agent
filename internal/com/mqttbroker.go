package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/secret"
)

const (
	topicOrchestration = "orchestration"
	topicNodeStatus    = "nodestatus"
	topicEdgeAppLogs   = "debug"
	topicNodePublicKey = "nodePublicKey"
	topicOrgPrivateKey = "orgPrivateKey"
)

var client mqtt.Client
var params struct {
	Broker    string
	NoTLS     bool
	Heartbeat int
}

type nodePublicKeyMsg struct {
	nodePublicKey string
}

func SetParams(opt model.Params) {
	params.Broker = opt.Broker
	params.NoTLS = opt.NoTLS
	params.Heartbeat = opt.Heartbeat
	log.Debugf("Set the following MQTT params: %+v", params)
}

func GetHeartbeat() int {
	return params.Heartbeat
}

func SendHeartbeat() error {
	nodeStatusTopic := topicNodeStatus + "/" + config.GetNodeId()
	msg, err := handler.GetStatusMessage()
	if err != nil {
		return err
	}
	log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	err = publishMessage(nodeStatusTopic, msg)
	if err != nil {
		return err
	}

	return nil
}

func SendEdgeAppLogs() {
	log.Debug("Check if new logs available for edge apps")
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		if manif.Status != model.EdgeAppUndeployed {
			edgeAppLogsTopic := config.GetNodeId() + "/" + manif.ManifestID + "/" + topicEdgeAppLogs
			since := manif.LastLogReadTime
			until := time.Now().UTC().Format(time.RFC3339Nano)

			msg, err := dataservice.GetDataServiceLogs(manif, since, until)
			if err != nil {
				log.Errorln("GetDataServiceLogs failed", ">> ManifestID:", manif.ManifestID, ">> Error:", err)
			}

			if len(msg.ContainerLogs) > 0 {
				log.Debugln("Sending edge app logs >>", "Topic:", edgeAppLogsTopic, ">> Body:", msg)
				err = publishMessage(edgeAppLogsTopic, msg)
				if err != nil {
					log.Errorln("Failed to publish logs", ">> Topic:", edgeAppLogsTopic, ">> Error:", err)
				}
			}

			manifest.SetLastLogRead(manif.ManifestUniqueID, until)
		}
	}
}

func SendNodePublicKey(nodePublicKey []byte) error {
	topic := config.GetNodeId() + "/" + topicNodePublicKey
	msg := nodePublicKeyMsg{
		nodePublicKey: string(nodePublicKey),
	}
	return publishMessage(topic, msg)
}

func ConnectNode() error {
	err := createMqttClient()
	if err != nil {
		return err
	}

	err = subscribeAndSetHandler(topicOrchestration, orchestrationHandler)
	if err != nil {
		return err
	}

	err = subscribeAndSetHandler(topicOrgPrivateKey, orgPrivKeyHandler)
	if err != nil {
		return err
	}

	return nil
}

func DisconnectNode() {
	log.Info("Disconnecting.....")
	if client.IsConnected() {
		client.Disconnect(250)
		log.Debug("MQTT client disconnected")
	}
}

func createMqttClient() error {
	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(params.Broker)
	channelOptions.SetClientID(config.GetNodeId())
	channelOptions.SetConnectionLostHandler(connectLostHandler)

	if !params.NoTLS {
		channelOptions.SetUsername(config.GetNodeId())
		channelOptions.SetPassword(config.GetPassword())
		tlsconfig, err := newTLSConfig()
		if err != nil {
			return err
		}
		channelOptions.SetTLSConfig(tlsconfig)
	}

	log.Debug("Starting MQTT client with options >> ", channelOptions)

	client = mqtt.NewClient(channelOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Debug("MQTT client is connected")

	return nil
}

func subscribeAndSetHandler(topic string, handler mqtt.MessageHandler) error {
	fullTopic := config.GetNodeId() + "/" + topic

	log.Debug("Subscribing to topic ", fullTopic)
	if token := client.Subscribe(fullTopic, 1, handler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}

	return nil
}

var orchestrationHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := handler.ProcessOrchestrationMessage(msg.Payload())
	if err != nil {
		log.Error(err)
	}
}

var orgPrivKeyHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	err := secret.ProcessOrgPrivKeyMessage(msg.Payload())
	if err != nil {
		log.Error(err)
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Warning("Connection lost ", err)
}

func newTLSConfig() (*tls.Config, error) {
	log.Debug("MQTT root cert path >> ", config.GetRootCertPath())

	certpool := x509.NewCertPool()
	rootCert, err := os.ReadFile(config.GetRootCertPath())
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(rootCert)

	configTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    certpool,
		ClientAuth: tls.NoClientCert,
	}
	return configTLS, nil
}

func publishMessage(topic string, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugln("Publishing message >> Topic:", topic, ">> Payload:", string(payload))
	// sending with QoS of 1 to ensure that the message gets delivered
	if token := client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
