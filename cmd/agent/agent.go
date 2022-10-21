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
	"github.com/weeveiot/weeve-agent/internal/secret"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

func init() {
	log.SetFormatter(&PlainFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	localManifest, disconnect := parseCLIoptions()

	err := manifest.InitKnownManifests()
	if err != nil {
		log.Fatal(err)
	}
	nodePubKey, err := secret.InitNodeKeypair()
	if err != nil {
		log.Fatal(err)
	}

	docker.SetupDockerClient()

	if localManifest != "" {
		err := dataservice.ReadDeployManifestLocal(localManifest)
		if err != nil {
			log.Fatal("Deployment of the local manifest failed! CAUSE --> ", err)
		}
	}

	err = com.RegisterNode()
	if err != nil {
		log.Fatal(err)
	}

	err = com.ConnectNode(setSubscriptionHandlers())
	if err != nil {
		log.Fatal(err)
	}

	if disconnect {
		log.Info("Undeploying all the edge applications ...")
		err := dataservice.UndeployAll()
		if err != nil {
			log.Fatal(err)
		}

		dataservice.SetNodeStatus(model.NodeDisconnected)
		err = dataservice.SendStatus()
		if err != nil {
			log.Fatal(err)
		}
		log.Info("weeve agent disconnected")
		os.Exit(0)
	}

	dataservice.SetNodeStatus(model.NodeConnected)

	err = com.SendNodePublicKey(nodePubKey)
	if err != nil {
		log.Fatal(err)
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
	logFile := &lumberjack.Logger{
		Filename:   filepath.ToSlash(opt.LogFileName),
		MaxSize:    opt.LogSize,
		MaxAge:     opt.LogAge,
		MaxBackups: opt.LogBackup,
		Compress:   opt.LogCompress,
	}

	var logOutput io.Writer

	// FLAG: Stdout
	if opt.Stdout {
		logOutput = io.MultiWriter(os.Stdout, logFile)
	} else {
		logOutput = logFile
	}
	log.SetOutput(logOutput)

	// FLAG: Include the logs from the Paho package
	if opt.MqttLogs {
		mqtt.ERROR = golog.New(logOutput, "[ERROR] ", 0)
		mqtt.CRITICAL = golog.New(logOutput, "[CRIT] ", 0)
		mqtt.WARN = golog.New(logOutput, "[WARN]  ", 0)
		mqtt.DEBUG = golog.New(logOutput, "[DEBUG] ", 0)
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

	return opt.ManifestPath, opt.Disconnect
}

func validateBrokerUrl(u *url.URL) {
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		log.Fatal("Error on spliting host port ", err)
	}

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatal("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Infof("Broker host->%v at port->%v over %v", host, port, u.Scheme)
}

func setSubscriptionHandlers() map[string]mqtt.MessageHandler {
	subscriptions := make(map[string]mqtt.MessageHandler)

	subscriptions[com.TopicOrchestration] = handler.OrchestrationHandler
	subscriptions[com.TopicOrgPrivateKey] = handler.OrgPrivKeyHandler
	subscriptions[com.TopicNodeDelete] = handler.NodeDeleteHandler

	return subscriptions
}

func monitorDataServiceStatus() {
	edgeApps, err := dataservice.GetDataServiceStatus()
	if err != nil {
		log.Error(err)
	}

	for {
		time.Sleep(time.Second * time.Duration(5))
		latestEdgeApps, statusChange, err := dataservice.CompareDataServiceStatus(edgeApps)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debug("Latest edge app status: ", latestEdgeApps)

		if statusChange {
			err := dataservice.SendStatus()
			if err != nil {
				log.Error(err)
				continue
			}
			edgeApps = latestEdgeApps
		}
	}
}

func sendHeartbeat() {
	for {
		err := dataservice.SendStatus()
		if err != nil {
			log.Error(err)
		}

		time.Sleep(time.Second * time.Duration(com.GetHeartbeat()))
	}
}

func sendEdgeAppLogs() {
	for {
		log.Debug("Check if new logs available for edge apps")
		knownManifests := manifest.GetKnownManifests()
		until := time.Now().UTC().Format(time.RFC3339Nano)

		for _, manif := range knownManifests {
			if manif.Status != model.EdgeAppUndeployed {
				dataservice.SendEdgeAppLogs(manif, until)
			}
		}

		time.Sleep(time.Second * time.Duration(config.GetEdgeAppLogIntervalSec()))
	}
}
