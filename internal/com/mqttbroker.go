package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/logger"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const topicOrchestration = "orchestration"
const topicNodeStatus = "nodestatus"
const topicEdgeAppLogs = "debug"

var connected = false
var publisher mqtt.Client
var subscriber mqtt.Client
var params struct {
	Broker    string
	NoTLS     bool
	Heartbeat int
}

func SetParams(opt model.Params) {
	params.Broker = opt.Broker
	params.NoTLS = opt.NoTLS
	params.Heartbeat = opt.Heartbeat
	logger.Log.Debugf("Set the following MQTT params: %+v", params)
}

func GetHeartbeat() int {
	return params.Heartbeat
}

func SendHeartbeat() error {
	logger.Log.Debug("Node registered >> ", config.GetRegistered(), " | connected >> ", connected)
	err := reconnectIfNecessary()
	if err != nil {
		return err
	}

	nodeStatusTopic := topicNodeStatus + "/" + config.GetNodeId()
	msg, err := handler.GetStatusMessage()
	if err != nil {
		return err
	}
	logger.Log.Debugln("Sending update >>", "Topic:", nodeStatusTopic, ">> Body:", msg)
	err = publishMessage(nodeStatusTopic, msg)
	if err != nil {
		return err
	}

	return nil
}

func SendEdgeAppLogs() {
	logger.Log.Debugln("Check if new logs available for edge apps")
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		if manif.Status != model.EdgeAppUndeployed {
			edgeAppLogsTopic := config.GetNodeId() + "/" + manif.ManifestID + "/" + topicEdgeAppLogs
			since := manif.LastLogReadTime
			until := time.Now().UTC().Format(time.RFC3339Nano)

			msg, err := dataservice.GetDataServiceLogs(manif, since, until)
			if err != nil {
				logger.Log.Errorln("GetDataServiceLogs failed", ">> ManifestID:", manif.ManifestID, ">> Error:", err)
			}

			if len(msg.ContainerLogs) > 0 {
				logger.Log.Debugln("Sending edge app logs >>", "Topic:", edgeAppLogsTopic, ">> Body:", msg)
				err = publishMessage(edgeAppLogsTopic, msg)
				if err != nil {
					logger.Log.Errorln("Failed to publish logs", ">> Topic:", edgeAppLogsTopic, ">> Error:", err)
				}
			}

			manifest.SetLastLogRead(manif.ManifestUniqueID, until)
		}
	}
}

func ConnectNode() error {
	var err error
	publisher, err = initBrokerChannel(config.GetNodeId() + "_pub")
	if err != nil {
		return err
	}
	subscriber, err = initBrokerChannel(config.GetNodeId() + "_sub")
	if err != nil {
		return err
	}

	connected = true
	return nil
}

func DisconnectNode() {
	logger.Log.Info("Disconnecting.....")
	if publisher != nil && publisher.IsConnected() {
		publisher.Disconnect(250)
		logger.Log.Debug("Publisher disconnected")
	}

	if subscriber != nil && subscriber.IsConnected() {
		subscriber.Disconnect(250)
		logger.Log.Debug("Subscriber disconnected")
	}
}

func initBrokerChannel(pubsubClientId string) (mqtt.Client, error) {
	logger.Log.Debug("Client id >> ", pubsubClientId)

	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(params.Broker)
	channelOptions.SetClientID(pubsubClientId)
	channelOptions.SetDefaultPublishHandler(messagePubHandler)
	channelOptions.OnConnectionLost = connectLostHandler
	if strings.Contains(pubsubClientId, "sub") {
		channelOptions.OnConnect = connectHandler
	}

	if !params.NoTLS {
		channelOptions.SetUsername(config.GetNodeId())
		channelOptions.SetPassword(config.GetPassword())
		tlsconfig, err := newTLSConfig()
		if err != nil {
			return nil, err
		}
		channelOptions.SetTLSConfig(tlsconfig)
	}

	logger.Log.Debug("options >> ", channelOptions)
	logger.Log.Debug("Parsing done! | MQTT configuration done!")

	pubsubClient := mqtt.NewClient(channelOptions)
	if token := pubsubClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	} else {
		logger.Log.Debug(pubsubClientId, " connected!")
	}

	return pubsubClient, nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	logger.Log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))

	if msg.Topic() == config.GetNodeId()+"/"+topicOrchestration {
		err := handler.ProcessMessage(msg.Payload())
		if err != nil {
			logger.Log.Error(err)
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	logger.Log.Info("ON connect >> connected >> registered : ", config.GetRegistered())

	if config.GetRegistered() {
		topicName := config.GetNodeId() + "/" + topicOrchestration

		logger.Log.Debug("ON connect >> subscribes >> topicName : ", topicName)
		if token := c.Subscribe(topicName, 0, messagePubHandler); token.Wait() && token.Error() != nil {
			logger.Log.Error("Error on subscribe connection: ", token.Error())
		}
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	logger.Log.Info("Connection lost ", err)
}

func newTLSConfig() (*tls.Config, error) {
	logger.Log.Debug("MQTT root cert path >> ", config.GetRootCertPath())

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

	if !publisher.IsConnected() {
		logger.Log.Infoln("Connecting.....")

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	payload, err := json.Marshal(message)
	if err != nil {
		logger.Log.Fatal(err)
	}

	logger.Log.Debugln("Publishing message >> Topic:", topic, ">> Payload:", string(payload))
	if token := publisher.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func reconnectIfNecessary() error {
	// Attempt reconnect
	if !publisher.IsConnected() {
		logger.Log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	if !subscriber.IsConnected() {
		logger.Log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	return nil
}
