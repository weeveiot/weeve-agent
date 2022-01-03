package testing

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const RootPath = "/"

func init() {
	dir := filepath.Join(filepath.Dir(os.Args[1]) + RootPath)
	Root, err := filepath.Abs(dir)
	if err != nil {
		log.Error("Error:\tCould not find root directory, Error:\t %v", err)
	}

	err = os.Chdir(Root)
	if err != nil {
		log.Error("Error:\tCould not move into the directory, Error:\t %v", err)
	}
}
