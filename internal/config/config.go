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
	NodeId       string
	Password     string
	APIkey       string
	NodeName     string
	Registered   bool
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

func GetAPIkey() string {
	return params.APIkey
}

func GetNodeName() string {
	return params.NodeName
}

func GetRegistered() bool {
	return params.Registered
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

	decoder := json.NewDecoder(jsonFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
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

	log.Debugf("Set node config to following params: %+v", params)
	if configChanged {
		writeNodeConfigToFile()
	}
}

func SetNodeId(nodeId string) {
	params.NodeId = nodeId

	writeNodeConfigToFile()
}

func SetRegistered(registered bool) {
	params.Registered = registered

	writeNodeConfigToFile()
}
