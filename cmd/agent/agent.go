package main

import (
	"fmt"
	"io"
	golog "log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/node"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

var opt model.Params
var parser = flags.NewParser(&opt, flags.Default)

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

	// Passing the arguments to the packages
	node.Opt = opt
	handler.Opt = opt

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

	// FLAG: ConfigPath
	if len(opt.ConfigPath) > 0 {
		handler.ConfigPath = opt.ConfigPath
	} else {
		// use the default path and filename
		handler.ConfigPath = path.Join(ioutility.GetExeDir(), handler.NodeConfigFile)
	}

	log.Info("Started logging!")

	log.Info("Logging level set to ", log.GetLevel(), "!")

	u, err := url.Parse(opt.Broker)
	if err != nil {
		log.Error("Error on parsing broker ", err)
		os.Exit(1)
	}
	node.ValidateBroker(u)

	// FLAG: Optionally disable TLS
	if opt.NoTLS {
		log.Info("TLS disabled!")
	} else {
		if u.Scheme != "tls" {
			log.Fatalf("Incorrect protocol, TLS is required unless --notls is set. You specified protocol in broker to: %v", u.Scheme)
		}
	}

	docker.SetupDockerClient()

	node.RegisterNode()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// MAIN LOOP
	go func() {
		for {
			log.Debug("Node registered >> ", node.Registered, " | connected >> ", node.Connected)
			node.NodeHeartbeat()
		}
	}()

	// Cleanup on ending the process
	<-done
	node.DisconnectNode()
}
