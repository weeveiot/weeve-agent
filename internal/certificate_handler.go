package internal

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

const NodeConfigFile = "nodeconfig.json"
const KeyCertificate = "Certificate"
const KeyPrivateKey = "PrivateKey"
const KeyNodeId = "NodeId"
const KeyNodeName = "NodeName"
const KeyAWSRootCert = "AWSRootCert"
const CertDirName = "certs"

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

	for key, certUrl := range certificates {

		// Get the data
		resp, err := http.Get(certUrl)
		if err != nil {
			log.Error("Error to download certificate: ", err)
			return nil
		}

		defer resp.Body.Close()

		// Create a new dir for certificates if not already created
		certDir := path.Join(util.GetExeDir(), CertDirName)
		err = os.MkdirAll(certDir, 0700)
		if err != nil {
			log.Error("Could not create the directory ", certDir, err)
		}

		// Create a new file to put the certificate in
		fileName := filepath.Base(resp.Request.URL.Path)
		fileNameWithPath := path.Join(certDir, fileName)
		out, err := os.Create(fileNameWithPath)
		if err != nil {
			log.Error("Error to create file: ", fileName, err)
			return nil
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			log.Error("Error to copy file: ", fileName, err)
			return nil
		}

		certificates[key] = fileName
	}

	return certificates
}

func CheckIfNodeAlreadyRegistered() bool {

	config := ReadNodeConfig()
	return len(config[KeyNodeId]) > 0
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) {

	nodeConfig := map[string]string{
		KeyNodeId:      nodeId,
		KeyCertificate: certificates[KeyCertificate],
		KeyPrivateKey:  certificates[KeyPrivateKey],
	}

	UpdateNodeConfig(nodeConfig)
}

func UpdateNodeConfig(attrs map[string]string) {
	configs := ReadNodeConfig()

	for k, v := range attrs {
		log.Debug(k, "value is", v)
		configs[k] = v
	}

	file, _ := json.MarshalIndent(configs, "", " ")

	NodeConfigFilePath := path.Join(util.GetExeDir(), NodeConfigFile)
	_ = ioutil.WriteFile(NodeConfigFilePath, file, 0644)
}

func ReadNodeConfig() map[string]string {
	NodeConfigFilePath := path.Join(util.GetExeDir(), NodeConfigFile)

	jsonFile, err := os.Open(NodeConfigFilePath)
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
