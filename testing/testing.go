package testing

import (
	"os"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		log.Error("Error:\tCould not move into the directory (%s)\n Error:\t %v", dir, err)
	}
}
