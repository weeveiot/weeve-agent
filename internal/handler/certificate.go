package handler

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
)

func DownloadCertificates(certificateUrl, keyUrl string) (string, string) {

	log.Info("Downloading certificates and keys ...")

	certDir := filepath.Dir(config.GetCertPath())
	certificatePath := downloadFile(certificateUrl, certDir)

	keyDir := filepath.Dir(config.GetKeyPath())
	keyPath := downloadFile(keyUrl, keyDir)

	return certificatePath, keyPath
}

func downloadFile(url, dir string) string {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		log.Error("Error to download certificate: ", err)
		return ""
	}

	defer resp.Body.Close()

	// Create a new file to put the certificate in
	fileName := filepath.Base(resp.Request.URL.Path)
	fullPath := path.Join(dir, fileName)
	out, err := os.Create(fullPath)
	if err != nil {
		log.Error("Error to create file: ", fileName, err)
		return ""
	}
	defer out.Close()

	log.Info("Downloaded ", fileName, ". Writing it into ", dir, "...")

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Error("Error to copy file: ", fileName, err)
		return ""
	}

	return fullPath
}
