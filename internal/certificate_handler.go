package internal

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"runtime"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"fmt"
	"os"
)

const NodeConfigFileName = "nodeconfig.json"
const CertificateKey = "Certificate"
const PrivateKeyKay = "PrivateKey"
const NodeIdKey = "NodeId"
const AWSRootCertKey = "AWSRootCert"

func DownloadCertificates(payload []byte) map[string]string {

	fmt.Println("Downloading certificates")

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
		_, b, _, _ := runtime.Caller(0)
		Root := filepath.Join(filepath.Dir(b), "../")
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
	data := map[string]string{
		NodeIdKey:      nodeId,
		CertificateKey: certificates[CertificateKey],
		PrivateKeyKay:  certificates[PrivateKeyKay],
		AWSRootCertKey: "AmazonRootCA1.pem",
	}

	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile(NodeConfigFileName, file, 0644)

	return true
}

func ReadNodeConfig() map[string]string {
	// Open our jsonFile
	jsonFile, err := os.Open(NodeConfigFileName)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var config map[string]string

	// unmarshal byteArray
	json.Unmarshal(byteValue, &config)

	return config
}
