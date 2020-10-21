// data_access
package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

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

// ReadImage by ImageId
func ReadImage(id string) types.ImageInspect {
	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	images, bytes, err := cli.ImageInspectWithRaw(ctx, id)
	if err != nil && bytes != nil {
		panic(err)
	}

	return images
}
