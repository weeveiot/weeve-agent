package docker

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

var ctx = context.Background()
var dockerClient *client.Client

func SetupDockerClient() {
	log.Debug("Initalizing docker client...")

	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Docker client creation failed! CAUSE --> ", err)
	}
}

func createContainer(containerConfig manifest.ContainerConfig) (string, error) {
	log.Debugln("Creating container", containerConfig.ContainerName, "from", containerConfig.ImageNameFull)

	config := &container.Config{
		Image:        containerConfig.ImageNameFull,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Env:          containerConfig.EnvArgs,
		Tty:          false,
		ExposedPorts: containerConfig.ExposedPorts,
		Labels:       containerConfig.Labels,
	}

	hostConfig := &container.HostConfig{
		LogConfig: container.LogConfig{
			Type: "local", // From https://docs.docker.com/config/containers/logging/local/: By default, the local driver preserves 100MB of log messages per container and uses automatic compression to reduce the size on disk. The 100MB default value is based on a 20M default size for each file and a default count of 5 for the number of such files (to account for log rotation).
		},
		PortBindings: containerConfig.PortBinding,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 100,
		},
		Mounts:    containerConfig.MountConfigs,
		Resources: containerConfig.Resources,
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
		return containerCreateResponse.ID, traceutility.Wrap(err)
	}
	log.Debug("Created container " + containerConfig.ContainerName)

	return containerCreateResponse.ID, nil
}

func StartContainer(containerID string) error {
	err := dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return traceutility.Wrap(err)
	}
	log.Debug("Started container ID ", containerID)

	return nil
}

func CreateAndStartContainer(containerConfig manifest.ContainerConfig) (string, error) {
	id, err := createContainer(containerConfig)
	if err != nil {
		return id, traceutility.Wrap(err)
	}

	err = StartContainer(id)
	if err != nil {
		return id, traceutility.Wrap(err)
	}

	return id, nil
}

func StopContainer(containerID string) error {
	stopTimeout := 2
	return dockerClient.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &stopTimeout})
}

func StopAndRemoveContainer(containerID string) error {
	if err := StopContainer(containerID); err != nil {
		log.Errorf("Unable to stop container %s: %s. Will try to force remove...", containerID, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := dockerClient.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		log.Errorf("Unable to remove container: %s", err)
		return traceutility.Wrap(err)
	}

	return nil
}

func ReadAllContainers() ([]types.Container, error) {
	log.Debug("Docker_container -> ReadAllContainers")
	options := types.ContainerListOptions{All: true}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}

func ReadEdgeAppContainers(manifestUniqueID model.ManifestUniqueID) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestUniqueID="+manifestUniqueID.String())
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	return containers, nil
}

func ReadContainerLogs(containerID string, since string, until string) ([]string, error) {
	logLines := []string{}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since,
		Until:      until,
	}

	reader, err := dockerClient.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		return logLines, traceutility.Wrap(err)
	}
	defer reader.Close()

	header := make([]byte, 8)
	for {
		_, err := reader.Read(header)
		if err != nil {
			if err == io.EOF {
				return logLines, nil
			}
			return logLines, traceutility.Wrap(err)
		}

		count := binary.BigEndian.Uint32(header[4:])
		line := make([]byte, count)
		_, err = reader.Read(line)
		if err != nil {
			if err == io.EOF {
				return logLines, nil
			}
			return logLines, traceutility.Wrap(err)
		}

		logLines = append(logLines, string(line))
	}
}

func InspectContainer(containerID string) (types.ContainerJSON, error) {
	containerJSON, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return types.ContainerJSON{}, traceutility.Wrap(err)
	}

	return containerJSON, nil
}
