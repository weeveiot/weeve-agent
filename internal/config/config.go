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

func writeNodeConfigToFile() {
	encodedJson, err := json.MarshalIndent(params, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(ConfigPath, encodedJson, 0644)
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

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &params)
	if err != nil {
		log.Fatal(err)
	}
}

func UpdateNodeConfig(opt model.Params) {
	const defaultNodeName = "New Node"
	const maxNumNodes = 10000

	readNodeConfigFromFile()

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
		if params.NodeName == "" || params.NodeName == defaultNodeName {
			params.NodeName = fmt.Sprintf(defaultNodeName+"%d", rand.Intn(maxNumNodes))
			configChanged = true
		}
	}

	if configChanged {
		writeNodeConfigToFile()
	}
}

func SetNodeId(nodeId string) {
	params.NodeId = nodeId

	writeNodeConfigToFile()
}

func SetCertPath(certificatePath, keyPath string) {
	params.CertPath = certificatePath
	params.KeyPath = keyPath

	writeNodeConfigToFile()
}
