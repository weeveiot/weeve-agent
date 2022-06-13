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

var Params struct {
	RootCertPath string
	CertPath     string
	KeyPath      string
	NodeId       string
	NodeName     string
	Registered   bool
}

func GetRootCertPath() string {
	return Params.RootCertPath
}

func GetCertPath() string {
	return Params.CertPath
}

func GetNodeId() string {
	return Params.NodeId
}

func GetNodeName() string {
	return Params.NodeName
}

func GetKeyPath() string {
	return Params.KeyPath
}

func WriteNodeConfigToFile() {
	encodedJson, err := json.MarshalIndent(Params, "", " ")
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
	err = decoder.Decode(&Params)
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
		Params.NodeId = opt.NodeId
		configChanged = true
	}

	if len(opt.RootCertPath) > 0 {
		Params.RootCertPath = opt.RootCertPath
		configChanged = true
	}

	if len(opt.CertPath) > 0 {
		Params.CertPath = opt.CertPath
		configChanged = true
	}

	if len(opt.KeyPath) > 0 {
		Params.KeyPath = opt.KeyPath
		configChanged = true
	}

	if len(opt.NodeName) > 0 {
		Params.NodeName = opt.NodeName
		configChanged = true
	} else {
		// randomize the default node name from the config file
		if Params.NodeName == "" || Params.NodeName == defaultNodeName {
			Params.NodeName = fmt.Sprintf(defaultNodeName+"%d", rand.Intn(maxNumNodes))
			configChanged = true
		}
	}

	log.Debugf("Set node config to following params: %+v", Params)
	if configChanged {
		WriteNodeConfigToFile()
	}
}

func SetNodeId(nodeId string) {
	Params.NodeId = nodeId

	WriteNodeConfigToFile()
}

func SetCertPath(certificatePath, keyPath string) {
	Params.CertPath = certificatePath
	Params.KeyPath = keyPath

	WriteNodeConfigToFile()
}

func IsNodeRegistered() bool {
	return Params.Registered
}
