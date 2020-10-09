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
)

func ReadAllContainers() []types.Container {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}

	return containers
}

func GetContainerLog(container string) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
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
		panic(err)
	}
	fmt.Print(logs)
	buf := new(bytes.Buffer)
	buf.ReadFrom(logs)
	logStr := buf.String()

	fmt.Printf("%s", logStr)

	return logStr
}

func StopContainers() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Print("Stopping container ", container.ID[:10], "... ", container.State)
		// if "State": "running"

		if container.State == "running" {
			if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
				panic(err)
			}
		}
		fmt.Println("Success")
	}
	return true
}

func StartContainers() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Print("Startings container ", container.ID[:10], "... ", container.State)
		// if "State": "running"

		if container.State != "running" {

			if err := cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
				panic(err)
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
		panic(err)
	}

	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	return true
}

func StopContainer(containerId string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStop(ctx, containerId, nil); err != nil {
		panic(err)
	}

	return true
}

func CreateContainer(containerName string, imageName string) string {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/"+imageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"echo", "Container " + containerName + " created"},
	}, &container.HostConfig{}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return "Container " + containerName + " created for image " + imageName
}
