package internal

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"

	"os"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

const NodeConfigFileName = "nodeconfig.json"
const CertificateKey = "Certificate"
const PrivateKeyKay = "PrivateKey"
const NodeIdKey = "NodeId"
const AWSRootCertKey = "AWSRootCert"
const RootPath = "/"

func DownloadCertificates(payload []byte) map[string]string {

	log.Info("Downloading certificates ...")

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message: ", err)
	}

	json := *jsonParsed
	certificates := map[string]string{
		CertificateKey: json.Search(CertificateKey).Data().(string),
		PrivateKeyKay:  json.Search(PrivateKeyKay).Data().(string),
	}

	for key, certPath := range certificates {

		// Get the data
		resp, err := http.Get(certPath)
		if err != nil {
			log.Error("Error to download certificate: ", err)
			return nil
		}

		defer resp.Body.Close()

		// Root folder of this project
		dir := filepath.Join(filepath.Dir(os.Args[1]) + RootPath)
		Root, err := filepath.Abs(dir)
		if err != nil {
			return nil
		}
		fileName := filepath.Base(resp.Request.URL.Path)
		fileNameWithPath := path.Join(Root, fileName)

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
	return len(config[NodeIdKey]) > 0
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) bool {

	nodeConfig := map[string]string{
		NodeIdKey:      nodeId,
		CertificateKey: certificates[CertificateKey],
		PrivateKeyKay:  certificates[PrivateKeyKay],
	}

	UpdateNodeConfig(nodeConfig)
	return true
}

func UpdateNodeConfig(attrs map[string]string) bool {
	configs := ReadNodeConfig()

	for k, v := range attrs {
		log.Debug(k, "value is", v)
		configs[k] = v
	}

	file, _ := json.MarshalIndent(configs, "", " ")

	// Root folder of this project
	dir := filepath.Join(filepath.Dir(os.Args[1]) + RootPath)
	Root, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	NodeConfigFilePath := path.Join(Root, NodeConfigFileName)
	_ = ioutil.WriteFile(NodeConfigFilePath, file, 0644)

	return true
}

func ReadNodeConfig() map[string]string {
	// Root folder of this project
	dir := filepath.Join(filepath.Dir(os.Args[1]) + RootPath)
	Root, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	NodeConfigFilePath := path.Join(Root, NodeConfigFileName)

	// Open our jsonFile
	jsonFile, err := os.Open(NodeConfigFilePath)
	if err != nil {
		log.Fatalf("Unable to open node configuration file: %v", err)
	}
	// read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Unable to read node configuration file: %v", err)
	}

	// we initialize our Users array
	var config map[string]string

	// unmarshal byteArray
	json.Unmarshal(byteValue, &config)

	return config
}
