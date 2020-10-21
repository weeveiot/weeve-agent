package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	// "github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"github.com/golang/gddo/httputil/header"
)

//TODO: Add the code for instantiating a pipeline in the node:
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func BuildPipeline(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	// Enforce content type exists
	if r.Header.Get("Content-Type") == "" {
		msg := "Content-Type header is not application/json"
		log.Error(msg)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	// Enforce content type is application/json
	// Note that we are using the gddo/httputil/header
	// package to parse and extract the value here, so the check works
	// even if the client includes additional charset or boundary
	// information in the header.
	value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if value != "application/json" {
		msg := "Content-Type header is not application/json"
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	// Now handle the payload, start by converting to []bytes
	// log.Debug("Raw POST body:", r.Body)
	// bodyBytes, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	log.Error(err)
	// 	msg := "Error in decoding JSON payload, check for valid JSON"
	// 	http.Error(w, msg, http.StatusBadRequest)
	// 	return
	// }
	// log.Debug("POST body as string:", string(bodyBytes))

	manifest := &model.ManifestReq{}

	err := json.NewDecoder(r.Body).Decode(manifest)
	if err != nil {
		fmt.Println(err)
	}

	log.Debug("Recieved manifest: ", manifest.Name)
	log.Debug("Number of modules: ", len(manifest.Modules))
	// m type ManifestReq struct {
	// 	ID      string     `json:"ID"`
	// 	Name    string     `json:"Name"`
	// 	Modules []Manifest `json:"Modules"`
	// }

	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)
		// image := docker.ReadImage(mod.ImageID)
		// log.Debug("Image: ", image)
		exists := docker.ImageExists(mod.ImageID)
		log.Debug("\tImage exists: ", exists)
	}

	// // Finally, convert the JSON bytes into a simplejson object
	// bodyJSON, err := simplejson.NewJson(bodyBytes)
	// if err != nil {
	// 	log.Error(err)
	// 	msg := "Error in decoding JSON payload, check for valid JSON"
	// 	http.Error(w, msg, http.StatusBadRequest)
	// 	return
	// }
	// log.Debug("POST body as simplejson:", bodyJSON)

	// bodyJSON

	// Now, assert keys
	// TODO: Move this into a proper schema parser
	// log.Debug(bodyJSON.Get("node"))
	// nodes = bodyJSON.MustArray("nodes")
	// fmt.Println(nodes)
	// log.Debug(bodyJSON.MustArray("node"))

}
