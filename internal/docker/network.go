package docker

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func AttachContainerNetwork(containerID string, networkName string) error {
	log.Debug("Build context and client")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		panic(err)
	}

	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", containerID, "to network", networkName)
	return nil
}
