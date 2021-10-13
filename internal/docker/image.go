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

// PullImages iterates modules and pulls images
// func PullImagesNew(manifest model.ManifestReq) bool {
// func PullImagesNew(imageNames []string) bool {
// 	for i := range imageNames {
// 		log.Debug("IMAGE NAMES:", imageNames)
// 		// Check if image exist in local
// 		exists := ImageExists(imageNames[i])
// 		if exists {
// 			log.Debug(fmt.Sprintf("\tImage %v %v, already exists on host", i, imageNames[i]))
// 		} else {
// 			log.Debug(fmt.Sprintf("\tImage %v %v, does not exist on host", i, imageNames[i]))
// 		}

// 		if exists == false {
// 			// Pull image if not exist in local
// 			log.Debug("\t\tPulling ", imageNames[i])
// 			exists = PullImage(imageNames[i])
// 			if exists == false {
// 				return false
// 			}
// 		}
// 	}

// 	return true
// }

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

	// Imager pull returns a Reader object, see:
	// https://stackoverflow.com/questions/44452679/golang-docker-api-parse-result-of-imagepull
	// log.Info("\t\tPulling image " + imageName)

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
		// fmt.Printf(".")
		// fmt.Printf("EVENT: %+v\n", event)
		// fmt.Printf(".\n")
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
	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

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

// Management API

// ReadImage by ImageId
func ReadImage(id string) types.ImageInspect {
	// https://docs.docker.com/engine/api/sdk/examples/#list-all-images
	// https://github.com/moby/moby/blob/master/client/image_list.go#L14

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
		log.Error(err)
		return nil
	}

	var searchFilter filters.Args
	// searchFilter.Add("Id", id)
	// searchFilter.Add("RepoTags", tag)
	// searchFilter.Add("Labels", label)
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

func CreateImage(id string, dataMap map[string]string) io.ReadCloser {
	panic("TODO - Refactor this")
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
