package main

import (
	"fmt"
	"io"
	golog "log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/edgeapp"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/secret"
)

type PlainFormatter struct {
	TimestampFormat string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s %s : %s\n", timestamp, entry.Level, entry.Message)), nil
}

var logFormatter = &PlainFormatter{
	TimestampFormat: "2006-01-02 15:04:05",
}

func init() {
	log.SetFormatter(logFormatter)
}

func main() {
	logToStdout, localManifest, deleteNode := parseCLIoptions()
	setupLogging(logToStdout)

	err := manifest.InitKnownManifests()
	if err != nil {
		log.Fatal("Initialization of known manifests failed! CAUSE --> ", err)
	}

	nodePubKey, err := secret.InitNodeKeypair()
	if err != nil {
		log.Fatal("Initialization of node keypair failed! CAUSE --> ", err)
	}

	docker.SetupDockerClient()

	if localManifest != "" {
		err := edgeapp.ReadDeployManifestLocal(localManifest)
		if err != nil {
			log.Fatal("Deployment of the local manifest failed! CAUSE --> ", err)
		}
	}

	err = com.RegisterNode()
	if err != nil {
		log.Fatal("Node registration failed! CAUSE --> ", err)
	}

	err = com.ConnectNode(setSubscriptionHandlers())
	if err != nil {
		log.Fatal("Failed to connect node! CAUSE --> ", err)
	}

	if deleteNode {
		handler.DeleteNode(model.NodeDisconnected)
		os.Exit(0)
	}

	edgeapp.SetNodeStatus(model.NodeConnected)

	err = com.SendNodePublicKey(nodePubKey)
	if err != nil {
		log.Fatal("Sending node public key failed! CAUSE --> ", err)
	}

	// Kill the agent on a keyboard interrupt
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// Start threads to send status messages
	go monitorEdgeAppStatus()
	go sendHeartbeat()
	go sendEdgeAppLogs()

	log.Info("Weeve-agent started and running...")
	// Cleanup on ending the process
	<-done
	err = com.DisconnectNode()
	if err != nil {
		log.Fatal("Disconnection of node failed! CAUSE --> ", err)
	}
}

func parseCLIoptions() (bool, string, bool) {
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

	if opt.Version {
		fmt.Println("weeve agent -", model.Version)
		os.Exit(0)
	}

	config.Set(opt)

	return opt.Stdout, opt.ManifestPath, opt.Delete
}

func setupLogging(toStdout bool) {
	l, _ := log.ParseLevel(config.Params.LogLevel)
	log.SetLevel(l)

	logFile := &lumberjack.Logger{
		Filename:   filepath.ToSlash(config.Params.LogFileName),
		MaxSize:    config.Params.LogSize,
		MaxAge:     config.Params.LogAge,
		MaxBackups: config.Params.LogBackup,
		Compress:   config.Params.LogCompress,
	}

	var logOutput io.Writer

	if toStdout {
		logOutput = io.MultiWriter(os.Stdout, logFile)
	} else {
		logOutput = logFile
	}
	log.SetOutput(logOutput)

	// create a logger that's not going to send it's messages to MQTT broker for the cases when it's not possible
	com.CreateMQTTLogger(logOutput, logFormatter, l)

	mqtt.ERROR = golog.New(logOutput, "error [MQTT]: ", golog.LstdFlags|golog.Lmsgprefix)
	mqtt.CRITICAL = golog.New(logOutput, "crit [MQTT]: ", golog.LstdFlags|golog.Lmsgprefix)
	mqtt.WARN = golog.New(logOutput, "warn [MQTT]: ", golog.LstdFlags|golog.Lmsgprefix)
	if config.Params.MqttLogs {
		mqtt.DEBUG = golog.New(logOutput, "debug [MQTT]: ", golog.LstdFlags|golog.Lmsgprefix)
	}

	log.Info("weeve agent - ", model.Version)
	log.Info("Started logging")
	log.Info("Logging level set to ", log.GetLevel())
}

func setSubscriptionHandlers() map[string]mqtt.MessageHandler {
	subscriptions := make(map[string]mqtt.MessageHandler)

	subscriptions[com.TopicOrchestration] = handler.OrchestrationHandler
	subscriptions[com.TopicOrgPrivateKey] = handler.OrgPrivKeyHandler
	subscriptions[com.TopicNodeDelete] = handler.NodeDeleteHandler

	return subscriptions
}

func monitorEdgeAppStatus() {
	log.Debug("Start monitering edge app status...")

	edgeApps, err := edgeapp.GetEdgeAppStatus()
	if err != nil {
		log.Error("GetEdgeAppStatus failed! CAUSE --> ", err)
	}

	for {
		time.Sleep(time.Second * time.Duration(5))
		latestEdgeApps, statusChange, err := edgeapp.CompareEdgeAppStatus(edgeApps)
		if err != nil {
			log.Error("CompareEdgeAppStatus failed! CAUSE --> ", err)
			continue
		}
		log.Debug("Latest edge app status: ", latestEdgeApps)

		if statusChange {
			err := edgeapp.SendStatus()
			if err != nil {
				log.Error("SendStatus failed! CAUSE --> ", err)
				continue
			}
			edgeApps = latestEdgeApps
		}
	}
}

func sendHeartbeat() {
	log.Debug("Start sending heartbeats...")

	for {
		err := edgeapp.SendStatus()
		if err != nil {
			log.Error("SendStatus failed! CAUSE --> ", err)
		}

		time.Sleep(time.Second * time.Duration(config.Params.Heartbeat))
	}
}

func sendEdgeAppLogs() {
	log.Debug("Start sending edge app logs...")

	for {
		knownManifests := manifest.GetKnownManifests()
		until := time.Now().UTC().Format(time.RFC3339Nano)

		for _, manif := range knownManifests {
			if manif.Status != model.EdgeAppUndeployed {
				edgeapp.SendEdgeAppLogs(*manif, until)
			}
		}

		time.Sleep(time.Second * time.Duration(config.Params.LogSendInvl))
	}
}
