package main

import (
	"fmt"
	"io"
	golog "log"
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
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util"
)

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

var opt model.Params
var nodeId string
var parser = flags.NewParser(&opt, flags.Default)
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

	// mqtt config
	internal.SubClientId = opt.SubClientId

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
	handler.ValidateUpdateConfig(nodeConfig, opt.NodeId, opt.RootCertPath, opt.CertPath, opt.KeyPath, opt.NodeName)

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
		internal.Registered = false
		publisher = internal.InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId+"/Registration", false, opt.Broker, opt.MqttLogs)
		subscriber = internal.InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId+"/Certificate", true, opt.Broker, opt.MqttLogs)
		for {
			published := internal.PublishMessages(publisher, nodeId, nodeConfig[handler.KeyNodeName], "Registration", opt.PubClientId, opt.TopicName)
			if published {
				break
			}
			time.Sleep(time.Second * 5)
		}
	} else {
		log.Info("Node already registered!")
		internal.Registered = true
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	go func() {
		for {
			log.Debug("Node registered >> ", internal.Registered, " | connected >> ", connected)
			if internal.Registered {
				if !connected {
					internal.DisconnectBroker(publisher, subscriber)
					nodeConfig = handler.ReadNodeConfig()
					publisher = internal.InitBrokerChannel(nodeConfig, opt.PubClientId+"/"+nodeId, false, opt.Broker, opt.MqttLogs)
					subscriber = internal.InitBrokerChannel(nodeConfig, opt.SubClientId+"/"+nodeId, true, opt.Broker, opt.MqttLogs)
					connected = true
				}
				internal.ReconnectIfNecessary(publisher, subscriber)
				internal.PublishMessages(publisher, nodeId, "", "All", opt.PubClientId, opt.TopicName)
				time.Sleep(time.Second * time.Duration(opt.Heartbeat))
			} else {
				time.Sleep(time.Second * 5)
			}
		}
	}()

	// Cleanup on ending the process
	<-done
	internal.DisconnectBroker(publisher, subscriber)
}
