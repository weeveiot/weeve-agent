package controller

import (
	"net/http"
	log "github.com/sirupsen/logrus"
)

//TODO: Add the code for instantiating a pipeline in the node:
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func BuildPipeline(w http.ResponseWriter, r *http.Request) {
	log.Debug("asdf")
}