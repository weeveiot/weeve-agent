//go:build !secunet

package docker

import (
	"context"
	"encoding/binary"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var ctx = context.Background()
var dockerClient *client.Client

type ContainerLog struct {
	ContainerID string `json:"containerID"`
	Log         []Log  `json:"log"`
}

type Log struct {
	Time   string `json:"time"`
	Stream string `json:"stream"`
	Log    string `json:"log"`
}

func SetupDockerClient() {
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
}

func createContainer(containerConfig manifest.ContainerConfig) (string, error) {
	imageName := containerConfig.ImageName + ":" + containerConfig.ImageTag

	log.Debugln("Creating container", containerConfig.ContainerName, "from", imageName)

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
		return containerCreateResponse.ID, err
	}
	log.Debug("Created container " + containerConfig.ContainerName)

	return containerCreateResponse.ID, nil
}

func StartContainer(containerID string) error {
	err := dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	log.Debug("Started container ID ", containerID)

	return nil
}

func CreateAndStartContainer(containerConfig manifest.ContainerConfig) (string, error) {
	id, err := createContainer(containerConfig)
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
		return err
	}

	return nil
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
		return err
	}

	return nil
}

func ReadAllContainers() ([]types.Container, error) {
	log.Debug("Docker_container -> ReadAllContainers")
	options := types.ContainerListOptions{All: true}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers, nil
}

func ReadDataServiceContainers(manifestUniqueID model.ManifestUniqueID) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manifestUniqueID.ManifestName)
	filter.Add("label", "versionNumber="+manifestUniqueID.VersionNumber)
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func ReadContainerLogs(containerID string, since string, until string) (ContainerLog, error) {
	dockerLogs := ContainerLog{ContainerID: containerID}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since,
		Until:      until,
		Timestamps: true,
		Follow:     false,
		Tail:       "",
		Details:    false,
	}

	reader, err := dockerClient.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		return dockerLogs, err
	}
	defer reader.Close()

	header := make([]byte, 8)
	for {
		var docLog Log
		_, err := reader.Read(header)
		if err != nil {
			if err == io.EOF {
				return dockerLogs, nil
			}
			return dockerLogs, err
		}

		count := binary.BigEndian.Uint32(header[4:])
		data := make([]byte, count)
		_, err = reader.Read(data)
		if err != nil {
			if err == io.EOF {
				return dockerLogs, nil
			}
			return dockerLogs, err
		}

		time, log, found := strings.Cut(string(data), " ")
		if found {
			docLog.Time = time
			docLog.Log = log
			switch header[0] {
			case 1:
				docLog.Stream = "Stdout"
			default:
				docLog.Stream = "Stderr"
			}

			dockerLogs.Log = append(dockerLogs.Log, docLog)
		}
	}
}

func InspectContainer(containerID string) (types.ContainerJSON, error) {
	containerJSON, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return containerJSON, nil
}
