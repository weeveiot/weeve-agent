// data_access
package docker

import (
	"os"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func StartContainer(containerId string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return false
	}

	err = cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		return false
	}

	out, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error(err)
		return false
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return true
}

func ReadAllContainers() ([]types.Container, error) {
	log.Debug("Docker_container -> ReadAllContainers")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil, err
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
		return nil, nil
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}

func ReadDataServiceContainers(manifestID string, version string) ([]types.Container, error) {
	log.Debug("Docker_container -> ReadDataServiceContainers")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil, err
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
		return nil, nil
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}

func StopContainer(containerId string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return err
	}

	if err := cli.ContainerStop(ctx, containerId, nil); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// StartCreateContainer is a utility function based on the Docker SDK
// The flow of logic;
// 1) Instantiate the docker client object
// 2) Configure the container with imageName and entryArgs
// 3) Configure the host with the network configuration and restart policy
// 4) Configure the network with endpoints
// 5) Create the container with the above 3 configurations, and the container name
// 6) Start the container
// 7) Return containerStart response
func StartCreateContainer(imageName string, startCommand model.ContainerConfig) (container.ContainerCreateCreatedBody, error) {
	var containerName = startCommand.ContainerName

	log.Debug("Creating container "+containerName, "from "+imageName)
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return container.ContainerCreateCreatedBody{}, err
	}

	containerConfig := &container.Config{
		Image:        imageName,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          startCommand.EntryPointArgs,
		Env:          startCommand.EnvArgs,
		Tty:          false,
		ExposedPorts: startCommand.ExposedPorts,
		Labels:       startCommand.Labels,
	}

	hostConfig := &container.HostConfig{
		PortBindings: startCommand.PortBinding,
		NetworkMode:  container.NetworkMode(startCommand.NetworkDriver),
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 100,
		},
		Mounts: startCommand.MountConfigs,
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			startCommand.NetworkName: {},
		},
	}

	containerCreateResponse, err := dockerClient.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName)
	if err != nil {
		log.Error(err)
		return containerCreateResponse, err
	}
	log.Debug("Created container " + containerName)

	// Start container
	err = dockerClient.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		return containerCreateResponse, err
	}
	log.Debug("Started container!")

	return containerCreateResponse, nil
}

// StopAndRemoveContainer Stop and remove a container
func StopAndRemoveContainer(containerID string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return err
	}

	if err := cli.ContainerStop(ctx, containerID, nil); err != nil {
		log.Printf("Unable to stop container %s: %s", containerID, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		log.Printf("Unable to remove container: %s", err)
		return err
	}

	return nil
}
