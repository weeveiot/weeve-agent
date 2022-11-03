//go:build !secunet

package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"

	log "github.com/sirupsen/logrus"
)

func PullImage(authConfig types.AuthConfig, imageName string) error {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	events, err := dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	d := json.NewDecoder(events)

	type Event struct {
		Status         string `json:"status"`
		Error          string `json:"error"`
		Progress       string `json:"progress"`
		ProgressDetail struct {
			Current int `json:"current"`
			Total   int `json:"total"`
		} `json:"progressDetail"`
	}

	var event *Event
	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, traceutility.FuncTrace())
		}
	}

	// Latest event for new image
	// EVENT: {Status:Status: Downloaded newer image for busybox:latest Error: Progress:[==================================================>]  699.2kB/699.2kB ProgressDetail:{Current:699243 Total:699243}}
	// Latest event for up-to-date image
	// EVENT: {Status:Status: Image is up to date for busybox:latest Error: Progress: ProgressDetail:{Current:0 Total:0}}
	if event != nil {
		if strings.Contains(event.Status, fmt.Sprintf("Downloaded newer image for %s", imageName)) {
			log.Info("Pulled image " + imageName + " into host")
		}
		if strings.Contains(event.Status, fmt.Sprintf("Image is up to date for %s", imageName)) {
			log.Info("Updated image " + imageName + " into host")
		}
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
			return false, errors.Wrap(err, traceutility.FuncTrace())
		}
	}

	return true, nil
}

func ImageRemove(imageID string) error {
	_, err := dockerClient.ImageRemove(ctx, imageID, types.ImageRemoveOptions{})
	if err != nil {
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	return nil
}

func GetImagesByName(images []string) ([]types.ImageSummary, error) {
	filter := filters.NewArgs()
	for _, image := range images {
		filter.Add("reference", image)
	}
	options := types.ImageListOptions{Filters: filter}

	return dockerClient.ImageList(ctx, options)
}
