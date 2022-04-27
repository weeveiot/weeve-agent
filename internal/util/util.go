package util

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func GetExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Could not get the path to the executable.")
	}
	dir := filepath.Dir(exePath)
	return dir
}

func LoadJsonBytes(manifestFileName string) []byte {
	manifestPath := path.Join(GetExeDir(), manifestFileName)

	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}
