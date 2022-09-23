package ioutility

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/weeveiot/weeve-agent/internal/logger"
)

func GetExeDir() string {
	exePath, err := os.Getwd()
	if err != nil {
		logger.Log.Fatal("Could not get the path to the executable.")
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
