// data_access
package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"

	log "github.com/sirupsen/logrus"
)

// PullImages iterates modules and pulls images
// func PullImagesNew(manifest model.ManifestReq) bool {
func PullImagesNew(imageNames []string) bool {
	for i := range imageNames {

		// Check if image exist in local
		exists := ImageExists(imageNames[i])
		log.Debug(fmt.Sprintf("\tImage %v %v, %v", i, imageNames[i], exists))
		// log.Debug("\tImage exists: ", exists)

		if exists == false {
			// Pull image if not exist in local
			log.Debug("\t\tPulling ", imageNames[i])
			exists = PullImage(imageNames[i])
			if exists == false {
				return false
			}
		}
	}

	return true
}

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
	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
	return images
}



// Management API

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

// // ReadImage by ImageId
// func ReadImage(id string) types.ImageInspect {
// 	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
// 	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	images, bytes, err := cli.ImageInspectWithRaw(ctx, id)
// 	if err != nil && bytes != nil {
// 		panic(err)
// 	}

// 	return images
// }

// SearchImages returns images based on filter (Currently working without filters)
func SearchImages(term string, id string, tag string) []registry.SearchResult {
	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	var searchFilter filters.Args
	// searchFilter.Add("Id", id)
	// searchFilter.Add("RepoTags", tag)
	// searchFilter.Add("Labels", label)
	options := types.ImageSearchOptions{Filters: searchFilter, Limit: 5}
	images, err := cli.ImageSearch(ctx, term, options)
	if err != nil {
		panic(err)
	}

	for k, image := range images {
		fmt.Println(k, image)
	}
	return images
}

func CreateImage(id string, dataMap map[string]string) io.ReadCloser {
	// For internal images created with our sources
	// https://github.com/moby/moby/blob/master/client/image_build.go#L20

	//	https://github.com/moby/moby/blob/master/client/image_create.go#L15

	// For external available images ex: ubuntu
	// https://github.com/moby/moby/blob/master/client/image_import.go#L15

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	imageName := "image1"

	// var imageBuildOptions types.ImageBuildResponse
	// imageBuildOptions = types.ImageBuildOptions{}
	// ImageBuildOptions{
	//		Tags           []string
	//    SuppressOutput bool
	//    RemoteContext  string
	//    NoCache        bool
	//    Remove         bool
	//    ForceRemove    bool
	//    PullParent     bool
	//    Isolation      container.Isolation
	//    CPUSetCPUs     string
	//    CPUSetMems     string
	//    CPUShares      int64
	//    CPUQuota       int64
	//    CPUPeriod      int64
	//    Memory         int64
	//    MemorySwap     int64
	//    CgroupParent   string
	//    NetworkMode    string
	//    ShmSize        int64
	//    Dockerfile     string
	//    Ulimits        []*units.Ulimit
	//    BuildArgs   map[string]*string
	//    AuthConfigs map[string]AuthConfig
	//    Context     io.Reader
	//    Labels      map[string]string
	//    Squash bool
	//    CacheFrom   []string
	//    SecurityOpt []string
	//    ExtraHosts  []string // List of extra hosts
	//    Target      string
	//    SessionID   string
	//    Platform    string
	//    Version BuilderVersion
	//    BuildID string
	//    Outputs []ImageBuildOutput
	// }
	// out, err := cli.ImageBuild(ctx, imageName, imageBuildOptions)
	// if err != nil {
	// 	panic(err)
	// }

	//		RegistryAuth :"",
	//		Platform     :""

	var imageCreateOptions types.ImageCreateOptions
	// imageCreateOptions = types.ImageCreateOptions{"", ""}

	out, err := cli.ImageCreate(ctx, imageName, imageCreateOptions)
	if err != nil {
		panic(err)
	}
	return out
}
