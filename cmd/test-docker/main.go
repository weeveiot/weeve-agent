package main

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)



func main( ) {
	log.SetLevel(log.DebugLevel)
	networkName := "bridge-net-test"
	imageName1 := "weevenetwork/mosquitto_broker"
	containerName1 := "mosquitto_broker"


	// DOCKER CLIENT //////////
	log.Debug("Build context and client")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		panic(err)
	}

	// CREATE THE NETWORK ////
	// Network options
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	// Create it
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Debug("Network ", networkName, " already exists")
	} else {
		log.Debug("Created network", networkName)
	}

	///////////////////////////////////////
	// CREATE AND ATTACH CONTAINER 1 //////
	///////////////////////////////////////

	// Config
	containerConfig := &container.Config{
		Image:        imageName1,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          nil,
		Tty:          false,
		ExposedPorts: nil,
	}

	hostConfig := &container.HostConfig{
		PortBindings: nil,
		NetworkMode: "bridge",
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	// Create it
	containerCreateResponse, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName1)
	if err != nil {
		panic(err)
	}
	log.Debug("Created container " + containerName1)

	// START CONTAINER /////
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		panic("Failed to start container")
	}
	log.Debug("Started container")

	// ATTACH TO NETWORK
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)

}