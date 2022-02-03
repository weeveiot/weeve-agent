package testing

import (
	"os"

	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

func init() {
	err := os.Chdir(util.GetExeDir())
	if err != nil {
		log.Error("Error:\tCould not move into the directory, Error:\t %v", err)
	}
}
