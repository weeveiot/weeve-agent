package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
)

func DownloadCertificates(certificateUrl, keyUrl string) (string, string, error) {

	log.Info("Downloading certificates and keys ...")

	certDir := filepath.Dir(config.GetCertPath())
	certificatePath, err := downloadFile(certificateUrl, certDir)
	if err != nil {
		return "", "", err
	}

	keyDir := filepath.Dir(config.GetKeyPath())
	keyPath, err := downloadFile(keyUrl, keyDir)
	if err != nil {
		return "", "", err
	}

	return certificatePath, keyPath, nil
}

func downloadFile(url, dir string) (string, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.New("Download failed. " + err.Error())
	}

	defer resp.Body.Close()

	// Create a new file to put the certificate in
	fileName := filepath.Base(resp.Request.URL.Path)
	fullPath := path.Join(dir, fileName)
	out, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	log.Info("Downloaded ", fileName, ". Writing it into ", dir, "...")

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}
