package com

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const topicRegistration = "Registration"
const topicCertificate = "Certificate"

var params struct {
	Broker          string
	StatusTopicName string
	PubClientId     string
	SubClientId     string
	NoTLS           bool
	Heartbeat       int
}

func SetParams(opt model.Params) {

	params.Broker = opt.Broker
	params.StatusTopicName = opt.StatusTopicName
	params.PubClientId = opt.PubClientId
	params.SubClientId = opt.SubClientId
	params.NoTLS = opt.NoTLS
	params.Heartbeat = opt.Heartbeat
}

var registered bool
var connected = false

var publisher mqtt.Client
var subscriber mqtt.Client

const registrationTimeout = 5

func RegisterNode() error {
	nodeId := config.GetNodeId()

	if nodeId == "" {
		log.Info("Registering node and downloading certificate and key ...")
		registered = false
		nodeId = uuid.New().String()
		config.SetNodeId(nodeId)
		var err error
		publisher, err = initBrokerChannel(params.PubClientId+"/"+nodeId+"/"+topicRegistration, false)
		if err != nil {
			return err
		}
		subscriber, err = initBrokerChannel(params.SubClientId+"/"+nodeId+"/"+topicCertificate, true)
		if err != nil {
			return err
		}
		for {
			err := publishMessage(topicRegistration)
			if err != nil {
				log.Errorln("Registration failed, gonna try again in", registrationTimeout, "seconds.", err.Error())
				time.Sleep(time.Second * registrationTimeout)
			} else {
				break
			}
		}
	} else {
		log.Info("Node already registered!")
		registered = true
	}

	return nil
}

func SendHeartbeat() error {
	log.Debug("Node registered >> ", registered, " | connected >> ", connected)
	defer time.Sleep(time.Second * time.Duration(params.Heartbeat))
	for !registered {
		log.Infoln("Node not registered. Gonna wait", registrationTimeout, "seconds...")
		time.Sleep(time.Second * registrationTimeout)
	}
	err := reconnectIfNecessary()
	if err != nil {
		return err
	}
	err = publishMessage(params.StatusTopicName)
	if err != nil {
		return err
	}
	return nil
}

func ConnectNode() error {
	var err error
	publisher, err = initBrokerChannel(params.PubClientId+"/"+config.GetNodeId(), false)
	if err != nil {
		return err
	}
	subscriber, err = initBrokerChannel(params.SubClientId+"/"+config.GetNodeId(), true)
	if err != nil {
		return err
	}

	connected = true
	return nil
}

func DisconnectNode() {
	log.Info("Disconnecting.....")
	if publisher != nil && publisher.IsConnected() {
		publisher.Disconnect(250)
		log.Debug("Publisher disconnected")
	}

	if subscriber != nil && subscriber.IsConnected() {
		subscriber.Disconnect(250)
		log.Debug("Subscriber disconnected")
	}
}

func initBrokerChannel(pubsubClientId string, isSubscribe bool) (mqtt.Client, error) {
	log.Debug("Client id >> ", pubsubClientId, " | subscription >> ", isSubscribe)

	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(params.Broker)
	channelOptions.SetClientID(pubsubClientId)
	channelOptions.SetDefaultPublishHandler(messagePubHandler)
	channelOptions.OnConnectionLost = connectLostHandler
	if isSubscribe {
		channelOptions.OnConnect = connectHandler
	}

	if !params.NoTLS {
		tlsconfig, err := newTLSConfig()
		if err != nil {
			return nil, err
		}
		channelOptions.SetTLSConfig(tlsconfig)
	}

	log.Debug("options >> ", channelOptions)
	log.Debug("Parsing done! | MQTT configuration done!")

	pubsubClient := mqtt.NewClient(channelOptions)
	if token := pubsubClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	} else {
		if isSubscribe {
			log.Debug("MQTT subscriber connected!")
		} else {
			log.Debug("MQTT Publisher connected!")
		}
	}

	return pubsubClient, nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	jsonParsed, err := gabs.ParseJSON(msg.Payload())
	if err != nil {
		log.Error("Error on parsing message: ", err)
		return
	}
	log.Debugln("Received message on topic:", msg.Topic(), "JSON:", *jsonParsed)

	if msg.Topic() == params.SubClientId+"/"+config.GetNodeId()+"/"+topicCertificate {
		certificateUrl := jsonParsed.Search(topicCertificate).Data().(string)
		keyUrl := jsonParsed.Search("PrivateKey").Data().(string)

		certificatePath, keyPath, err := handler.DownloadCertificates(certificateUrl, keyUrl)
		if err != nil {
			log.Error(err)
			return
		}

		config.SetCertPath(certificatePath, keyPath)
		registered = true
		log.Info("Node registration done | Certificates downloaded!")

	} else {
		operation := strings.Replace(msg.Topic(), params.SubClientId+"/"+config.GetNodeId()+"/", "", 1)

		err = handler.ProcessMessage(operation, msg.Payload())
		if err != nil {
			log.Error(err)
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected >> registered : ", registered)
	var topicName string
	topicName = params.SubClientId + "/" + config.GetNodeId() + "/" + topicCertificate
	if registered {
		topicName = params.SubClientId + "/" + config.GetNodeId() + "/+"
	}

	log.Debug("ON connect >> subscribes >> topicName : ", topicName)
	if token := c.Subscribe(topicName, 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost ", err)
}

func newTLSConfig() (*tls.Config, error) {
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

func publishMessage(msgType string) error {

	if !publisher.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	var fullTopic string
	var payload []byte
	var err error
	switch msgType {
	case topicRegistration:
		fullTopic = params.PubClientId + "/" + config.GetNodeId() + "/" + topicRegistration

		msg := handler.GetRegistrationMessage(config.GetNodeId(), config.GetNodeName())
		log.Debugln("Sending registration request.", ">> Body:", msg)
		payload, err = json.Marshal(msg)
		if err != nil {
			log.Fatal(err)
		}

	case params.StatusTopicName:
		fullTopic = params.PubClientId + "/" + config.GetNodeId() + "/" + params.StatusTopicName
		msg := handler.GetStatusMessage(config.GetNodeId())
		log.Debugln("Sending update >>", "Topic:", params.StatusTopicName, ">> Body:", msg)
		payload, err = json.Marshal(msg)
		if err != nil {
			log.Fatal(err)
		}

	default:
		return errors.New("Topic not supported")
	}

	log.Debugln("Publishing message >> ", fullTopic, string(payload))
	if token := publisher.Publish(fullTopic, 0, false, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func reconnectIfNecessary() error {
	// Attempt reconnect
	if !publisher.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	if !subscriber.IsConnected() {
		log.Infoln("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	return nil
}