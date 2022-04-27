package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var ConfigPath string

var params struct {
	RootCertPath string
	CertPath     string
	NodeId       string
	NodeName     string
	KeyPath      string
}

const MAX_NUM_NODES = 10000

func GetRootCertPath() string {
	return params.RootCertPath
}

func GetCertPath() string {
	return params.CertPath
}

func GetNodeId() string {
	return params.NodeId
}

func GetNodeName() string {
	return params.NodeName
}

func GetKeyPath() string {
	return params.KeyPath
}

func WriteNodeConfigToFile() {
	file, err := json.MarshalIndent(params, "", " ")
	if err != nil {
		log.Error(err)
	}

	err = ioutil.WriteFile(ConfigPath, file, 0644)
	if err != nil {
		log.Error(err)
	}
}

func ReadNodeConfigFromFile() {
	jsonFile, err := os.Open(ConfigPath)
	if err != nil {
		log.Fatalf("Unable to open node configuration file: %v", err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Unable to read node configuration file: %v", err)
	}

	json.Unmarshal(byteValue, &params)
}

func UpdateNodeConfig(opt model.Params) {
	ReadNodeConfigFromFile()

	var configChanged bool
	if len(opt.NodeId) > 0 {
		params.NodeId = opt.NodeId
		configChanged = true
	}

	if len(opt.RootCertPath) > 0 {
		params.RootCertPath = opt.RootCertPath
		configChanged = true
	}

	if len(opt.CertPath) > 0 {
		params.CertPath = opt.CertPath
		configChanged = true
	}

	if len(opt.KeyPath) > 0 {
		params.KeyPath = opt.KeyPath
		configChanged = true
	}

	if len(opt.NodeName) > 0 {
		params.NodeName = opt.NodeName
		configChanged = true
	} else {
		// randomize the default node name from the config file
		if params.NodeName == "" || params.NodeName == "New Node" {
			params.NodeName = fmt.Sprintf("New Node%d", rand.Intn(MAX_NUM_NODES))
			configChanged = true
		}
	}

	if configChanged {
		WriteNodeConfigToFile()
	}
}

func SetNodeId(nodeId string) {
	params.NodeId = nodeId

	WriteNodeConfigToFile()
}

func SetCertPath(certificatePath, keyPath string) {
	params.CertPath = certificatePath
	params.KeyPath = keyPath

	WriteNodeConfigToFile()
}
