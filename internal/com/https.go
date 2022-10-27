package com

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/config"
)

func RegisterNode() error {
	if config.Params.NodeId == "" {
		// TODO: do node registration stuff
		return errors.New("registration is not implemented at the moment. make sure to provide a valid node id and name")
	} else {
		log.Info("Node already registered!")
	}

	return nil
}
