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

func DeployManifest(man model.Manifest, command string) string {

	log.Info("Data service deployment initiated, ")

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return "FAILED"
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	jsonlines.Delete(constants.ManifestFile, "id", man.Manifest.Search("id").Data().(string))

	//******** STEP 1 - Deploy or Redeploy process *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	manifestName := man.Manifest.Search("name").Data().(string)
	if command == "deploy" {
		// Check if data service already exist
		containerExists := DataServiceExist(manifestID, version)
		if containerExists {
			log.Error(fmt.Sprintf("\tData service %v, %v already exist", manifestID, version))
			return "Data service already exist"
		}
	} else if command == "redeploy" {
		// Clean old data service resources
		result := UndeployDataService(manifestID, version)
		if !result {
			log.Error("\tError while cleaning old data service - ", result)
			return "FAILED"
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
				return "FAILED"
			}
		}
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}

	//******** STEP 3 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = man.GetLabels()

	networkName := docker.GetNetworkName(manifestName)
	if networkName == "" {
		log.Error("Failed to generate Network Name")
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}

	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		log.Error("Error trying to create network " + networkName)
		panic(err)

	}
	log.Info("Created network named ", networkName)

	_ = resp
	// log.Info(resp.ID, resp.Warning)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info("Start all containers")
	var contianers_cmd = man.GetContainerStart(networkName)

	if contianers_cmd == nil || len(contianers_cmd) <= 0 {
		log.Error("No valid contianers in Manifest")
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}
	for _, startCommand := range contianers_cmd {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag, startCommand)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand)
		log.Info("\tSuccessfully created with args: ", startCommand.EntryPointArgs, containerCreateResponse)
		if err != nil {
			failed = true
			log.Info("Started")
			return "FAILED"
		}
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}
	man.Manifest.Set("SUCCESS", "status")
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

	log.Info("Data service deployed")

	// TODO: Proper return/error handling
	return "SUCCESS"
}

func StopDataService(manifestID string, version string) {

	log.Info("Stopping data service:", manifestID, version)

	containers := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State == "running" {
			log.Info("\tStopping container:", strings.Join(container.Names[:], ","))
			docker.StopContainer(container.ID)
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		}
	}

	log.Info("Data service stopped")
}

func StartDataService(manifestID string, version string) {

	log.Info("Starting data service:", manifestID, version)

	containers := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("\tStarting container:", strings.Join(container.Names[:], ","))
			docker.StartContainer(container.ID)
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	log.Info("Data service started")
}

func UndeployDataService(manifestID string, version string) bool {

	log.Info("Undeploying data service", manifestID, version)

	// Check if data service already exist
	containerExists := DataServiceExist(manifestID, version)
	if !containerExists {
		log.Error(fmt.Sprintf("\tData service %v, %v does not exist", manifestID, version))
		return false
	}

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
	dsContainers := docker.ReadDataServiceContainers(manifestID, version)
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

	log.Info("Undeployed data service")

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

	return len(networks) > 0
}
