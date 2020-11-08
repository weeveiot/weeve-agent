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

	///////////////////////////////////////
	// CREATE AND ATTACH CONTAINER 1 //////
	///////////////////////////////////////
	// Config
	containerConfig := &container.Config{
		Image:        imageName1,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          EntryPointArgs1,
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

	// Start container
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		panic("Failed to start container")
	}
	log.Debug("Started container")

	// Attach to network
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)



	///////////////////////////////////////
	// CREATE AND ATTACH CONTAINER 2 //////
	///////////////////////////////////////
	// Config
	containerConfig = &container.Config{
		Image:        imageName2,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          EntryPointArgs2,
		Tty:          false,
		ExposedPorts: nil,
	}

	hostConfig = &container.HostConfig{
		PortBindings: nil,
		NetworkMode: "bridge",
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
			MaximumRetryCount: 100,
		},
	}

	networkConfig = &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	// Create it
	containerCreateResponse, err = cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName2)
	if err != nil {
		panic(err)
	}
	log.Debug("Created container " + containerName1)

	// Start container
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		panic("Failed to start container")
	}
	log.Debug("Started container 2")

	// Attach to network
	// var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)



	///////////////////////////////////////
	// CREATE AND ATTACH CONTAINER 3 //////
	///////////////////////////////////////
	// Config
	containerConfig = &container.Config{
		Image:        imageName3,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          EntryPointArgs3,
		Tty:          false,
		ExposedPorts: nil,
	}

	hostConfig = &container.HostConfig{
		PortBindings: nil,
		NetworkMode: "bridge",
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
			MaximumRetryCount: 100,
		},
	}

	networkConfig = &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	// Create it
	containerCreateResponse, err = cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName3)
	if err != nil {
		panic(err)
	}
	log.Debug("Created container " + containerName1)

	// Start container
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		panic("Failed to start container")
	}
	log.Debug("Started container 3")

	// Attach to network
	// var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, containerCreateResponse.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)
}