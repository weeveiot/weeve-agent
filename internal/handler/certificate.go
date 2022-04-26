package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const NodeConfigFile = "nodeconfig.json"
const KeyCertificate = "Certificate"
const KeyPrivateKey = "PrivateKey"
const KeyNodeId = "NodeId"
const KeyNodeName = "NodeName"
const KeyAWSRootCert = "AWSRootCert"
const CertDirName = "certs"

var ConfigPath string
var Opt model.Params

func DownloadCertificates(payload []byte) map[string]string {

	log.Info("Downloading certificates ...")

	json, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message: ", err)
	}

	certificates := map[string]string{
		KeyCertificate: json.Search(KeyCertificate).Data().(string),
		KeyPrivateKey:  json.Search(KeyPrivateKey).Data().(string),
	}

	nodeConfig := ReadNodeConfig()

	for key, certUrl := range certificates {

		// Get the data
		resp, err := http.Get(certUrl)
		if err != nil {
			log.Error("Error to download certificate: ", err)
			return nil
		}

		defer resp.Body.Close()

		// Create a new file to put the certificate in
		fileName := filepath.Base(resp.Request.URL.Path)
		certDir := filepath.Dir(nodeConfig[key])
		fileNameWithPath := path.Join(certDir, fileName)
		out, err := os.Create(fileNameWithPath)
		if err != nil {
			log.Error("Error to create file: ", fileName, err)
			return nil
		}
		defer out.Close()

		log.Info("Downloaded ", fileName, ". Writing it into ", certDir, "...")

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			log.Error("Error to copy file: ", fileName, err)
			return nil
		}

		certificates[key] = fileNameWithPath
	}

	return certificates
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) {

	nodeConfig := map[string]string{
		KeyNodeId:      nodeId,
		KeyCertificate: certificates[KeyCertificate],
		KeyPrivateKey:  certificates[KeyPrivateKey],
	}

	WriteToNodeConfig(nodeConfig)
}

func WriteToNodeConfig(attrs map[string]string) {
	configs := ReadNodeConfig()

	for k, v := range attrs {
		log.Debug(k, " value is ", v)
		configs[k] = v
	}

	file, _ := json.MarshalIndent(configs, "", " ")

	_ = ioutil.WriteFile(ConfigPath, file, 0644)
}

func ReadNodeConfig() map[string]string {
	jsonFile, err := os.Open(ConfigPath)
	if err != nil {
		log.Fatalf("Unable to open node configuration file: %v", err)
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Unable to read node configuration file: %v", err)
	}

	var config map[string]string

	json.Unmarshal(byteValue, &config)

	return config
}

func UpdateNodeConfig(nodeConfigs map[string]string) {
	var configChanged bool
	nodeConfig := map[string]string{}
	if Opt.NodeId != "register" {
		nodeConfig[KeyNodeId] = Opt.NodeId
		configChanged = true
	}

	if len(Opt.RootCertPath) > 0 {
		nodeConfig[KeyAWSRootCert] = Opt.RootCertPath
		configChanged = true
	}

	if len(Opt.CertPath) > 0 {
		nodeConfig[KeyCertificate] = Opt.CertPath
		configChanged = true
	}

	if len(Opt.KeyPath) > 0 {
		nodeConfig[KeyPrivateKey] = Opt.KeyPath
		configChanged = true
	}

	if len(Opt.NodeName) > 0 {
		nodeConfig[KeyNodeName] = Opt.NodeName
		configChanged = true
	} else {
		nodeNm := nodeConfigs[KeyNodeName]
		if nodeNm == "" {
			nodeNm = "New Node"
		}
		if nodeNm == "New Node" {
			nodeNm = fmt.Sprintf("%s%d", nodeNm, mathrand.Intn(10000))
			nodeConfig[KeyNodeName] = nodeNm
			configChanged = true
		}
	}

	if configChanged {
		WriteToNodeConfig(nodeConfig)
	}
}

func ValidateBroker(u *url.URL) {
	// OPTION: Parse and validate the Broker url

	host, port, _ := net.SplitHostPort(u.Host)

	// Strictly require protocol and host in Broker specification
	if (len(strings.TrimSpace(host)) == 0) || (len(strings.TrimSpace(u.Scheme)) == 0) {
		log.Fatal("Error in --broker option: Specify both protocol:\\\\host in the Broker URL")
	}

	log.Info(fmt.Sprintf("Broker host->%v at port->%v over %v", host, port, u.Scheme))

	log.Debug("Broker >> ", Opt.Broker)
}
