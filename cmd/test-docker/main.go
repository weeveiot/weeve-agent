package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)


func main( ) {
	networkName := "bridge-net-test"
	imageName := "eclipse-mosquitto:latest"
	containerName := "c1"

	// CLIENT
	log.Debug("Build context and client")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		// log.Error(err)
		return false
	}

	// CREATE THE NETWORK
	log.Debug("Create the network", networkName)
	// Network options
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	fmt.Println(networkCreateOptions)
	// Create it
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		panic(err)
	}
	log.Debug("Created network", networkName)
	log.Debug(resp.ID, resp.Warning)


	// CONFIG CONTAINER
	log.Debug("Container configuration object for ContainerCreate()")
	containerConfig := &container.Config{
		Image:        imageName,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          nil,
		Tty:          false,
		ExposedPorts: nil,
	}

	log.Debug("Host configuration object for ContainerCreate()")
	hostConfig := &container.HostConfig{
		PortBindings: nil,
		NetworkMode: "bridge",
	}

	log.Debug("Network configuration object for ContainerCreate()")
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	// CREATE THE CONTAINER
	containerCreateResponse, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName)
	if err != nil {
		log.Error(err)
	}
	log.Debug("Created container " + containerName)

	// ATTACH TO NETWORK
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)

}