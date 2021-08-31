package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Jeffail/gabs/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

type Params struct {
	NodeId      string `long:"nodeId" short:"i" description:"ID of this node" required:"true"`
	Verbose     []bool `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker      string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	PubClientId string `long:"pubClientId" short:"c" description:"Publisher ClientId" required:"true"`
	SubClientId string `long:"subClientId" short:"s" description:"Subscriber ClientId" required:"true"`
	TopicName   string `long:"publish" short:"t" description:"Topic Name" required:"true"`
	Cert        string `long:"cert" short:"f" description:"Certificate to connect Broker" required:"false"`
	HostUrl     string `long:"publicurl" short:"u" description:"Public URL to connect from public" required:"false"`
	NodeApiPort string `long:"nodeport" short:"p" description:"Port where edge node api is listening" required:"true"`
	Heartbeat   int    `long:"heartbeat" description:"Heartbeat time in seconds" required:"false" default:"30"`
	NoTLS       bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
}

type StatusMessage struct {
	Id                 string           `json:"ID"`
	Timestamp          int64            `json:"timestamp"`
	Connectivity       string           `json:"connectivity"`
	ActiveServiceCount int              `json:"activeServiceCount"`
	ServiceCount       int              `json:"serviceCount"`
	DeployStatus       []ManifestStatus `json:"deployStatus"`
	DeviceParams       DeviceParams     `json:"deviceParams"`
}

type ManifestStatus struct {
	ManifestId      string `json:"manifestId"`
	ManifestVersion string `json:"manifestVersion"`
	Status          string `json:"status"`
}

type DeviceParams struct {
	Sensors string `json:"sensors"`
	Uptime  string `json:"uptime"`
	CpuTemp string `json:"cputemp"`
}

var opt Params
var parser = flags.NewParser(&opt, flags.Default)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
	log.Info("Started logging")
}

func NewTLSConfig(CertPrefix string) (config *tls.Config, err error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile("AmazonRootCA1.pem")
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(CertPrefix+"-certificate.pem.crt", CertPrefix+"-private.pem.key")
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

	if topic_rcvd == "CheckVersion" {

	} else if topic_rcvd == "deploy" {

		jsonParsed, err := gabs.ParseJSON(msg.Payload())
		if err != nil {
			log.Error(err)
		} else {
			log.Debug(jsonParsed)
		}

		post([]byte(msg.Payload()), "http://localhost:"+opt.NodeApiPort+"/pipelines")
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info(fmt.Println("Connected"))
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info(fmt.Printf("Connect lost: %v\n", err))
}

// var reconnectHandler mqtt.ReconnectHandler = func(client mqtt.Client, opts mqtt.ClientOptions) {
// 	fmt.Printf("ReConnect lost:")
// }

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if _, err := parser.Parse(); err != nil {
		log.Error(err)
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
	log.Debug("CertPrefix: ", opt.Cert)
	log.Debug("TopicName: ", opt.TopicName)
	log.Debug("Heartbeat time: ", opt.Heartbeat)
	if opt.NoTLS {
		log.Info("TLS disabled!")
	}

	// log.Debug(tlsconfig)
	// fmt.Println(tlsconfig)

	// Build the options for the publish client
	publisherOptions := mqtt.NewClientOptions()
	publisherOptions.AddBroker(opt.Broker)
	publisherOptions.SetClientID(opt.PubClientId)
	publisherOptions.SetDefaultPublishHandler(messagePubHandler)
	publisherOptions.OnConnectionLost = connectLostHandler
	// if !opt.NoTLS {
	// 	publisherOptions.SetTLSConfig(tlsconfig)
	// }
	// log.Debug(fmt.Sprintf("Publisher options: %+v\n", publisherOptions))

	// Build the options for the subscribe client
	subscriberOptions := mqtt.NewClientOptions()
	subscriberOptions.AddBroker(opt.Broker)
	subscriberOptions.SetClientID(opt.SubClientId)
	subscriberOptions.SetDefaultPublishHandler(messagePubHandler)
	subscriberOptions.OnConnectionLost = connectLostHandler
	// if !opt.NoTLS {
	// 	subscriberOptions.SetTLSConfig(tlsconfig)
	// }

	if !opt.NoTLS {
		tlsconfig, err := NewTLSConfig(opt.Cert)
		if err != nil {
			log.Fatalf("failed to create TLS configuration: %v", err)
		}
		log.Debug(tlsconfig)
		subscriberOptions.SetTLSConfig(tlsconfig)
		publisherOptions.SetTLSConfig(tlsconfig)
	}
	// } else {
	// 	tlsconfig := nil
	// }

	// opts.SetReconnectingHandler(messagePubHandler, opts)
	// opts.OnConnect = connectHandler

	subscriberOptions.OnConnect = func(c mqtt.Client) {
		log.Info("ON connect ")
		if token := c.Subscribe(opt.SubClientId+"/"+opt.NodeId+"/+", 0, messagePubHandler); token.Wait() && token.Error() != nil {
			log.Fatalf("subscribe connection: %v", token.Error())
		}
	}
	// log.Debug(fmt.Sprintf("Subscriber options: %+v\n", subscriberOptions))

	publisher := mqtt.NewClient(publisherOptions)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create publisher connection: %v", token.Error())
	}
	log.Debug(fmt.Sprintf("MQTT publisher client: %+v\n", publisher))

	subscriber := mqtt.NewClient(subscriberOptions)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create subscriber connection: %v", token.Error())
	}
	log.Debug(fmt.Sprintf("MQTT subscriber client: %+v\n", subscriber))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			if !publisher.IsConnected() {
				log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

				if token := publisher.Connect(); token.Wait() && token.Error() != nil {
					log.Fatalf("failed to create publisher connection: %v", token.Error())
				}
			}

			if !subscriber.IsConnected() {
				log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

				if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
					log.Fatalf("failed to create subscriber connection: %v", token.Error())
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

	var filter = map[string]string{"ID": "10"}

	statuses := jsonlines.Read(constants.StatusFile, "", "", filter, false)
	manifests := jsonlines.Read(constants.ManifestFile, "", "", nil, false)

	var mani []ManifestStatus
	var deviceParams = DeviceParams{"10", "10", "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		log.Info(rec)
		mani = append(mani, ManifestStatus{rec["id"].(string), rec["version"].(string), rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if "SUCCESS" == rec["status"].(string) {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	msg := StatusMessage{opt.NodeId, millis, "Available", actv_cnt, serv_cnt, mani, deviceParams}

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending update.", opt.TopicName, statuses, msg, string(b_msg))
	if token := cl.Publish(opt.PubClientId+"/"+opt.NodeId+"/"+opt.TopicName, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}
}

func post(jsonReq []byte, nextHost string) {
	fmt.Printf("Next host %s", nextHost)
	resp, err := http.Post(nextHost, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
	if err != nil {
		log.Info("Post API Connection error: %v", err)
	} else {

		defer resp.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		// Convert response body to string
		bodyString := string(bodyBytes)
		log.Info(bodyString)
	}
}
