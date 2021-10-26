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

func DownloadCertificates(payload []byte) map[string]string {

	fmt.Println("Downloading certificates")

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message: ", err)
	}

	json := *jsonParsed
	certificates := map[string]string{
		"Certificate": json.Search("Certificate").Data().(string),
		"PrivateKey":  json.Search("PrivateKey").Data().(string),
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

		certificates[key] = fileNameWithPath
	}

	return certificates
}

func CheckIfNodeAlreadyRegistered() bool {

	// Open our jsonFile
	jsonFile, err := os.Open("nodeconfig.json")
	if err != nil {
		fmt.Println(err)
		return false
	}
	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var config map[string]string

	// unmarshal byteArray
	json.Unmarshal(byteValue, &config)
	if (len(config["NodeId"]) > 0) && (len(config["Certificate"]) > 0) && (len(config["PrivateKey"]) > 0) {
		return true
	}

	return false
}

func MarkNodeRegistered(nodeId string, certificates map[string]string) bool {
	data := map[string]string{
		"NodeId":      nodeId,
		"Certificate": certificates["Certificate"],
		"PrivateKey":  certificates["PrivateKey"],
	}

	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("nodeconfig.json", file, 0644)

	return true
}
