package docker

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"

	log "github.com/sirupsen/logrus"
)

func PullImage(authConfig types.AuthConfig, imageName string) error {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return traceutility.Wrap(err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	events, err := dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer events.Close()

	d := json.NewDecoder(events)

	var event *jsonmessage.JSONMessage
	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return traceutility.Wrap(err)
		}
		log.Debugln(event.Status, event.Progress)
	}

	if event != nil {
		log.Info(event.Status)
	}

	return nil
}

// Check if the image exists in the local context
// Return an error only if something went wrong, if the image is not found the error is nil
func ImageExists(imageName string) (bool, error) {
	_, _, err := dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		} else {
			return false, traceutility.Wrap(err)
		}
	}

	return true, nil
}

func ImageRemove(imageID string) error {
	_, err := dockerClient.ImageRemove(ctx, imageID, types.ImageRemoveOptions{})
	if err != nil {
		return traceutility.Wrap(err)
	}

	return nil
}

func GetImagesByName(images []string) ([]types.ImageSummary, error) {
	if len(images) == 0 {
		return nil, nil
	}

	filter := filters.NewArgs()
	for _, image := range images {
		filter.Add("reference", image)
	}
	options := types.ImageListOptions{Filters: filter}

	return dockerClient.ImageList(ctx, options)
}
