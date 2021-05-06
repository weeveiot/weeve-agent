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
	ClientId    string `long:"clientId" short:"c" description:"ClientId" required:"true"`
	TopicName   string `long:"topic" short:"t" description:"Topic Name" required:"true"`
	Cert        string `long:"cert" short:"f" description:"Certificate to connect Broker" required:"false"`
	HostUrl     string `long:"publicurl" short:"u" description:"Public URL to connect from public" required:"false"`
	NodeApiPort string `long:"nodeport" short:"p" description:"Port where edge node api is listening" required:"true"`
}

type StatusMessage struct {
	Id                 string           `json:"ID"`
	Timestamp          int64            `json:"timestamp"`
	Connectivity       string           `json:"connectivity"`
	ActiveServiceCount int64            `json:"activeServiceCount"`
	ServiceCount       int64            `json:"serviceCount"`
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
var Broker string
var NodeId string
var CertPrefix string
var ClientId string
var TopicName string

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
	log.Info("Started logging")
}

func NewTLSConfig() (config *tls.Config, err error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile("AmazonRootCA1.pem")
	if err != nil {
		return
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(CertPrefix+"-certificate.pem.crt", CertPrefix+"-private.pem.key")
	if err != nil {
		return
	}

	config = &tls.Config{
		RootCAs:      certpool,
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		Certificates: []tls.Certificate{cert},
	}
	return
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info(" messagePubHandler Received message on topic: ", msg.Topic(), "\nMessage: %s\n", msg.Payload())

	topic_rcvd := ""

	if strings.HasPrefix(msg.Topic(), ClientId+"/"+NodeId+"/") {
		topic_rcvd = strings.Replace(msg.Topic(), ClientId+"/"+NodeId+"/", "", 1)
	}

	if topic_rcvd == "CheckVersion" {

	} else if topic_rcvd == "Deploy" {
		post([]byte(msg.Payload()), "http://localhost:"+opt.NodeApiPort+"/pipelines")
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info(fmt.Println("Connected"))
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info(fmt.Printf("Connect lost: %v", err))
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

	Broker = opt.Broker
	NodeId = opt.NodeId
	CertPrefix = opt.Cert
	ClientId = opt.ClientId
	TopicName = opt.TopicName

	tlsconfig, err := NewTLSConfig()
	if err != nil {
		log.Fatalf("failed to create TLS configuration: %v", err)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(Broker)
	opts.SetClientID(ClientId).SetTLSConfig(tlsconfig)
	opts.SetDefaultPublishHandler(messagePubHandler)
	// opts.SetReconnectingHandler(messagePubHandler, opts)
	// opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	opts.OnConnect = func(c mqtt.Client) {
		log.Info("ON connect ")
		if token := c.Subscribe(ClientId+"/"+NodeId+"/+", 0, messagePubHandler); token.Wait() && token.Error() != nil {
			log.Fatalf("subscribe connection: %v", token.Error())
		}
	}

	log.Info(opts)

	cl := mqtt.NewClient(opts)
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create connection: %v", token.Error())
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			if !cl.IsConnected() {
				log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

				if token := cl.Connect(); token.Wait() && token.Error() != nil {
					log.Fatalf("failed to create connection: %v", token.Error())
				}

			}
			// 			// fmt.Println("Listening for new events.")
			// 			// if token := cl.Subscribe(ClientId+"/"+NodeId+"/#", 1, onMessageReceived); token.Wait() && token.Error() != nil {
			// 			// 	log.Fatalf("failed to create subscription: %v", token.Error())
			// 			// }
			// 		}

			PublishMessages(cl)

			log.Info("Sleeping..... 30 * ", time.Second)
			time.Sleep(time.Second * 30)
			log.Info("waken..... 30 * ", time.Second)
		}
	}()
	<-done
	// time.Sleep(time.Second * 60)
	// if cl.IsConnected() {
	// 	log.Info("Disconnecting.....")
	// 	cl.Disconnect(250)
	// }
	<-c
}

func PublishMessages(cl mqtt.Client) {

	var filter = map[string]string{"ID": "10"}

	statuses := jsonlines.Read(constants.StatusFile, "", "", filter, false)

	var mani []ManifestStatus
	var deviceParams = DeviceParams{"10", "10", "20"}

	mani = append(mani, ManifestStatus{"1211446464", "1.0", "Running"})

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	msg := StatusMessage{NodeId, millis, "Available", 0, 0, mani, deviceParams}

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending update.", opt.TopicName, statuses, msg, string(b_msg))
	if token := cl.Publish(opt.ClientId+"/"+NodeId+"/"+opt.TopicName, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}
}

func post(jsonReq []byte, nextHost string) {
	fmt.Printf("Next host %s", nextHost)
	resp, err := http.Post(nextHost, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
	if err != nil {
		log.Fatalf("Post API Connection error: %v", err)
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert response body to string
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)
}
