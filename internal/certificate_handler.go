package internal

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

func DownloadCertificates(payload []byte) map[string]string {

	log.Info("Downloading certificates ...")

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message: ", err)
	}

	certificates := map[string]string{
		constants.KeyCertificate: jsonParsed.Search(constants.KeyCertificate).Data().(string),
		constants.KeyPrivateKey:  jsonParsed.Search(constants.KeyPrivateKey).Data().(string),
	}

	for key, certUrl := range certificates {

		// Get the data
		resp, err := http.Get(certUrl)
		if err != nil {
			log.Error("Error to download certificate: ", err)
			return nil
		}

		defer resp.Body.Close()

		dir, _ := os.Getwd()
		fileName := filepath.Base(resp.Request.URL.Path)
		fileNameWithPath := path.Join(dir, "..", "..", fileName)

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
	return len(config[constants.KeyNodeId]) > 0
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) bool {

	nodeConfig := map[string]string{
		constants.KeyNodeId:      nodeId,
		constants.KeyCertificate: certificates[constants.KeyCertificate],
		constants.KeyPrivateKey:  certificates[constants.KeyPrivateKey],
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

	dir, _ := os.Getwd()
	NodeConfigFilePath := path.Join(dir, "..", "..", constants.NodeConfigFile)
	_ = ioutil.WriteFile(NodeConfigFilePath, file, 0644)

	return true
}

func ReadNodeConfig() map[string]string {
	dir, _ := os.Getwd()
	NodeConfigFilePath := path.Join(dir, "..", "..", constants.NodeConfigFile)

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
