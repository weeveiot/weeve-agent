// data_access
package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	log "github.com/sirupsen/logrus"
)

func PullImage(imgDetails model.RegistryDetails) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return false
	}

	authConfig := types.AuthConfig{
		Username: imgDetails.UserName,
		Password: imgDetails.Password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Error(err)
		return false
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	events, err := cli.ImagePull(ctx, imgDetails.ImageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		log.Error(err)
		return false
	}
	// defer out.Close()

	// To write all
	//io.Copy(os.Stdout, out)

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
			return false
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

	return true
}

// Check if the image exists in the local context
// Return bool
func ImageExists(id string) bool {
	image := ReadImage(id)

	if image.ID != "" {
		return true
	} else {
		return false
	}
}

// To be listed for selection of images in management app
func ReadAllImages() []types.ImageSummary {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		log.Error(err)
		return nil
	}

	if false {
		for _, image := range images {
			fmt.Println(image.ID)
		}
	}

	return images
}

// ReadImage by ImageId
func ReadImage(id string) types.ImageInspect {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return types.ImageInspect{}
	}

	images, bytes, err := cli.ImageInspectWithRaw(ctx, id)
	if err != nil && bytes != nil {
		log.Error(err)
		return types.ImageInspect{}
	}

	return images
}

// SearchImages returns images based on filter (Currently working without filters)
func SearchImages(term string, id string, tag string) []registry.SearchResult {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}

	var searchFilter filters.Args
	options := types.ImageSearchOptions{Filters: searchFilter, Limit: 5}
	images, err := cli.ImageSearch(ctx, term, options)
	if err != nil {
		log.Error(err)
		return nil
	}

	for k, image := range images {
		fmt.Println(k, image)
	}
	return images
}
