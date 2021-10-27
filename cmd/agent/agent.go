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
	"path/filepath"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal"
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
	MqttLogs     bool   `long:"mqttlogs" short:"m" description:"For developer - Display detailed MQTT logging messages" required:"false"`
	NoTLS        bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
}

var opt Params
var parser = flags.NewParser(&opt, flags.Default)

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

func PublishMessages(cl mqtt.Client) {

	msg := internal.GetStatusMessage(opt.NodeId)

	b_msg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	log.Info("Sending update.", opt.TopicName, msg, string(b_msg))
	if token := cl.Publish(opt.PubClientId+"/"+opt.NodeId+"/"+opt.TopicName, 0, false, b_msg); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}
}

// The message fallback handler used for incoming messages

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info("Received message on topic: ", msg.Topic(), "\nMessage: %s\n", msg.Payload())

	topic_rcvd := ""

	// TODO: Refactor (remove) this , we already hwave a proper subscription in the connectHandler!
	if strings.HasPrefix(msg.Topic(), opt.SubClientId+"/"+opt.NodeId+"/") {
		topic_rcvd = strings.Replace(msg.Topic(), opt.SubClientId+"/"+opt.NodeId+"/", "", 1)
	}

	internal.ProcessMessage(topic_rcvd, msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	log.Info("ON connect >> connected")
	if token := c.Subscribe(opt.SubClientId+"/"+opt.NodeId+"/+", 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Error("Error on subscribe connection: ", token.Error())
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Info("Connection lost", err)
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
		if _, err := os.Stat(opt.RootCertPath); os.IsNotExist(err) {
			log.Fatalf("Root certificate path does not exist: %v", opt.RootCertPath)
		} else {
			log.Debug("Root server certificate: ", opt.RootCertPath)
		}
		if _, err := os.Stat(opt.CertPath); os.IsNotExist(err) {
			log.Fatalf("Client certificate path does not exist: %v", opt.CertPath)
		} else {
			log.Debug("Client certificate: ", opt.CertPath)
		}
		if _, err := os.Stat(opt.KeyPath); os.IsNotExist(err) {
			log.Fatalf("Client private key path does not exist: %v", opt.KeyPath)
		} else {
			log.Debug("Client private key: ", opt.KeyPath)
		}
	}

	// OPTIONS: ID and topics
	log.Debug("NodeId: ", opt.NodeId)
	log.Debug("Heartbeat time: ", opt.Heartbeat)
	statusPublishTopic := opt.PubClientId + "/" + opt.NodeId
	log.Debug("Status heartbeat publishing to topic: ", statusPublishTopic)

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

	// Optionally add the TLS configuration to the 2 client options
	if !opt.NoTLS {
		tlsconfig, err := NewTLSConfig(opt.CertPath)
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	publisher := mqtt.NewClient(publisherOptions)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		log.Errorf("failed to create publisher connection: %v", token.Error())
	} else {
		log.Debug("MQTT publisher connected")
	}

	subscriber := mqtt.NewClient(subscriberOptions)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		log.Errorf("failed to create subscriber connection: %v", token.Error())
	} else {
		log.Debug("MQTT subscriber connected")
	}

	// MAIN LOOP
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
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

			PublishMessages(publisher)

			log.Info("Sleeping ", opt.Heartbeat)
			time.Sleep(time.Second * time.Duration(opt.Heartbeat))
		}
	}()
	<-done

	// Cleanup on ending the process
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
