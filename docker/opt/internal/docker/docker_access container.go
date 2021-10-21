// data_access
package docker

import (
	"io"
	"os"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	log "github.com/sirupsen/logrus"
)

func PullImage(imageName string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return false
	}

	// os.Stdout,_ = os.Open(os.DevNull)

	//TODO: Need to disable Stdout!!
	log.Info("\t\tPulling image " + imageName)
	// _, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		log.Error(err)
		return false
	}
	defer out.Close()

	io.Copy(os.Stdout, out)

	log.Info("Pulled image " + imageName + " into host")

	return true
}

func CreateContainer(containerName string, imageName string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
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

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
		// return "StartFailed"
		return false
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
			return false
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error(err)
		return false
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	// return "Container " + containerName + " created for image " + imageName
	return true
}
