//go:build !secunet

package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/weeveiot/weeve-agent/internal/model"

	log "github.com/sirupsen/logrus"
)

func PullImage(imgDetails model.RegistryDetails) error {
	authConfig := types.AuthConfig{
		Username: imgDetails.UserName,
		Password: imgDetails.Password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Error(err)
		return err
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	events, err := dockerClient.ImagePull(ctx, imgDetails.ImageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		log.Error(err)
		return err
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
			log.Error(err)
			return err
		}
	}

	// Latest event for new image
	// EVENT: {Status:Status: Downloaded newer image for busybox:latest Error: Progress:[==================================================>]  699.2kB/699.2kB ProgressDetail:{Current:699243 Total:699243}}
	// Latest event for up-to-date image
	// EVENT: {Status:Status: Image is up to date for busybox:latest Error: Progress: ProgressDetail:{Current:0 Total:0}}
	if event != nil {
		if strings.Contains(event.Status, fmt.Sprintf("Downloaded newer image for %s", imgDetails.ImageName)) {
			log.Info("Pulled image " + imgDetails.ImageName + " into host")
		}
		if strings.Contains(event.Status, fmt.Sprintf("Image is up to date for %s", imgDetails.ImageName)) {
			log.Info("Updated image " + imgDetails.ImageName + " into host")
		}
	}

	return nil
}

// Check if the image exists in the local context
// Return an error only if something went wrong, if the image is not found the error is nil
func ImageExists(id string) (bool, error) {
	_, _, err := dockerClient.ImageInspectWithRaw(ctx, id)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func ImageRemove(imageID string) error {
	_, err := dockerClient.ImageRemove(ctx, imageID, types.ImageRemoveOptions{})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
