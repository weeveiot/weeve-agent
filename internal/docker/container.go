// data_access
package docker

import (
	"fmt"
	"io"
	"os"

	"bytes"
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

func StartContainers() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		log.Error(err)
	}

	for _, container := range containers {
		fmt.Print("Startings container ", container.ID[:10], "... ", container.State)
		// if "State": "running"

		if container.State != "running" {

			if err := cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
				log.Error(err)
			}
		}
		fmt.Println("Success")
	}
	return true
}

func StartContainer(containerId string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	err = cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		return false
	}

	//TODO: Wait for containers
	// statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// log.Error(err)
	// 		return false
	// 	}
	// case <-statusCh:
	// }

	out, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error(err)
		return false
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return true
}

func ReadAllContainers() []types.Container {
	log.Debug("Docker_container -> ReadAllContainers")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers
}

func GetContainerLog(container string) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error("Env Error ", err)
	}

	options := types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "40",
	}

	logs, err := cli.ContainerLogs(context.Background(), container, options)
	if err != nil {
		log.Error("Log fetch Error ", err)
	}
	log.Debug("Logs ", logs)
	buf := new(bytes.Buffer)
	buf.ReadFrom(logs)
	logStr := buf.String()

	log.Debug("Log string ", logStr)

	return logStr
}

func StopContainers() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		log.Error(err)
	}

	for _, container := range containers {
		fmt.Print("Stopping container ", container.ID[:10], "... ", container.State)
		// if "State": "running"

		if container.State == "running" {
			if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
				log.Error(err)
			}
		}
		fmt.Println("Success")
	}
	return true
}

func StopContainer(containerId string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStop(ctx, containerId, nil); err != nil {
		log.Error(err)
	}

	return true
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
func StartCreateContainer(imageName string, containerName string, entryArgs []string) (container.ContainerCreateCreatedBody, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		panic(err)
	}

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
		NetworkMode:  "bridge",
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 100,
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}

	containerCreateResponse, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName)
	if err != nil {
		panic(err)
	}
	log.Debug("\tCreated container " + containerName)

	// Start container
	err = cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		panic("Failed to start container")
	}
	log.Debug("\tStarted container")

	return containerCreateResponse, nil
}

// StopAndRemoveContainer Stop and remove a container
func StopAndRemoveContainer(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStop(ctx, containerName, nil); err != nil {
		log.Printf("Unable to stop container %s: %s", containerName, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		log.Printf("Unable to remove container: %s", err)
		return err
	}

	return nil
}

// ContainerExists returns status of container existance as true or false
func ContainerExists(containerName string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}

	for _, container := range containers {
		// fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		findContainer := util.StringArrayContains(container.Names, "/"+containerName)
		if findContainer {
			return true
		}
	}

	return false
}

func CreateContainer(containerName string, imageName string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		// log.Error(err)
		return false
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"echo", "Container " + containerName + " created"},
	}, &container.HostConfig{}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		log.Error(err)
		// return "CreateFailed"
		return false
	}

	if !StartContainer(resp.ID) {
		return false
	}

	return true
}

func CreateContainer1(containerName string, imageName string) string {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/"+imageName, types.ImagePullOptions{})
	if err != nil {
		log.Error(err)
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"echo", "Container " + containerName + " created"},
	}, &container.HostConfig{}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return "Container " + containerName + " created for image " + imageName
}

/*
func CreateContainerOptsArgs(startCmd model.ContainerConfig, networkName string) bool {

	// fmt.Println(startCmd)
	spew.Dump(startCmd)

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		// log.Error(err)
		return false
	}

	containerConfig := &container.Config{
		Image:        startCmd.ImageName + ":" + startCmd.ImageTag,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          startCmd.EntryPointArgs,
		Tty:          false,
		ExposedPorts: startCmd.ExposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings: startCmd.PortBinding,
		NetworkMode:  startCmd.NetworkMode,
	}

	resp, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		&startCmd.NetworkConfig,
		// &network.NetworkingConfig{},
		nil,
		startCmd.ContainerName)
	// fmt.Println(resp)
	if err != nil {
		log.Error(err)
		// return "CreateFailed"
		return false
	}
	log.Debug("Created container " + startCmd.ContainerName)

	// containerStarted := StartContainer(resp.ID)

	// if !containerStarted {
	// 	log.Debug("Did not start container")
	// 	return false
	// }

	// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// case <-statusCh:
	// }

	// out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	// if err != nil {
	// 	panic(err)
	// }

	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	// cli.NetworkConnect(ctx, "TEST", resp.ID, config *network.EndpointSettings)
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, resp.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)

	return true
}
*/
