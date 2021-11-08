package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	golog "log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/google/uuid"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

type Params struct {
	Verbose     []bool `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker      string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	PubClientId string `long:"pubClientId" short:"c" description:"Publisher ClientId" required:"true"`
	SubClientId string `long:"subClientId" short:"s" description:"Subscriber ClientId" required:"true"`
	TopicName   string `long:"publish" short:"t" description:"Topic Name" required:"true"`
	Heartbeat   int    `long:"heartbeat" short:"h" description:"Heartbeat time in seconds" required:"false" default:"30"`
	MqttLogs    bool   `long:"mqttlogs" short:"m" description:"For developer - Display detailed MQTT logging messages" required:"false"`
	NoTLS       bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
}

var opt Params
var nodeId string
var publisher mqtt.Client
var subscriber mqtt.Client
var parser = flags.NewParser(&opt, flags.Default)
var sendHeartbeats = false
var nodeConfig map[string]string

// logging into terminal and files
func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.ToSlash("file_path along with file_name"), //eg. xxx/xxx/xxx/file_name.txt
		MaxSize:    1,                                                  //Size limit of a single .txt file in MB. Default -> 100MB
		MaxAge:     30,                                                 //Number of days to retain the files. Default -> no file deletion based on age
		MaxBackups: 10,                                                 //Maximum number of old files to retain. Default -> retain all old files
		LocalTime:  false,                                              //time in UTC
		Compress:   true,                                               //option to compress the files
	}

	multiWriter := io.MultiWriter(os.Stderr, lumberjackLogger)

	logFormatter := new(log.TextFormatter)
	logFormatter.TimestampFormat = time.RFC1123Z
	logFormatter.FullTimestamp = true

	log.SetFormatter(logFormatter)
	log.SetLevel(log.InfoLevel)
	log.SetOutput(multiWriter)

	log.SetLevel(log.DebugLevel)
	log.Info("Started logging")
}

func main() {

	// Parse the CLI options
	if _, err := parser.Parse(); err != nil {
		log.Error("Error on command line parser ", err)
		os.Exit(1)
	}

	// FLAG: Show the logs from the Paho package at STDOUT
	if opt.MqttLogs {
		mqtt.ERROR = golog.New(os.Stdout, "[ERROR] ", 0)
		mqtt.CRITICAL = golog.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = golog.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = golog.New(os.Stdout, "[DEBUG] ", 0)
	}

	// FLAG: Verbose
	if len(opt.Verbose) >= 1 {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Logging level set to ", log.GetLevel())

	// OPTION: Parse and validated the Broker url
	u, err := url.Parse(opt.Broker)
	if err != nil {
		log.Error("Error on parsing broker ", err)
		os.Exit(1)
	}

	host, port, _ := net.SplitHostPort(u.Host)

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatalf("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Info(fmt.Sprintf("Broker host %v at port %v over %v\n", host, port, u.Scheme))

	log.Debug("Broker: ", opt.Broker)

	// FLAG: Optionally disable TLS
	if opt.NoTLS {
		log.Info("TLS disabled!")
	} else {
		if u.Scheme != "tls" {
			log.Fatalf("Incorrect protocol, TLS is required unless --notls is set. You specified protocol in broker to: %v", u.Scheme)
		}
	}

	nodeRegistered := internal.CheckIfNodeAlreadyRegistered()
	if !nodeRegistered {
		log.Info("Registering node!")
		sendHeartbeats = false
		InitializeBroker(certPubHandler, certConnectHandler, false)
		PublishRegistrationMessage(publisher)
	} else {
		sendHeartbeats = true
		log.Info("Node already registered!")
		InitializeBroker(messagePubHandler, connectHandler, true)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			if sendHeartbeats {
				CheckBrokerConnection()
				PublishMessages(publisher)
			}
			time.Sleep(time.Second * time.Duration(opt.Heartbeat))
		}
	}()
	<-done

	// Cleanup on ending the process
	<-c
	DisconnectBrocker()
}

func InitializeBroker(lMessageHandler mqtt.MessageHandler, lConnectHandler mqtt.OnConnectHandler, startHeartbeats bool) bool {

	log.Debug("Heartbeat time: ", opt.Heartbeat)
	sendHeartbeats = startHeartbeats

	// Read node configurations
	nodeConfig = internal.ReadNodeConfig()
	nodeRegistered := len(nodeConfig[internal.NodeIdKey]) > 0

	var statusPublishTopic string
	var nodeSubscribeTopic string

	if nodeRegistered {
		nodeId = nodeConfig[internal.NodeIdKey]
		statusPublishTopic = opt.PubClientId + "/" + nodeId
		nodeSubscribeTopic = opt.SubClientId + "/" + nodeId
	} else {
		nodeId = uuid.New().String()
		statusPublishTopic = opt.PubClientId + "/" + nodeId + "/Registration"
		nodeSubscribeTopic = opt.SubClientId + "/" + nodeId + "/Certificate"
	}

	// OPTIONS: ID and topics
	log.Debug("NodeId: ", nodeId)
	log.Debug("Status heartbeat publishing to topic: ", statusPublishTopic)
	log.Debug("This node is subscribed to topic: ", nodeSubscribeTopic)

	// Build the options for the publish client
	publisherOptions := mqtt.NewClientOptions()
	publisherOptions.AddBroker(opt.Broker)
	publisherOptions.SetClientID(statusPublishTopic)
	publisherOptions.SetDefaultPublishHandler(lMessageHandler)
	publisherOptions.OnConnectionLost = connectLostHandler

	// Build the options for the subscribe client
	subscriberOptions := mqtt.NewClientOptions()
	subscriberOptions.AddBroker(opt.Broker)
	subscriberOptions.SetClientID(nodeSubscribeTopic)
	subscriberOptions.SetDefaultPublishHandler(lMessageHandler)
	subscriberOptions.OnConnectionLost = connectLostHandler
	subscriberOptions.OnConnect = lConnectHandler

	// Optionally add the TLS configuration to the 2 client options
	if !opt.NoTLS {
		tlsconfig, err := NewTLSConfig()
		if err != nil {
			log.Fatalf("failed to create TLS configuration: %v", err)
		}
		// log.Debug("Tls Config >> ", tlsconfig)
		subscriberOptions.SetTLSConfig(tlsconfig)
		log.Debug("TLS set on subscriber options")
		publisherOptions.SetTLSConfig(tlsconfig)
		log.Debug("TLS set on publisher options")
	}

	log.Debug("Publisher options:\n", publisherOptions)
	log.Debug("Subscriber options:\n", subscriberOptions)

	log.Debug("Finished parsing and MQTT configuration")

	publisher = mqtt.NewClient(publisherOptions)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create publisher connection: %v", token.Error())
	} else {
		log.Debug("MQTT publisher connected")
	}

	subscriber = mqtt.NewClient(subscriberOptions)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create subscriber connection: %v", token.Error())
	} else {
		log.Debug("MQTT subscriber connected")
	}

	return nodeRegistered
}

func NewTLSConfig() (config *tls.Config, err error) {
	// Root folder of this project
	_, b, _, _ := runtime.Caller(0)
	Root := filepath.Join(filepath.Dir(b), "../..")

	rootCert := path.Join(Root, nodeConfig[internal.AWSRootCertKey])
	nodeCert := path.Join(Root, nodeConfig[internal.CertificateKey])
	pvtKey := path.Join(Root, nodeConfig[internal.PrivateKeyKay])

	log.Debug("Node Certificate: ", nodeCert)
	log.Debug("Node PrivateKey: ", pvtKey)

	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(rootCert)
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(nodeCert, pvtKey)
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

func CheckBrokerConnection() {
	// Attempt reconnect
	if !publisher.IsConnected() {
		log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := publisher.Connect(); token.Wait() && token.Error() != nil {
			log.Error("failed to create publisher connection: ", token.Error())
		}
	}

	if !subscriber.IsConnected() {
		log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())

		if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
			log.Errorf("failed to create subscriber connection: %v", token.Error())
		}
	}
}

func PublishMessages(cl mqtt.Client) {

	msg := GetStatusMessage(nodeId)

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending update.", opt.TopicName, msg, string(b_msg))
	if token := cl.Publish(opt.PubClientId+"/"+nodeId+"/"+opt.TopicName, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}
}

func GetStatusMessage(nodeId string) model.StatusMessage {
	manifests := jsonlines.Read(constants.ManifestFile, "", "", nil, false)

	var mani []model.ManifestStatus
	var deviceParams = model.DeviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		mani = append(mani, model.ManifestStatus{ManifestId: rec["id"].(string), ManifestVersion: rec["version"].(string), Status: rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if rec["status"].(string) == "SUCCESS" {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	return model.StatusMessage{Id: nodeId, Timestamp: millis, Status: "Available", ActiveServiceCount: actv_cnt, ServiceCount: serv_cnt, ServicesStatus: mani, DeviceParams: deviceParams}
}

func PublishRegistrationMessage(cl mqtt.Client) {

	msg := GetRegistrationMessage(nodeId)

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending registration request.", "Registration", msg, string(b_msg))
	if token := cl.Publish(opt.PubClientId+"/"+nodeId+"/"+"Registration", 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send registration request: %v", token.Error())
	}
}

func GetRegistrationMessage(nodeId string) model.RegistrationMessage {
	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000

	return model.RegistrationMessage{Id: nodeId, Timestamp: millis, Status: "Registering", Operation: "Registration", Name: "NodeName"}
}

func DisconnectBrocker() {
	if publisher != nil && publisher.IsConnected() {
		log.Info("Disconnecting.....")
		publisher.Disconnect(250)
	}

	if subscriber != nil && subscriber.IsConnected() {
		log.Info("Disconnecting.....")
		subscriber.Disconnect(250)
	}
}

// The message fallback handler used for incoming messages

var certPubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info("Received message on topic: ", msg.Topic())

	topic_rcvd := ""

	if strings.HasPrefix(msg.Topic(), opt.SubClientId+"/"+nodeId+"/") {
		topic_rcvd = strings.Replace(msg.Topic(), opt.SubClientId+"/"+nodeId+"/", "", 1)
	}

	if msg.Topic() == opt.SubClientId+"/"+nodeId+"/Certificate" {
		certificates := internal.DownloadCertificates(msg.Payload())
		if certificates != nil {
			marked := internal.MarkNodeRegistered(nodeId, certificates)
			if marked {
				nodeRegistered := InitializeBroker(messagePubHandler, connectHandler, true)
				sendHeartbeats = nodeRegistered
			}
		}
	} else {
		internal.ProcessMessage(topic_rcvd, msg.Payload(), false)
	}
}

var certConnectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected")
	if token := c.Subscribe(opt.SubClientId+"/"+nodeId+"/Certificate", 0, certPubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info("Received message on topic: ", msg.Topic())

	topic_rcvd := ""

	if strings.HasPrefix(msg.Topic(), opt.SubClientId+"/"+nodeId+"/") {
		topic_rcvd = strings.Replace(msg.Topic(), opt.SubClientId+"/"+nodeId+"/", "", 1)
	}

	internal.ProcessMessage(topic_rcvd, msg.Payload(), false)
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected")
	if token := c.Subscribe(opt.SubClientId+"/"+nodeId+"/+", 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost", err)
}
