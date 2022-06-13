package com

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
)

const registrationTimeout = 5

func RegisterNode() error {
	if !config.GetRegistered() {
		log.Info("Registering node and downloading certificate and key ...")
		time.Sleep(registrationTimeout)
		// TODO: do node registration stuff
	} else {
		log.Info("Node already registered!")
	}

	return nil
}
