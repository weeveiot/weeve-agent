package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/config"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

const (
	TopicOrchestration = "orchestration/#"
	topicNodeStatus    = "nodestatus"
	topicAgentLogs     = "agentlogs"
	topicAppLogs       = "applogs"
	topicNodePublicKey = "nodePublicKey"
	TopicOrgPrivateKey = "orgKey"
	TopicNodeDelete    = "delete"
)

var mqttLogger log.Logger
var client mqtt.Client
var subscriptionsMap map[string]mqtt.MessageHandler

func SendHeartbeat(msg StatusMsg) error {
	nodeStatusTopic := topicNodeStatus + "/" + config.Params.NodeId
	log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	return publishMessage(nodeStatusTopic, msg, true)
}

func SendEdgeAppLogs(msg EdgeAppLogMsg) error {
	if len(msg.ContainerLogs) > 0 {
		edgeAppLogsTopic := topicAppLogs + "/" + config.Params.NodeId
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
	log.Debug("Connecting node...")

	subscriptionsMap = subscriptions

	err := createMqttClient()
	if err != nil {
		return traceutility.Wrap(err)
	}

	addMqttHookToLogs(log.DebugLevel)
	return nil
}

func DisconnectNode() error {
	log.Info("Disconnecting node...")
	if client.IsConnected() {
		err := sendDisconnectedStatus()
		if err != nil {
			return traceutility.Wrap(err)
		}
		client.Disconnect(250)
		log.Debug("MQTT client disconnected")
	}
	return nil
}

func createMqttClient() error {
	log.Debug("Creating MQTT client...")

	// Build the options for the mqtt client
	nodeStatusTopic := topicNodeStatus + "/" + config.Params.NodeId
	willPayload, err := json.Marshal(disconnectedMsg)
	if err != nil {
		return traceutility.Wrap(err)
	}

	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(config.Params.Broker)
	channelOptions.SetClientID(config.Params.NodeId)
	channelOptions.SetOnConnectHandler(onConnectHandler)
	channelOptions.SetConnectionLostHandler(connectLostHandler)
	channelOptions.SetWill(nodeStatusTopic, string(willPayload), 1, true)

	if !config.Params.NoTLS {
		channelOptions.SetUsername(config.Params.NodeId)
		channelOptions.SetPassword(config.Params.Password)
		tlsconfig, err := newTLSConfig()
		if err != nil {
			return traceutility.Wrap(err)
		}
		channelOptions.SetTLSConfig(tlsconfig)
	}

	log.Debugf("Starting MQTT client with options >> %+v", channelOptions)

	client = mqtt.NewClient(channelOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return traceutility.Wrap(token.Error())
	}

	return nil
}

func subscribeAndSetHandler(topic string, handler mqtt.MessageHandler) error {
	fullTopic := config.Params.NodeId + "/" + topic

	log.Debug("Subscribing to topic ", fullTopic)
	if token := client.Subscribe(fullTopic, 1, handler); token.Wait() && token.Error() != nil {
		mqttLogger.Error("Cannot subscribe to topic: ", token.Error())
	}

	return nil
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	mqttLogger.Warning("Connection lost. Error: ", err)
}

var onConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Debug("MQTT client is (re)connected")

	for topic, handler := range subscriptionsMap {
		err := subscribeAndSetHandler(topic, handler)
		if err != nil {
			log.Error(traceutility.Wrap(err))
		}
	}
}

func newTLSConfig() (*tls.Config, error) {
	log.Debug("MQTT root cert path >> ", config.Params.RootCertPath)

	certpool := x509.NewCertPool()
	rootCert, err := os.ReadFile(config.Params.RootCertPath)
	if err != nil {
		return nil, traceutility.Wrap(err)
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
		return traceutility.Wrap(err)
	}

	// sending with QoS of 1 to ensure that the message gets delivered
	if token := client.Publish(topic, 1, retained, payload); token.WaitTimeout(time.Second) {
		if token.Error() != nil {
			return traceutility.Wrap(token.Error())
		}
	} else {
		mqttLogger.Error("Timeout! Message not published! msg: ", string(payload))
	}

	return nil
}

func CreateMQTTLogger(out io.Writer, formatter log.Formatter, level log.Level) {
	mqttLogger = log.Logger{
		Out:       out,
		Formatter: formatter,
		Level:     level,
	}
}
