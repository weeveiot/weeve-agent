package internal

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/handler"
)

var Registered bool

var Broker string
var PubClientId string
var SubClientId string
var TopicName string
var NoTLS bool
var NodeId string

func InitBrokerChannel(nodeConfig map[string]string, pubsubClientId string, isSubscribe bool) mqtt.Client {

	// var pubsubClient mqtt.Client

	log.Debug("Client id >> ", pubsubClientId, " | subscription >> ", isSubscribe)

	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(Broker)
	channelOptions.SetClientID(pubsubClientId)
	channelOptions.SetDefaultPublishHandler(messagePubHandler)
	channelOptions.OnConnectionLost = connectLostHandler
	if isSubscribe {
		channelOptions.OnConnect = connectHandler
	}

	// Optionally add the TLS configuration to the 2 client options
	if !NoTLS {
		tlsconfig, err := NewTLSConfig(nodeConfig)
		if err != nil {
			log.Fatalf("failed to create TLS configuration: %v", err)
		}
		// log.Debug("Tls Config >> ", tlsconfig)
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
		log.Infoln("Received message on topic: ", msg.Topic(), *jsonParsed)
	}

	topic_rcvd := ""

	if msg.Topic() == SubClientId+"/"+NodeId+"/Certificate" {
		certificates := handler.DownloadCertificates(msg.Payload())
		if certificates != nil {
			time.Sleep(time.Second * 10)
			handler.MarkNodeRegistered(NodeId, certificates)
			Registered = true
			log.Info("Node registration done | Certificates downloaded!")
		}
	} else {
		if strings.HasPrefix(msg.Topic(), SubClientId+"/"+NodeId+"/") {
			topic_rcvd = strings.Replace(msg.Topic(), SubClientId+"/"+NodeId+"/", "", 1)
		}

		handler.ProcessMessage(topic_rcvd, msg.Payload(), false)
	}
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected >> registered : ", Registered)
	var topicName string
	topicName = SubClientId + "/" + NodeId + "/Certificate"
	if Registered {
		topicName = SubClientId + "/" + NodeId + "/+"
	}

	log.Debug("ON connect >> subscribes >> topicName : ", topicName)
	if token := c.Subscribe(topicName, 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost", err)
}

func NewTLSConfig(nodeConfig map[string]string) (config *tls.Config, err error) {
	log.Debug("MQTT root cert path >> ", nodeConfig[handler.KeyAWSRootCert])

	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(nodeConfig[handler.KeyAWSRootCert])
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	log.Debug("MQTT cert path >> ", nodeConfig[handler.KeyCertificate])
	log.Debug("MQTT key path >> ", nodeConfig[handler.KeyPrivateKey])

	cert, err := tls.LoadX509KeyPair(nodeConfig[handler.KeyCertificate], nodeConfig[handler.KeyPrivateKey])
	if err != nil {
		return nil, err
	}

	config = &tls.Config{
		RootCAs:      certpool,
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		Certificates: []tls.Certificate{cert},
	}
	return config, nil
}

func PublishMessages(publisher mqtt.Client, pubNodeId string, nodeName string, msgType string) bool {

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
		topicNm = PubClientId + "/" + pubNodeId + "/" + "Registration"

		msg := handler.GetRegistrationMessage(pubNodeId, nodeName)
		log.Infoln("Sending registration request.", "Registration", msg)
		b_msg, err = json.Marshal(msg)
		if err != nil {
			log.Fatalf("Marshall error: %v", err)
		}

	} else {
		topicNm = PubClientId + "/" + pubNodeId + "/" + TopicName
		msg := handler.GetStatusMessage(pubNodeId)
		log.Info("Sending update >> ", "Topic: ", TopicName, " >> Body: ", msg)
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

func ReconnectIfNecessary(publisher mqtt.Client, subscriber mqtt.Client) {
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

func DisconnectBroker(publisher mqtt.Client, subscriber mqtt.Client) {
	if publisher != nil && publisher.IsConnected() {
		log.Info("Disconnecting.....")
		publisher.Disconnect(250)
	}

	if subscriber != nil && subscriber.IsConnected() {
		log.Info("Disconnecting.....")
		subscriber.Disconnect(250)
	}
}
