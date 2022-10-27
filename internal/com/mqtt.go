package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
)

const (
	TopicOrchestration = "orchestration"
	topicNodeStatus    = "nodestatus"
	topicLogs          = "debug"
	topicNodePublicKey = "nodePublicKey"
	TopicOrgPrivateKey = "orgKey"
	TopicNodeDelete    = "delete"
)

var client mqtt.Client

func SendHeartbeat(msg StatusMsg) error {
	nodeStatusTopic := topicNodeStatus + "/" + config.Params.NodeId
	log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	return publishMessage(nodeStatusTopic, msg, true)
}

func SendEdgeAppLogs(msg EdgeAppLogMsg) error {
	if len(msg.ContainerLogs) > 0 {
		edgeAppLogsTopic := config.Params.NodeId + "/" + msg.ManifestID + "/" + topicLogs
		log.Debugln("Sending edge app logs >>", "Topic:", edgeAppLogsTopic, ">> Body:", msg)
		return publishMessage(edgeAppLogsTopic, msg, false)
	}

	return nil
}

func SendNodePublicKey(nodePublicKey []byte) error {
	topic := topicNodePublicKey + "/" + config.Params.NodeId
	msg := nodePublicKeyMsg{
		NodePublicKey: string(nodePublicKey),
	}
	log.Debugln("Sending nodePublicKey >>", "Topic:", topic, ">> Body:", msg)
	return publishMessage(topic, msg, true)
}

func sendDisconnectedStatus() error {
	nodeStatusTopic := topicNodeStatus + "/" + config.Params.NodeId
	msg := disconnectedMsg
	log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	return publishMessage(nodeStatusTopic, msg, true)
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

	addMqttHookToLogs(log.DebugLevel)
	return nil
}

func DisconnectNode() error {
	log.Info("Disconnecting.....")
	if client.IsConnected() {
		err := sendDisconnectedStatus()
		if err != nil {
			return err
		}
		client.Disconnect(250)
		log.Debug("MQTT client disconnected")
	}
	return nil
}

func createMqttClient() error {
	// Build the options for the mqtt client
	nodeStatusTopic := topicNodeStatus + "/" + config.Params.NodeId
	willPayload, err := json.Marshal(disconnectedMsg)
	if err != nil {
		return err
	}

	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(config.Params.Broker)
	channelOptions.SetClientID(config.Params.NodeId)
	channelOptions.SetConnectionLostHandler(connectLostHandler)
	channelOptions.SetWill(nodeStatusTopic, string(willPayload), 1, true)

	if !config.Params.NoTLS {
		channelOptions.SetUsername(config.Params.NodeId)
		channelOptions.SetPassword(config.Params.Password)
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
	fullTopic := config.Params.NodeId + "/" + topic

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
	log.Debug("MQTT root cert path >> ", config.Params.RootCertPath)

	certpool := x509.NewCertPool()
	rootCert, err := os.ReadFile(config.Params.RootCertPath)
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
		return err
	}

	// sending with QoS of 1 to ensure that the message gets delivered
	if token := client.Publish(topic, 1, retained, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
