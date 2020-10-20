package controller

import (
	"net/http"
	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
)

func Status(w http.ResponseWriter, r *http.Request) {
	json := simplejson.New()
	json.Set("status", "ok")
	json.Set("name", "Edge Pipeline Service")
	json.Set("location", "SIMULATION")
	json.Set("version", "0.0.1")
	payload, err := json.MarshalJSON()
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}