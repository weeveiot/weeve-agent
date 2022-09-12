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

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

// The init function in the main package is called before anything else
// Setup the logging here
func init() {
	const dateTimeFormat = "2006-01-02 15:04:05"

	plainFormatter := new(PlainFormatter)
	plainFormatter.TimestampFormat = dateTimeFormat
	log.SetFormatter(plainFormatter)
}

// The main package is a special package which is used with the programs that are executable and this package contains main() function
// The entrypoint for this binary
func main() {
	localManifest, disconnect := parseCLIoptions()

	manifest.InitKnownManifests()

	docker.SetupDockerClient()

	if localManifest != "" {
		err := handler.ReadDeployManifestLocal(localManifest)
		if err != nil {
			log.Fatal("Deployment of the local manifest failed! CAUSE --> ", err)
		}
	}

	err := com.RegisterNode()
	if err != nil {
		log.Fatal(err)
	}
	com.DisconnectNode()
	err = com.ConnectNode()
	if err != nil {
		log.Fatal(err)
	}

	if disconnect {
		log.Info("Undeploying all the edge applications ...")
		err := dataservice.UndeployAll()
		if err != nil {
			log.Fatal(err)
		}
		err = com.SendHeartbeat()
		if err != nil {
			log.Error(err)
		}
		log.Info("weeve agent disconnected")
		os.Exit(0)
	}

	// Kill the agent on a keyboard interrupt
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// Start threads to send status messages
	go monitorDataServiceStatus()
	go sendHeartbeat()
	go sendEdgeAppLogs()

	// Cleanup on ending the process
	<-done
	com.DisconnectNode()
}

func parseCLIoptions() (string, bool) {
	// The config file is used to store the agent configuration
	// If the agent binary restarts, this file will be used to start the agent again
	const configFileName = "nodeconfig.json"

	var opt model.Params

	parser := flags.NewParser(&opt, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		e, ok := err.(*flags.Error)
		if ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
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

	// FLAG: Stdout
	if opt.Stdout {
		multiWriter := io.MultiWriter(os.Stdout, logger)
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

	log.Info("Started logging")
	log.Info("Logging level set to ", log.GetLevel())

	// FLAG: ConfigPath
	if len(opt.ConfigPath) > 0 {
		config.ConfigPath = opt.ConfigPath
	} else {
		// use the default path and filename
		config.ConfigPath = path.Join(ioutility.GetExeDir(), configFileName)
	}
	log.Debug("Loading config file from ", config.ConfigPath)

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

	// FLAG: Broker, NoTLS, Heartbeat, TopicName
	com.SetParams(opt)
	handler.SetDisconnected(opt.Disconnect)

	return opt.ManifestPath, opt.Disconnect
}

func validateBrokerUrl(u *url.URL) {
	host, port, _ := net.SplitHostPort(u.Host)

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatal("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Infof("Broker host->%v at port->%v over %v", host, port, u.Scheme)
}

func monitorDataServiceStatus() {
	edgeApps, err := handler.GetDataServiceStatus()
	if err != nil {
		log.Error(err)
	}

	for {
		latestEdgeApps, statusChange, err := handler.CompareDataServiceStatus(edgeApps)
		if err != nil {
			log.Error(err)
		}
		if statusChange {
			err = com.SendHeartbeat()
			if err != nil {
				log.Error(err)
			}
		}
		edgeApps = latestEdgeApps
		log.Debug("Latest edge app status: ", edgeApps)
		time.Sleep(time.Second * time.Duration(5))
	}
}

func sendHeartbeat() {
	for {
		err := com.SendHeartbeat()
		if err != nil {
			log.Error(err)
		}
		time.Sleep(time.Second * time.Duration(com.GetHeartbeat()))
	}
}

func sendEdgeAppLogs() {
	for {
		err := com.SendEdgeAppLogs()
		if err != nil {
			log.Error(err)
		}
		time.Sleep(time.Second * time.Duration(config.GetEdgeAppLogIntervalSec()))
	}
}
