package ioutility

import (
	"os"
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
