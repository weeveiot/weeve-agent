package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/handler"
)

var Params struct {
	Broker      string
	TopicName   string
	PubClientId string
	SubClientId string
	NoTLS       bool
	Heartbeat   int
}

var registered bool
var connected = false

var publisher mqtt.Client
var subscriber mqtt.Client

func RegisterNode() {
	nodeId := config.GetNodeId()

	if nodeId == "" {
		log.Info("Registering node and downloading certificate and key ...")
		registered = false
		nodeId = uuid.New().String()
		config.SetNodeId(nodeId)
		publisher = InitBrokerChannel(Params.PubClientId+"/"+nodeId+"/Registration", false)
		subscriber = InitBrokerChannel(Params.SubClientId+"/"+nodeId+"/Certificate", true)
		for {
			published := PublishMessages("Registration")
			if published {
				break
			}
			time.Sleep(time.Second * 5)
		}
	} else {
		log.Info("Node already registered!")
		registered = true
	}
}

func NodeHeartbeat() {
	log.Debug("Node registered >> ", registered, " | connected >> ", connected)
	if registered {
		ConnectNode()
		ReconnectIfNecessary()
		PublishMessages("All")
		time.Sleep(time.Second * time.Duration(Params.Heartbeat))
	} else {
		time.Sleep(time.Second * 5)
	}
}

func ConnectNode() {
	if !connected {
		DisconnectNode()
		publisher = InitBrokerChannel(Params.PubClientId+"/"+config.GetNodeId(), false)
		subscriber = InitBrokerChannel(Params.SubClientId+"/"+config.GetNodeId(), true)
		connected = true
	}
}

func InitBrokerChannel(pubsubClientId string, isSubscribe bool) mqtt.Client {
	log.Debug("Client id >> ", pubsubClientId, " | subscription >> ", isSubscribe)

	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(Params.Broker)
	channelOptions.SetClientID(pubsubClientId)
	channelOptions.SetDefaultPublishHandler(messagePubHandler)
	channelOptions.OnConnectionLost = connectLostHandler
	if isSubscribe {
		channelOptions.OnConnect = connectHandler
	}

	// Optionally add the TLS configuration to the 2 client options
	if !Params.NoTLS {
		tlsconfig, err := NewTLSConfig()
		if err != nil {
			log.Fatalf("failed to create TLS configuration: %v", err)
		}
		channelOptions.SetTLSConfig(tlsconfig)
		log.Debug("TLS set on options.")
	}

	log.Debug("options >> ", channelOptions)

	log.Debug("Parsing done! | MQTT configuration done!")

	pubsubClient := mqtt.NewClient(channelOptions)
	if token := pubsubClient.Connect(); token.Wait() && token.Error() != nil {
		if isSubscribe {
			log.Fatalf("failed to create subscriber connection: %v", token.Error())
		} else {
			log.Fatalf("failed to create publisher connection: %v", token.Error())
		}
	} else {
		if isSubscribe {
			log.Debug("MQTT subscriber connected!")
		} else {
			log.Debug("MQTT Publisher connected!")
		}
	}

	return pubsubClient
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	jsonParsed, err := gabs.ParseJSON(msg.Payload())
	if err != nil {
		log.Error("Error on parsing message: ", err)
	}
	log.Infoln("Received message on topic:", msg.Topic(), "JSON:", *jsonParsed)

	if msg.Topic() == Params.SubClientId+"/"+config.GetNodeId()+"/Certificate" {
		certificateUrl := jsonParsed.Search("Certificate").Data().(string)
		keyUrl := jsonParsed.Search("PrivateKey").Data().(string)
		certificatePath, keyPath := handler.DownloadCertificates(certificateUrl, keyUrl)
		config.SetCertPath(certificatePath, keyPath)
		registered = true
		log.Info("Node registration done | Certificates downloaded!")

	} else {
		operation := strings.Replace(msg.Topic(), Params.SubClientId+"/"+config.GetNodeId()+"/", "", 1)

		handler.ProcessMessage(operation, msg.Payload(), false)
	}
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected >> registered : ", registered)
	var topicName string
	topicName = Params.SubClientId + "/" + config.GetNodeId() + "/Certificate"
	if registered {
		topicName = Params.SubClientId + "/" + config.GetNodeId() + "/+"
	}

	log.Debug("ON connect >> subscribes >> topicName : ", topicName)
	if token := c.Subscribe(topicName, 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost ", err)
}

func NewTLSConfig() (*tls.Config, error) {
	log.Debug("MQTT root cert path >> ", config.GetRootCertPath())

	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(config.GetRootCertPath())
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	log.Debug("MQTT cert path >> ", config.GetCertPath())
	log.Debug("MQTT key path >> ", config.GetKeyPath())

	cert, err := tls.LoadX509KeyPair(config.GetCertPath(), config.GetKeyPath())
	if err != nil {
		return nil, err
	}

	configTLS := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      certpool,
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		Certificates: []tls.Certificate{cert},
	}
	return configTLS, nil
}

func PublishMessages(msgType string) bool {

	if !publisher.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			log.Error("failed to create publisher connection: ", token.Error())
			return false
		}
	}

	var topicNm string
	var b_msg []byte
	var err error
	if msgType == "Registration" {
		topicNm = Params.PubClientId + "/" + config.GetNodeId() + "/" + "Registration"

		msg := handler.GetRegistrationMessage(config.GetNodeId(), config.GetNodeName())
		log.Infoln("Sending registration request.", "Registration", msg)
		b_msg, err = json.Marshal(msg)
		if err != nil {
			log.Fatalf("Marshall error: %v", err)
		}

	} else {
		topicNm = Params.PubClientId + "/" + config.GetNodeId() + "/" + Params.TopicName
		msg := handler.GetStatusMessage(config.GetNodeId())
		log.Info("Sending update >> ", "Topic: ", Params.TopicName, " >> Body: ", msg)
		b_msg, err = json.Marshal(msg)
		if err != nil {
			log.Fatalf("Marshall error: %v", err)
		}
	}

	log.Debugln("Publishing message >> ", topicNm, string(b_msg))
	if token := publisher.Publish(topicNm, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
		return false
	}
	return true

}

func ReconnectIfNecessary() {
	// Attempt reconnect
	if !publisher.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			log.Error("failed to create publisher connection: ", token.Error())
		}
	}

	if !subscriber.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
			log.Error("failed to create subscriber connection: ", token.Error())
		}
	}
}

func DisconnectNode() {
	if publisher != nil && publisher.IsConnected() {
		log.Info("Disconnecting.....")
		publisher.Disconnect(250)
	}

	if subscriber != nil && subscriber.IsConnected() {
		log.Info("Disconnecting.....")
		subscriber.Disconnect(250)
	}
}
