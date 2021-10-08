package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

func DeployManifest(man model.Manifest, command string) bool {

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		jsonlines.Insert(constants.ManifestFile, "Invalid Manifest.")
		return false
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	jsonlines.Delete(constants.ManifestFile, "id", man.Manifest.Search("id").Data().(string))

	//******** STEP 1 - Deploy or Redeploy process *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	if command == "deploy" {
		// Check if data service already exist
		containerExists := DataServiceExist(manifestID, version)
		if containerExists {
			log.Info(fmt.Sprintf("\tData service %v already exist", man.Manifest.Search("name").Data().(string)))
			LogSuccess(man, "status")
			return true
		}
	} else if command == "redeploy" {
		// Clean old data service resources
		result := UndeployDataService(manifestID, version)
		if !result {
			log.Info("\tError while cleaning old data service - ", result)
			LogFailure(man, "could not redeploy")
			return false
		}
	}

	//******** STEP 2 - Pull all images *************//
	// Pull all images as required
	log.Info("Iterating modules, pulling image into host if missing")

	for i, imgDetails := range man.ImageNamesWithRegList() {
		// Check if image exist in local
		exists := docker.ImageExists(imgDetails.ImageName)
		if exists { // Image already exists, continue
			log.Info(fmt.Sprintf("\tImage %v %v, already exists on host", i, imgDetails.ImageName))
		} else { // Pull this image
			log.Info(fmt.Sprintf("\tImage %v %v, does not exist on host", i, imgDetails.ImageName))
			log.Info("\t\tPulling ", imgDetails.ImageName, imgDetails)
			exists = docker.PullImage(imgDetails)
			if !exists {
				failed = true
				msg := "404 - Unable to pull image " + imgDetails.ImageName
				log.Error(msg)
				LogFailure(man, msg)
				return false
			}
		}
	}

	if failed {
		LogFailure(man, "status")
		return false
	}

	//******** STEP 3 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		LogFailure(man, "Failed to create a client for docker.")
		return false
	}

	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = man.GetLabels()

	networkName := docker.GetNetworkName(man)
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		LogFailure(man, "Error trying to create network "+networkName)
		return false
	}
	log.Info("Created network named ", networkName)

	_ = resp
	// log.Info(resp.ID, resp.Warning)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info("Start all containers")
	var contianers_cmd = man.GetContainerStart(networkName)

	if contianers_cmd == nil || len(contianers_cmd) <= 0 {
		log.Error("No valid contianers in Manifest")
		LogFailure(man, "No valid contianers in Manifest")
		return false
	}
	for _, startCommand := range contianers_cmd {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag, startCommand)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand)
		if err != nil {
			failed = true
			log.Error("Failed to create and start container: " + imageAndTag)
			LogFailure(man, "Failed to create and start container: "+imageAndTag)
			return false
		}
		log.Info("\tSuccessfully created with args: ", startCommand.EntryPointArgs, containerCreateResponse)
		log.Info("Started")

		// // Attach to network
		// var netConfig network.EndpointSettings
		// err = cli.NetworkConnect(ctx, startCommand.NetworkName, containerCreateResponse.ID, &netConfig)
		// if err != nil {
		// 	panic(err)
		// }
		// log.Info("\tConnected to network", startCommand.NetworkName)
	}

	if failed {
		LogFailure(man, "status")
		return false
	}
	LogSuccess(man, "status")

	log.Info("Completed")

	// TODO: Proper return/error handling
	return true
}

func StopDataService(manifestID string, version string) bool {
	log.Info("Stopping data service:", manifestID, version)
	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		return true
	}
	for _, container := range containers {
		if container.State == "running" {
			log.Info("\tStopping container:", strings.Join(container.Names[:], ","))
			status := docker.StopContainer(container.ID)
			if !status {
				log.Error("\tCould not stop a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		}
	}
	return true
}

func StartDataService(manifestID string, version string) bool {
	log.Info("Starting data service:", manifestID, version)
	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		return false
	}
	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("\tStarting container:", strings.Join(container.Names[:], ","))
			status := docker.StartContainer(container.ID)
			if !status {
				log.Error("\tCould not start a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}
	return true
}

func UndeployDataService(manifestID string, version string) bool {
	log.Info("Undeploying ", manifestID, version)

	// Set up Background Context and Client for Docker API calls
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return false
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	imageContainers := make(map[string]int)

	containers := docker.ReadAllContainers()
	if containers == nil {
		log.Error("Failed to read all containers.")
		return false
	}
	dsContainers := docker.ReadDataServiceContainers(manifestID, version)
	if dsContainers == nil {
		log.Error("Failed to read data service containers.")
		return false
	}
	for _, container := range containers {

		imageContainers[container.ImageID] = imageContainers[container.ImageID] + 1

		for _, dsContainer := range dsContainers {
			if container.ID == dsContainer.ID {
				log.Info("\tStop And Remove Container - ", dsContainer.ID)
				// Stop and delete container
				err := docker.StopAndRemoveContainer(dsContainer.ID)
				if err != nil {
					log.Error(err)
					return false
				}

				imageContainers[container.ImageID] = imageContainers[container.ImageID] - 1
			}
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	for imageID, containersCount := range imageContainers {
		if containersCount == 0 {
			log.Info("\tRemove Image - ", imageID)
			_, err := cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{})
			if err != nil {
				log.Error(err)
				return false
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info("Pruning networks")
	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("Pruned networks:", pruneReport)

	return true
}

// DataServiceExist returns status of data service existance as true or false
func DataServiceExist(manifestID string, version string) bool {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := dockerClient.NetworkList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}

	if len(networks) > 0 {
		return true
	}

	return false
}

func LogFailure(man model.Manifest, message string) {
	man.Manifest.Set("FAILED", message)
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
}

func LogSuccess(man model.Manifest, message string) {
	man.Manifest.Set("SUCCESS", message)
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
}
