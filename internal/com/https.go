package com

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/config"
)

func RegisterNode() error {
	log.Info("Registering the node...")
	if config.Params.NodeId == "" || config.Params.NodeName == "" {
		// TODO: do node registration stuff
		return errors.New("registration is not implemented at the moment. make sure to provide a valid node id and name")
	} else {
		log.Info("Node already registered!")
	}

	return nil
}
