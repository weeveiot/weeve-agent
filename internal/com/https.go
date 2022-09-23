package com

import (
	"time"

	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/logger"
)

const registrationTimeout = 5

func RegisterNode() error {
	if !config.GetRegistered() {
		logger.Log.Info("Registering node and downloading certificate and key ...")
		time.Sleep(registrationTimeout)
		config.SetRegistered(true)
		// TODO: do node registration stuff
	} else {
		logger.Log.Info("Node already registered!")
	}

	return nil
}
