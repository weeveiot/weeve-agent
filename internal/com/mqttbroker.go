package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const (
	TopicOrchestration = "orchestration"
	topicNodeStatus    = "nodestatus"
	topicEdgeAppLogs   = "debug"
	topicNodePublicKey = "nodePublicKey"
	TopicOrgPrivateKey = "orgKey"
)

var client mqtt.Client
var params struct {
	Broker    string
	NoTLS     bool
	Heartbeat int
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

func SendHeartbeat(msg StatusMsg) error {
	nodeStatusTopic := topicNodeStatus + "/" + config.GetNodeId()
	log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	err := publishMessage(nodeStatusTopic, msg, false)
	if err != nil {
		return err
	}

	return nil
}

func SendEdgeAppLogs(msg EdgeAppLogMsg) error {
	if len(msg.ContainerLogs) > 0 {
		edgeAppLogsTopic := config.GetNodeId() + "/" + msg.ManifestID + "/" + topicEdgeAppLogs
		log.Debugln("Sending edge app logs >>", "Topic:", edgeAppLogsTopic, ">> Body:", msg)
		err := publishMessage(edgeAppLogsTopic, msg, false)
		if err != nil {
			log.Errorln("Failed to publish logs", ">> Topic:", edgeAppLogsTopic, ">> Error:", err)
			return err
		}
	}

	return nil
}

func SendNodePublicKey(nodePublicKey []byte) error {
	topic := topicNodePublicKey + "/" + config.GetNodeId()
	msg := nodePublicKeyMsg{
		NodePublicKey: string(nodePublicKey),
	}
	log.Debugln("Sending nodePublicKey >>", "Topic:", topic, ">> Body:", msg)
	return publishMessage(topic, msg, true)
}

func ConnectNode(subscriptions map[string]mqtt.MessageHandler) error {
	err := createMqttClient()
	if err != nil {
		return err
	}

	for topic, handler := range subscriptions {
		err = subscribeAndSetHandler(topic, handler)
		if err != nil {
			return err
		}
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

func publishMessage(topic string, message interface{}, retained bool) error {
	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugln("Publishing message >> Topic:", topic, ">> Payload:", string(payload))
	// sending with QoS of 1 to ensure that the message gets delivered
	if token := client.Publish(topic, 1, retained, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
