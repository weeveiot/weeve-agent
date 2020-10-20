package controller

import (
	"io/ioutil"
	"net/http"

	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"

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
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		msg := "Error in decoding JSON payload, check for valid JSON"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Debug("POST body as string:", string(bodyBytes))

	// Finally, convert the JSON bytes into a simplejson object
	bodyJSON, err := simplejson.NewJson(bodyBytes)
	if err != nil {
		log.Error(err)
		msg := "Error in decoding JSON payload, check for valid JSON"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Debug("POST body as simplejson:", bodyJSON)

	// Now, assert keys
	// TODO: Move this into a proper schema parser
	// log.Debug(bodyJSON.Get("node"))
	// nodes = bodyJSON.MustArray("nodes")
	// fmt.Println(nodes)
	// log.Debug(bodyJSON.MustArray("node"))

}
