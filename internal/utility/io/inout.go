package ioutility

import (
	"os"
	"path/filepath"
	"strings"

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

func FirstToUpper(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(string(str[0])) + str[1:]
}
