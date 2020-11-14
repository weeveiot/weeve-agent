package main

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func startCreateContainer(imageName string, containerName string, entryArgs []string) (container.ContainerCreateCreatedBody, error) {
	// Docker Client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		panic(err)
	}

	// Configuration of a container
	containerConfig := &container.Config{
		Image:        imageName,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          entryArgs,
		Tty:          false,
		ExposedPorts: nil,
	}

	hostConfig := &container.HostConfig{
		PortBindings: nil,
		NetworkMode: "bridge",
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
			MaximumRetryCount: 100,
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}

	// Create the container
	containerCreateResponse, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName)
	if err != nil {
		log.Error(err)
		return container.ContainerCreateCreatedBody{}, err
	}
	log.Debug("Created container " + containerName)

	// Start container
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		// panic("Failed to start container")
		return container.ContainerCreateCreatedBody{}, err
	}
	log.Debug("Started container")

	return containerCreateResponse, nil
}

func main( ) {
	log.SetLevel(log.DebugLevel)
	networkName := "bridge-net-test"

	imageName1 := "weevenetwork/mosquitto_broker"
	containerName1 := "mosquitto_broker"
	var EntryPointArgs1 []string

	imageName2 := "weevenetwork/mosquitto_sub"
	containerName2 := "mosquitto_sub"
	EntryPointArgs2 := []string{"-h mosquitto_broker", "-p 1883", "-t hello/hello"}

	imageName3 := "weevenetwork/mosquitto_pub"
	containerName3 := "mosquitto_pub"
	EntryPointArgs3 := []string{"-h mosquitto_broker", "-p 1883", "-t hello/hello"}

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

	// CREATE AND ATTACH CONTAINER 1 //////
	containerCreateResponse, err := startCreateContainer(imageName1, containerName1, EntryPointArgs1)

	// Attach to network
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)

	// CREATE AND ATTACH CONTAINER 2 //////

	containerCreateResponse, err = startCreateContainer(imageName2, containerName2, EntryPointArgs2)

	// Attach to network
	// var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)


	// CREATE AND ATTACH CONTAINER 3 //////
	containerCreateResponse, err = startCreateContainer(imageName3, containerName3, EntryPointArgs3)

	// Attach to network
	// var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)
}