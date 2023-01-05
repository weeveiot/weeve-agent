package config

import (
	"encoding/json"
	"net"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/model"
)

type ParamStruct struct {
	Broker       string
	NodeId       string
	NodeName     string
	NoTLS        bool
	Password     string
	RootCertPath string
	LogLevel     string
	LogFileName  string
	LogSize      int
	LogAge       int
	LogBackup    int
	LogCompress  bool
	MqttLogs     bool
	Heartbeat    int
	LogSendInvl  int
}

// default values
var Params = ParamStruct{
	NoTLS:        false,
	Password:     "",
	RootCertPath: "ca.crt",
	LogLevel:     "info",
	LogFileName:  "Weeve_Agent.log",
	LogSize:      1,
	LogAge:       1,
	LogBackup:    5,
	LogCompress:  false,
	MqttLogs:     false,
	Heartbeat:    10,
	LogSendInvl:  60,
}

func Set(opt model.Params) {
	if opt.ConfigPath != "" {
		log.Info("Loading config file from ", opt.ConfigPath)
		readNodeConfigFromFile(opt.ConfigPath)
	}
	applyCLIparams(opt)
	validateConfig()
	log.Infof("Set node config to following params: %+v", Params)
}

func readNodeConfigFromFile(configPath string) {
	jsonFile, err := os.Open(configPath)
	if err != nil {
		log.Fatal("Failed to open config file! CAUSE --> ", err)
	}
	defer jsonFile.Close()

	decoder := json.NewDecoder(jsonFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&Params)
	if err != nil {
		log.Fatal("Failed to parse config params! CAUSE --> ", err)
	}
}

func applyCLIparams(opt model.Params) {
	if opt.Broker != "" {
		Params.Broker = opt.Broker
	}

	if opt.NodeId != "" {
		Params.NodeId = opt.NodeId
	}

	if opt.NodeName != "" {
		Params.NodeName = opt.NodeName
	}

	if opt.NoTLS {
		Params.NoTLS = opt.NoTLS
	}

	if opt.Password != "" {
		Params.Password = opt.Password
	}

	if opt.RootCertPath != "" {
		Params.RootCertPath = opt.RootCertPath
	}

	if opt.LogLevel != "" {
		Params.LogLevel = opt.LogLevel
	}

	if opt.LogFileName != "" {
		Params.LogFileName = opt.LogFileName
	}

	if opt.LogSize > 0 {
		Params.LogSize = opt.LogSize
	}

	if opt.LogAge > 0 {
		Params.LogAge = opt.LogAge
	}

	if opt.LogBackup > 0 {
		Params.LogBackup = opt.LogBackup
	}

	if opt.LogCompress {
		Params.LogCompress = opt.LogCompress
	}

	if opt.MqttLogs {
		Params.MqttLogs = opt.MqttLogs
	}

	if opt.Heartbeat > 0 {
		Params.Heartbeat = opt.Heartbeat
	}

	if opt.LogSendInvl > 0 {
		Params.LogSendInvl = opt.LogSendInvl
	}
}

func validateConfig() {
	if Params.Broker == "" {
		log.Fatal("no broker specified")
	}

	brokerUrl, err := url.Parse(Params.Broker)
	if err != nil {
		log.Fatal("Error on parsing broker ", err)
	}
	validateBrokerUrl(brokerUrl)

	if Params.NoTLS {
		log.Info("TLS disabled!")
	} else {
		if brokerUrl.Scheme != "tls" {
			log.Fatalf("Incorrect protocol, TLS is required unless --notls is set. You specified protocol in broker to: %v", brokerUrl.Scheme)
		}
	}
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
