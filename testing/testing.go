package testing

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/util"
)

func init() {
	err := os.Chdir(util.GetExeDir())
	if err != nil {
		log.Error("Error:\tCould not move into the directory, Error:\t %v", err)
	}
}
