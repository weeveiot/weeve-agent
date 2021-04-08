package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"io/ioutil"
	"time"
)

type Options struct {
	NodeId    string `long:"nodeId" short:"i" description:"ID of this node" required:"true"`
	Verbose   []bool `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker    string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	ClientId  string `long:"clientId" short:"c" description:"ClientId" required:"true"`
	TopicName string `long:"topic" short:"t" description:"Topic Name" required:"true"`
	Cert      string `long:"cert" short:"f" description:"Certificate to connect Broker" required:"false"`
	HostUrl   string `long:"publicurl" short:"u" description:"Public URL to connect from public" required:"false"`

	// TODO: We only need this for AWS ECR integration...
	// RoleArn string `long:"role" short:"r" description:"Role Arn" required:"false"`
}

type Message struct {
	Status          string
	ManifestId      string
	ManifestVersion string
	Time            int64
	HostUrl         string
}

var opt Options
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

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func main() {
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
	opts.SetDefaultPublishHandler(f)
	log.Info(opts)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Info("Connecting.....", time.Now().String(), time.Now().UnixNano())
	cl := mqtt.NewClient(opts)
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to create connection: %v", token.Error())
	}

	m := Message{"Available", "1", "1", 1294706395881547000, opt.HostUrl}
	b, err := json.Marshal(m)
	if err != nil {
		log.Fatalf("Marshall error: %v", err)
	}

	fmt.Println("Sending update.")
	if token := cl.Publish(ClientId+"/"+NodeId+"/"+TopicName, 0, false, b); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to send update: %v", token.Error())
	}

	// fmt.Println("Listening for new events.")
	// if token := cl.Subscribe(ClientId+"/"+NodeId+"/#", 0, nil); token.Wait() && token.Error() != nil {
	// 	log.Fatalf("failed to create subscription: %v", token.Error())
	// }
	log.Info("Disconnecting.....")
	cl.Disconnect(250)

	log.Info("Sleeping..... 30 * ", time.Second)
	time.Sleep(time.Second * 30)

	<-c
}
