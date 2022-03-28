package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var ctx = context.Background()
var dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

func init() {
	if err != nil {
		log.Error(err)
		panic(err)
	}
}

func CreateContainer(containerConfig model.ContainerConfig) (string, error) {
	imageName := containerConfig.ImageName + ":" + containerConfig.ImageTag

	log.Debug("Creating container", containerConfig.ContainerName, "from", imageName)

	config := &container.Config{
		Image:        imageName,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          containerConfig.EntryPointArgs,
		Env:          containerConfig.EnvArgs,
		Tty:          false,
		ExposedPorts: containerConfig.ExposedPorts,
		Labels:       containerConfig.Labels,
	}

	hostConfig := &container.HostConfig{
		PortBindings: containerConfig.PortBinding,
		NetworkMode:  container.NetworkMode(containerConfig.NetworkDriver),
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 100,
		},
		Mounts: containerConfig.MountConfigs,
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			containerConfig.NetworkName: {},
		},
	}

	containerCreateResponse, err := dockerClient.ContainerCreate(ctx,
		config,
		hostConfig,
		networkConfig,
		nil,
		containerConfig.ContainerName)
	if err != nil {
		log.Error(err)
		return containerCreateResponse.ID, err
	}
	log.Debug("Created container " + containerConfig.ContainerName)

	return containerCreateResponse.ID, nil
}

func StartContainer(containerId string) error {
	err = dockerClient.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	log.Debug("Started container ID ", containerId)

	return nil
}

func CreateAndStartContainer(containerConfig model.ContainerConfig) (string, error) {
	id, err := CreateContainer(containerConfig)
	if err != nil {
		return id, err
	}

	err = StartContainer(id)
	if err != nil {
		return id, err
	}

	return id, nil
}

func StopContainer(containerID string) error {
	if err := dockerClient.ContainerStop(ctx, containerID, nil); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func StopAndRemoveContainer(containerID string) error {
	if err := StopContainer(containerID); err != nil {
		log.Errorf("Unable to stop container %s: %s\nWill try to force remove...", containerID, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := dockerClient.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		log.Errorf("Unable to remove container: %s", err)
		return err
	}

	return nil
}

func ReadAllContainers() ([]types.Container, error) {
	log.Debug("Docker_container -> ReadAllContainers")
	options := types.ContainerListOptions{All: true}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}

func ReadDataServiceContainers(manifestID string, version string) ([]types.Container, error) {
	log.Debug("Docker_container -> ReadDataServiceContainers")

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}
