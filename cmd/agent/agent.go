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

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/docker"
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

// logging into the terminal and files
func init() {
	plainFormatter := new(PlainFormatter)
	plainFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(plainFormatter)
}

func main() {
	parseCLIoptions()

	model.InitKnownManifests()

	docker.SetupDockerClient()

	err := com.RegisterNode()
	if err != nil {
		log.Fatal(err)
	}
	com.DisconnectNode()
	err = com.ConnectNode()
	if err != nil {
		log.Fatal(err)
	}

	// Kill the agent on a keyboard interrupt
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	go func() {
		for {
			err = com.SendHeartbeat()
			if err != nil {
				log.Error(err)
			}
		}
	}()

	// Cleanup on ending the process
	<-done
	com.DisconnectNode()
}

func parseCLIoptions() {
	var opt model.Params
	parser := flags.NewParser(&opt, flags.Default)

	if _, err := parser.Parse(); err != nil {
		log.Fatal("Error on command line parser ", err)
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
	if opt.Verbose {
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

	// FLAG: ConfigPath
	if len(opt.ConfigPath) > 0 {
		config.ConfigPath = opt.ConfigPath
	} else {
		// use the default path and filename
		const configFileName = "nodeconfig.json"
		config.ConfigPath = path.Join(util.GetExeDir(), configFileName)
	}
	config.UpdateNodeConfig(opt)

	// FLAG: Broker
	brokerUrl, err := url.Parse(opt.Broker)
	if err != nil {
		log.Fatal("Error on parsing broker ", err)
	}
	validateBrokerUrl(brokerUrl)

	// FLAG: NoTLS
	if opt.NoTLS {
		log.Info("TLS disabled!")
	} else {
		if brokerUrl.Scheme != "tls" {
			log.Fatalf("Incorrect protocol, TLS is required unless --notls is set. You specified protocol in broker to: %v", brokerUrl.Scheme)
		}
	}

	// FLAG: Broker, NoTLS, Heartbeat, PubClientId, SubClientId, TopicName
	com.SetParams(opt)
}

func validateBrokerUrl(u *url.URL) {
	host, port, _ := net.SplitHostPort(u.Host)

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatal("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Infof("Broker host->%v at port->%v over %v", host, port, u.Scheme)
}
