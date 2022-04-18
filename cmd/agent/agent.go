package main

import (
	"fmt"
	"io"
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

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/google/uuid"
	"github.com/weeveiot/weeve-agent/internal"
	"github.com/weeveiot/weeve-agent/internal/handler"
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

	// mqtt config
	internal.Broker = opt.Broker
	internal.NoTLS = opt.NoTLS
	internal.PubClientId = opt.PubClientId
	internal.SubClientId = opt.SubClientId
	internal.TopicName = opt.TopicName

	if len(opt.ConfigPath) > 0 {
		handler.ConfigPath = opt.ConfigPath
	} else {
		// use the default path and filename
		handler.ConfigPath = path.Join(util.GetExeDir(), handler.NodeConfigFile)
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

	nodeConfig = handler.ReadNodeConfig()
	validateUpdateConfig(nodeConfig)

	// Read node configurations
	nodeConfig = handler.ReadNodeConfig()

	isRegistered := len(nodeConfig[handler.KeyNodeId]) > 0

	if opt.NodeId == "register" && !isRegistered {
		nodeId = uuid.New().String()
	} else {
		nodeId = nodeConfig[handler.KeyNodeId]
	}
	internal.NodeId = nodeId

	if !isRegistered {
		log.Info("Registering node and downloading certificate and key ...")
		registered = false
		publisher = internal.InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId+"/Registration", false)
		subscriber = internal.InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId+"/Certificate", true)
		for {
			published := internal.PublishMessages(publisher, nodeId, nodeConfig[handler.KeyNodeName], "Registration")
			if published {
				break
			}
			time.Sleep(time.Second * 5)
		}
		time.Sleep(time.Second * 25)
		nodeConfig = handler.ReadNodeConfig()
		registered = len(nodeConfig[handler.KeyNodeId]) > 0
		internal.Registered = registered
	} else {
		log.Info("Node already registered!")
		registered = true
		internal.Registered = registered
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	go func() {
		for registered {
			log.Debug("Node registered >> ", registered, " | connected >> ", connected)
			if !connected {
				internal.DisconnectBroker(publisher, subscriber)
				nodeConfig = handler.ReadNodeConfig()
				publisher = internal.InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId, false)
				subscriber = internal.InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId, true)
				connected = true
			}
			internal.ReconnectIfNecessary(publisher, subscriber)
			internal.PublishMessages(publisher, nodeId, "", "All")
			time.Sleep(time.Second * time.Duration(opt.Heartbeat))
		}
	}()

	// Cleanup on ending the process
	<-done
	internal.DisconnectBroker(publisher, subscriber)
}

func validateUpdateConfig(nodeConfigs map[string]string) {
	var configChanged bool
	nodeConfig := map[string]string{}
	if opt.NodeId != "register" {
		nodeConfig[handler.KeyNodeId] = opt.NodeId
		configChanged = true
	}

	if len(opt.RootCertPath) > 0 {
		nodeConfig[handler.KeyAWSRootCert] = opt.RootCertPath
		configChanged = true
	}

	if len(opt.CertPath) > 0 {
		nodeConfig[handler.KeyCertificate] = opt.CertPath
		configChanged = true
	}

	if len(opt.KeyPath) > 0 {
		nodeConfig[handler.KeyPrivateKey] = opt.KeyPath
		configChanged = true
	}

	if len(opt.NodeName) > 0 {
		nodeConfig[handler.KeyNodeName] = opt.NodeName
		configChanged = true
	} else {
		nodeNm := nodeConfigs[handler.KeyNodeName]
		if nodeNm == "" {
			nodeNm = "New Node"
		}
		if nodeNm == "New Node" {
			nodeNm = fmt.Sprintf("%s%d", nodeNm, mathrand.Intn(10000))
			nodeConfig[handler.KeyNodeName] = nodeNm
			configChanged = true
		}
	}

	if configChanged {
		handler.UpdateNodeConfig(nodeConfig)
	}
}
