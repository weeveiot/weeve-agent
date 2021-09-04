package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
)

type Params struct {
	NodeId       string `long:"nodeId" short:"i" description:"ID of this node" required:"true"`
	Verbose      []bool `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker       string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	PubClientId  string `long:"pubClientId" short:"c" description:"Publisher ClientId" required:"true"`
	SubClientId  string `long:"subClientId" short:"s" description:"Subscriber ClientId" required:"true"`
	TopicName    string `long:"publish" short:"t" description:"Topic Name" required:"true"`
	RootCertPath string `long:"rootcert" short:"r" description:"Path to MQTT broker (server) certificate" required:"false"`
	CertPath     string `long:"cert" short:"f" description:"Path to certificate to authenticate to Broker" required:"false"`
	KeyPath      string `long:"key" short:"k" description:"Path to private key to authenticate to Broker" required:"false"`
	Heartbeat    int    `long:"heartbeat" short:"h" description:"Heartbeat time in seconds" required:"false" default:"30"`
	NoTLS        bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
}

var opt Params
var parser = flags.NewParser(&opt, flags.Default)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
	log.Info("Started logging")
}

func NewTLSConfig(CertPath string) (config *tls.Config, err error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(opt.RootCertPath)
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(opt.CertPath, opt.KeyPath)
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

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info(" messagePubHandler Received message on topic: ", msg.Topic(), "\nMessage: %s\n", msg.Payload())

	topic_rcvd := ""

	if strings.HasPrefix(msg.Topic(), opt.SubClientId+"/"+opt.NodeId+"/") {
		topic_rcvd = strings.Replace(msg.Topic(), opt.SubClientId+"/"+opt.NodeId+"/", "", 1)
	}

	internal.ProcessMessage(topic_rcvd, msg.Payload())

}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected")
	if token := c.Subscribe(opt.SubClientId+"/"+opt.NodeId+"/+", 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("subscribe connection: %v", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost", err)
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if _, err := parser.Parse(); err != nil {
		log.Error("Error on flgas parser ", err)
		os.Exit(1)
	}

	if len(opt.Verbose) >= 1 {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Logging level set to ", log.GetLevel())

	log.Debug("Broker: ", opt.Broker)
	log.Debug("NodeId: ", opt.NodeId)
	log.Debug("Heartbeat time: ", opt.Heartbeat)
	if opt.NoTLS {
		log.Info("TLS disabled!")
	} else {
		log.Debug("Root server certificate: ", opt.RootCertPath)
		log.Debug("Client certificate: ", opt.CertPath)
		log.Debug("Client private key: ", opt.KeyPath)
	}

	statusPublishTopic := opt.PubClientId + "/" + opt.NodeId
	log.Debug("Status heartbeat publishing to topic: ", opt.TopicName)

	nodeSubscribeTopic := opt.SubClientId + "/" + opt.NodeId
	log.Debug("This node is subscribed to topic: ", nodeSubscribeTopic)

	// Build the options for the publish client
	publisherOptions := mqtt.NewClientOptions()
	publisherOptions.AddBroker(opt.Broker)
	publisherOptions.SetClientID(statusPublishTopic)
	publisherOptions.SetDefaultPublishHandler(messagePubHandler)
	publisherOptions.OnConnectionLost = connectLostHandler

	// Build the options for the subscribe client
	subscriberOptions := mqtt.NewClientOptions()
	subscriberOptions.AddBroker(opt.Broker)
	subscriberOptions.SetClientID(nodeSubscribeTopic)
	subscriberOptions.SetDefaultPublishHandler(messagePubHandler)
	subscriberOptions.OnConnectionLost = connectLostHandler
	// sub_opts.SetReconnectingHandler(messagePubHandler, opts)
	subscriberOptions.OnConnect = connectHandler

	if !opt.NoTLS {
		tlsconfig, err := NewTLSConfig(opt.CertPath)
		if err != nil {
			log.Fatalf("failed to create TLS configuration: %v", err)
		}
		// log.Debug("Tls Config >> ", tlsconfig)
		// subscriberOptions.SetTLSConfig(tlsconfig)
		publisherOptions.SetTLSConfig(tlsconfig)
	}

	// log.Debug("Info on Sub & Pub >> ", subscriberOptions, publisherOptions)

	publisher := mqtt.NewClient(publisherOptions)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		log.Error("failed to create publisher connection: %v", token.Error())
	}
	// log.Debug("MQTT publisher client: \n", publisher)
	log.Debug("MQTT publisher connected")

	subscriber := mqtt.NewClient(subscriberOptions)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		log.Error("failed to create subscriber connection: %v", token.Error())
	}
	log.Debug("MQTT subscriber connected")
	// log.Debug("MQTT subscriber client: \n", subscriber)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			if !publisher.IsConnected() {
				log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

				if token := publisher.Connect(); token.Wait() && token.Error() != nil {
					log.Error("failed to create publisher connection: %v", token.Error())
				}
			}

			if !subscriber.IsConnected() {
				log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

				if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
					log.Error("failed to create subscriber connection: %v", token.Error())
				}
			}

			PublishMessages(publisher)

			log.Info("Sleeping ", opt.Heartbeat)
			time.Sleep(time.Second * time.Duration(opt.Heartbeat))
		}
	}()
	<-done

	<-c
	if publisher.IsConnected() {
		log.Info("Disconnecting.....")
		publisher.Disconnect(250)
	}

	if subscriber.IsConnected() {
		log.Info("Disconnecting.....")
		subscriber.Disconnect(250)
	}
}

func PublishMessages(cl mqtt.Client) {

	manifests := jsonlines.Read(constants.ManifestFile, "", "", nil, false)

	var mani []model.ManifestStatus
	var deviceParams = model.DeviceParams{"10", "10", "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		log.Info("Record on manifests >> ", rec)
		mani = append(mani, model.ManifestStatus{rec["id"].(string), rec["version"].(string), rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if "SUCCESS" == rec["status"].(string) {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	msg := model.StatusMessage{opt.NodeId, millis, "Available", actv_cnt, serv_cnt, mani, deviceParams}

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending update.", opt.TopicName, msg, string(b_msg))
	if token := cl.Publish(opt.PubClientId+"/"+opt.NodeId+"/"+opt.TopicName, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}
}

// func post(jsonReq []byte, nextHost string) {
// 	fmt.Printf("Next host %s", nextHost)
// 	resp, err := http.Post(nextHost, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
// 	if err != nil {
// 		log.Info("Post API Connection error: %v", err)
// 	} else {

// 		defer resp.Body.Close()
// 		bodyBytes, _ := ioutil.ReadAll(resp.Body)

// 		// Convert response body to string
// 		bodyString := string(bodyBytes)
// 		log.Info(bodyString)
// 	}
// }
