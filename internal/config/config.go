package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var ConfigPath string
var params Params

type Params struct {
	Password          string
	APIkey            string
	Registered        bool
	EdgeAppLogInvlSec int
	Broker            string
	Heartbeat         int
	MqttLogs          bool
	NoTLS             bool
	LogLevel          string
	LogFileName       string
	LogSize           int
	LogAge            int
	LogBackup         int
	LogCompress       bool
	LogSendInvl       int
	NodeId            string
	NodeName          string
	RootCertPath      string
}

func GetRootCertPath() string {
	return params.RootCertPath
}

func GetNodeId() string {
	return params.NodeId
}

func GetPassword() string {
	return params.Password
}

func GetRegistered() bool {
	return params.Registered
}

func GetEdgeAppLogIntervalSec() int {
	return params.EdgeAppLogInvlSec
}

func writeNodeConfigToFile() {
	encodedJson, err := json.MarshalIndent(params, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(ConfigPath, encodedJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func readNodeConfigFromFile() {
	jsonFile, err := os.Open(ConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	decoder := json.NewDecoder(jsonFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	if err != nil {
		log.Fatal(err)
	}
}

func UpdateNodeConfig(opt model.Params) Params {
	const defaultNodeName = "New Node"
	const maxNumNodes = 10000

	readNodeConfigFromFile()

	var configChanged bool
	if opt.Broker != "" {
		params.Broker = opt.Broker
		configChanged = true
	}

	if opt.Heartbeat > 0 {
		params.Heartbeat = opt.Heartbeat
		configChanged = true
	}

	if opt.MqttLogs {
		params.MqttLogs = opt.MqttLogs
		configChanged = true
	}

	if opt.NoTLS {
		params.NoTLS = opt.NoTLS
		configChanged = true
	}

	if opt.LogLevel != "" {
		params.LogLevel = opt.LogLevel
		configChanged = true
	}

	if opt.LogFileName != "" {
		params.LogFileName = opt.LogFileName
		configChanged = true
	}

	if opt.LogSize > 0 {
		params.LogSize = opt.LogSize
		configChanged = true
	}

	if opt.LogAge > 0 {
		params.LogAge = opt.LogAge
		configChanged = true
	}

	if opt.LogBackup > 0 {
		params.LogBackup = opt.LogBackup
		configChanged = true
	}

	if opt.LogCompress {
		params.LogCompress = opt.LogCompress
		configChanged = true
	}

	if opt.LogSendInvl > 0 {
		params.EdgeAppLogInvlSec = opt.LogSendInvl
		configChanged = true
	}

	if opt.NodeId != "" {
		params.NodeId = opt.NodeId
		configChanged = true
	}

	if opt.NodeName != "" {
		params.NodeName = opt.NodeName
		configChanged = true
	} else {
		// randomize the default node name from the config file
		if params.NodeName == "" || params.NodeName == defaultNodeName {
			params.NodeName = fmt.Sprintf(defaultNodeName+"%d", rand.Intn(maxNumNodes))
			configChanged = true
		}
	}

	if opt.RootCertPath != "" {
		params.RootCertPath = opt.RootCertPath
		configChanged = true
	}

	if configChanged {
		writeNodeConfigToFile()
	}

	return params
}

func SetRegistered(registered bool) {
	params.Registered = registered

	writeNodeConfigToFile()
}
