package testing

import (
	"os"

	log "github.com/sirupsen/logrus"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

func init() {
	err := os.Chdir(ioutility.GetExeDir())
	if err != nil {
		log.Error("Error:\tCould not move into the directory, Error:\t %v", err)
	}
}
