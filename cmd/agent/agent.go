package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	golog "log"
	mathrand "math/rand"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Jeffail/gabs/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/google/uuid"
	"github.com/weeveiot/weeve-agent/internal"

	"github.com/weeveiot/weeve-agent/internal/util"
)

type Params struct {
	Verbose      []bool `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker       string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	PubClientId  string `long:"pubClientId" short:"c" description:"Publisher ClientId" required:"true"`
	SubClientId  string `long:"subClientId" short:"s" description:"Subscriber ClientId" required:"true"`
	TopicName    string `long:"publish" short:"t" description:"Topic Name" required:"true"`
	Heartbeat    int    `long:"heartbeat" short:"h" description:"Heartbeat time in seconds" required:"false" default:"30"`
	MqttLogs     bool   `long:"mqttlogs" short:"m" description:"For developer - Display detailed MQTT logging messages" required:"false"`
	NoTLS        bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
	LogLevel     string `long:"loglevel" short:"l" default:"info" description:"Set the logging level" required:"false"`
	LogFileName  string `long:"logfilename" default:"Weeve_Agent.log" description:"Set the name of the log file" required:"false"`
	LogSize      int    `long:"logsize" default:"1" description:"Set the size of each log files (MB)" required:"false"`
	LogAge       int    `long:"logage" default:"1" description:"Set the time period to retain the log files (days)" required:"false"`
	LogBackup    int    `long:"logbackup" default:"5" description:"Set the max number of log files to retain" required:"false"`
	LogCompress  bool   `long:"logcompress" description:"To compress the log files" required:"false"`
	NodeId       string `long:"nodeId" short:"i" description:"ID of this node" required:"false" default:"register"`
	NodeName     string `long:"name" short:"n" description:"Name of this node to be registered" required:"false"`
	RootCertPath string `long:"rootcert" short:"r" description:"Path to MQTT broker (server) certificate" required:"false"`
	CertPath     string `long:"cert" short:"f" description:"Path to certificate to authenticate to Broker" required:"false"`
	KeyPath      string `long:"key" short:"k" description:"Path to private key to authenticate to Broker" required:"false"`
	ConfigPath   string `long:"config" description:"Path to the .json config file" required:"false"`
	ManifestPath string `long:"manifest" description:"Path to the  .json manifest file" required:"false"`
	TopicRcvd    string `long:"topic" description:"topic for manifest deployment" required:"false"`
}

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

var opt Params
var nodeId string
var parser = flags.NewParser(&opt, flags.Default)
var registered = false
var connected = false
var DeploymentStatus = false

// logging into the terminal and files
func init() {

	plainFormatter := new(PlainFormatter)
	plainFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(plainFormatter)

}

func main() {

	// Parse the CLI options
	if _, err := parser.Parse(); err != nil {
		log.Error("Error on command line parser ", err)
		os.Exit(1)
	}

	if len(opt.ConfigPath) > 0 {
		internal.ConfigPath = opt.ConfigPath
	} else {
		// use the default path and filename
		internal.ConfigPath = path.Join(util.GetExeDir(), internal.NodeConfigFile)
	}
	var manifestFile []byte
	// file path for the manifest.json
	if len(opt.ManifestPath) > 0 {
		internal.ManifestPath = opt.ManifestPath
		//read Manifest file
		manifestFile = internal.ReadManifest()
		DeploymentStatus = internal.DeploManifestLocal(opt.TopicRcvd, manifestFile, false)
	}
	// FLAG: LogLevel
	l, _ := log.ParseLevel(opt.LogLevel)
	log.SetLevel(l)

	// LOG CONFIGS
	logger := &lumberjack.Logger{
		Filename:   filepath.ToSlash(opt.LogFileName),
		MaxSize:    opt.LogSize,
		MaxAge:     opt.LogAge,
		MaxBackups: opt.LogBackup,
		Compress:   opt.LogCompress,
	}

	// FLAG: Verbose
	if len(opt.Verbose) >= 1 {
		multiWriter := io.MultiWriter(os.Stderr, logger)
		log.SetOutput(multiWriter)
	} else {
		log.SetOutput(logger)
	}

	// FLAG: Show the logs from the Paho package at STDOUT
	if opt.MqttLogs {
		mqtt.ERROR = golog.New(logger, "[ERROR] ", 0)
		mqtt.CRITICAL = golog.New(logger, "[CRIT] ", 0)
		mqtt.WARN = golog.New(logger, "[WARN]  ", 0)
		mqtt.DEBUG = golog.New(logger, "[DEBUG] ", 0)
	}

	log.Info("Started logging!")

	log.Info("Logging level set to ", log.GetLevel(), "!")

	// OPTION: Parse and validate the Broker url
	u, err := url.Parse(opt.Broker)
	if err != nil {
		log.Error("Error on parsing broker ", err)
		os.Exit(1)
	}

	host, port, _ := net.SplitHostPort(u.Host)

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatal("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Info(fmt.Sprintf("Broker host->%v at port->%v over %v", host, port, u.Scheme))

	log.Debug("Broker >> ", opt.Broker)

	// FLAG: Optionally disable TLS
	if opt.NoTLS {
		log.Info("TLS disabled!")
	} else {
		if u.Scheme != "tls" {
			log.Fatalf("Incorrect protocol, TLS is required unless --notls is set. You specified protocol in broker to: %v", u.Scheme)
		}
	}
	var publisher mqtt.Client
	var subscriber mqtt.Client
	var nodeConfig map[string]string

	nodeConfig = internal.ReadNodeConfig()
	validateUpdateConfig(nodeConfig)

	// Read node configurations
	nodeConfig = internal.ReadNodeConfig()

	isRegistered := len(nodeConfig[internal.KeyNodeId]) > 0

	if opt.NodeId == "register" && !isRegistered {
		nodeId = uuid.New().String()
	} else {
		nodeId = nodeConfig[internal.KeyNodeId]
	}
	if opt.ManifestPath == "" || (len(opt.ManifestPath) > 0 && DeploymentStatus) {
		if !isRegistered {
			log.Info("Registering node and downloading certificate and key ...")
			registered = false
			publisher = InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId+"/Registration", false)
			subscriber = InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId+"/Certificate", true)
			for {
				published := PublishMessages(publisher, nodeId, nodeConfig[internal.KeyNodeName], "Registration")
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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	go func() {
		for {
			log.Debug("Node registered >> ", registered, " | connected >> ", connected)
			if registered {
				if !connected {
					DisconnectBroker(publisher, subscriber)
					nodeConfig = internal.ReadNodeConfig()
					publisher = InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId, false)
					subscriber = InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId, true)
					connected = true
				}
				ReconnectIfNecessary(publisher, subscriber)
				PublishMessages(publisher, nodeId, "", "All")
			}

			time.Sleep(time.Second * time.Duration(opt.Heartbeat))
		}
	}()

	// Cleanup on ending the process
	<-done
	DisconnectBroker(publisher, subscriber)
}

func InitBrokerChannel(nodeConfig map[string]string, pubsubClientId string, isSubscribe bool) mqtt.Client {

	// var pubsubClient mqtt.Client

	log.Debug("Client id >> ", pubsubClientId, " | subscription >> ", isSubscribe)

	// Build the options for the mqtt client
	channelOptions := mqtt.NewClientOptions()
	channelOptions.AddBroker(opt.Broker)
	channelOptions.SetClientID(pubsubClientId)
	channelOptions.SetDefaultPublishHandler(messagePubHandler)
	channelOptions.OnConnectionLost = connectLostHandler
	if isSubscribe {
		channelOptions.OnConnect = connectHandler
	}

	// Optionally add the TLS configuration to the 2 client options
	if !opt.NoTLS {
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

func NewTLSConfig(nodeConfig map[string]string) (config *tls.Config, err error) {
	log.Debug("MQTT root cert path >> ", nodeConfig[internal.KeyAWSRootCert])

	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(nodeConfig[internal.KeyAWSRootCert])
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	log.Debug("MQTT cert path >> ", nodeConfig[internal.KeyCertificate])
	log.Debug("MQTT key path >> ", nodeConfig[internal.KeyPrivateKey])

	cert, err := tls.LoadX509KeyPair(nodeConfig[internal.KeyCertificate], nodeConfig[internal.KeyPrivateKey])
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
		topicNm = opt.PubClientId + "/" + pubNodeId + "/" + "Registration"

		msg := internal.GetRegistrationMessage(pubNodeId, nodeName)
		log.Infoln("Sending registration request.", "Registration", msg)
		b_msg, err = json.Marshal(msg)
		if err != nil {
			log.Fatalf("Marshall error: %v", err)
		}

	} else {
		topicNm = opt.PubClientId + "/" + pubNodeId + "/" + opt.TopicName
		msg := internal.GetStatusMessage(pubNodeId)
		log.Info("Sending update >> ", "Topic: ", opt.TopicName, " >> Body: ", msg)
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

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	jsonParsed, err := gabs.ParseJSON(msg.Payload())
	if err != nil {
		log.Infoln("Received message on topic: ", msg.Topic(), *jsonParsed)
	}

	topic_rcvd := ""

	if msg.Topic() == opt.SubClientId+"/"+nodeId+"/Certificate" {
		certificates := internal.DownloadCertificates(msg.Payload())
		if certificates != nil {
			time.Sleep(time.Second * 10)
			internal.MarkNodeRegistered(nodeId, certificates)
			registered = true
			log.Info("Node registration done | Certificates downloaded!")
		}
	} else {
		if strings.HasPrefix(msg.Topic(), opt.SubClientId+"/"+nodeId+"/") {
			topic_rcvd = strings.Replace(msg.Topic(), opt.SubClientId+"/"+nodeId+"/", "", 1)
		}
		log.Info(topic_rcvd)
		log.Info(msg.Payload())
		internal.ProcessMessage(topic_rcvd, msg.Payload(), false)
	}
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected >> registered : ", registered)
	var topicName string
	topicName = opt.SubClientId + "/" + nodeId + "/Certificate"
	if registered {
		topicName = opt.SubClientId + "/" + nodeId + "/+"
	}

	log.Debug("ON connect >> subscribes >> topicName : ", topicName)
	if token := c.Subscribe(topicName, 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost", err)
}

func validateUpdateConfig(nodeConfigs map[string]string) {
	var configChanged bool
	nodeConfig := map[string]string{}
	if opt.NodeId != "register" {
		nodeConfig[internal.KeyNodeId] = opt.NodeId
		configChanged = true
	}

	if len(opt.RootCertPath) > 0 {
		nodeConfig[internal.KeyAWSRootCert] = opt.RootCertPath
		configChanged = true
	}

	if len(opt.CertPath) > 0 {
		nodeConfig[internal.KeyCertificate] = opt.CertPath
		configChanged = true
	}

	if len(opt.KeyPath) > 0 {
		nodeConfig[internal.KeyPrivateKey] = opt.KeyPath
		configChanged = true
	}

	if len(opt.NodeName) > 0 {
		nodeConfig[internal.KeyNodeName] = opt.NodeName
		configChanged = true
	} else {
		nodeNm := nodeConfigs[internal.KeyNodeName]
		if nodeNm == "" {
			nodeNm = "New Node"
		}
		if nodeNm == "New Node" {
			nodeNm = fmt.Sprintf("%s%d", nodeNm, mathrand.Intn(10000))
			nodeConfig[internal.KeyNodeName] = nodeNm
			configChanged = true
		}
	}

	if configChanged {
		internal.UpdateNodeConfig(nodeConfig)
	}
}
