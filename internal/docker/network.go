package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
)

func ReadAllNetworks() []types.NetworkResource {
	log.Debug("Docker_container -> ReadAllNetworks")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}

	networks, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}

	return networks
}

func GetNetworkName(m model.Manifest) string {
	name := m.Manifest.Search("name").Data().(string)
	networkName := ""

	// Manifest name
	length := 11
	if len(name) <= 0 {
		return ""
	} else if len(name) > length {
		name = name[:length]
	}

	// Get last network count
	networks := ReadAllNetworks()
	if len(networks) > 0 {
		count := 0
		// Retreive last created container count
		for _, network := range networks {
			netLastCount := network.Name[len(network.Name)-3:]
			netCount, _ := strconv.Atoi(netLastCount)
			if netCount > count {
				count = netCount
			}
		}

		if count == 0 {
			// This is first container
			networkName = fmt.Sprint(name, "_001")
		} else if count < 9 {
			networkName = fmt.Sprint(name, "_00", count+1)
		} else if count < 99 {
			networkName = fmt.Sprint(name, "_0", count+1)
		} else {
			networkName = fmt.Sprint(name, "_", count+1)
		}
	} else {
		networkName = fmt.Sprint(name, "_001")
	}

	return strings.ReplaceAll(networkName, " ", "")
}

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
