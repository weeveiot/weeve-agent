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

func DeployManifest(man model.Manifest) string {

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return "FAILED"
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	jsonlines.Delete(constants.ManifestFile, "id", man.Manifest.Search("id").Data().(string))

	//******** STEP 1 - Pull all *************//
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

	//******** STEP 2 - Check containers, stop and remove *************//
	log.Info("Checking containers, stopping and removing")

	for _, containerName := range man.ContainerNamesList() {

		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Stop + remove container if exists, start fresh
		if containerExists {
			log.Info("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				failed = true
				log.Error(err)
				return "FAILED"
			}
			log.Info("\tContainer ", containerName, " removed")
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

	log.Info("Pruning networks")
	filter := filters.NewArgs()

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	if err != nil {
		log.Error(err)
		log.Error("Error trying to prune network")
		panic(err)

	}
	log.Info("Pruned:", pruneReport)
	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true

	networkName := man.GetNetworkName()
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
	var contianers_cmd = man.GetContainerStart()

	if contianers_cmd == nil || len(contianers_cmd) <= 0 {
		log.Error("No valid contianers in Manifest")
		man.Manifest.Set("FAILED", "No valid contianers in Manifest")
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

		// // Attach to network
		// var netConfig network.EndpointSettings
		// err = cli.NetworkConnect(ctx, startCommand.NetworkName, containerCreateResponse.ID, &netConfig)
		// if err != nil {
		// 	panic(err)
		// }
		// log.Info("\tConnected to network", startCommand.NetworkName)
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}
	man.Manifest.Set("SUCCESS", "status")
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

	log.Info("Started")

	// TODO: Proper return/error handling
	return "SUCCESS"
}

func StopDataService(serviceId string, dataservice_name string) {
	log.Info("Stopping data service:", dataservice_name)
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			// Container's names are in form: "/container_name"
			if strings.HasPrefix(name[1:], serviceId) {
				log.Info("\tStopping container:", name)
				docker.StopContainer(container.ID)
			}
		}
	}
}

func StartDataService(serviceId string, dataservice_name string) {
	log.Info("Starting data service:", dataservice_name)
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			// Container's names are in form: "/container_name"
			if strings.HasPrefix(name[1:], serviceId) {
				log.Info("\tStarting container:", name)
				docker.StartContainer(container.ID)
			}
		}
	}
}

func UnDeployManifest(man model.Manifest) string {

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return "FAILED"
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	jsonlines.Delete(constants.ManifestFile, "id", man.Manifest.Search("id").Data().(string))

	//******** STEP 1 - Check containers, stop and remove *************//
	log.Info("Checking containers, stopping and removing")

	for _, containerName := range man.ContainerNamesList() {

		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Stop + remove container if exists, start fresh
		if containerExists {
			log.Info("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				failed = true
				log.Error(err)
				return "FAILED"
			}
			log.Info("\tContainer ", containerName, " removed")
		}
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return "FAILED"
	}

	//******** STEP 2 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	log.Info("Pruning networks")
	filter := filters.NewArgs()

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	log.Info("Pruned:", pruneReport)

	networkName := man.GetNetworkName()

	if err != nil {
		log.Error(err)
		log.Error("Error trying to create network " + networkName)
		panic(err)

	}
	log.Info("Removed network ", networkName)

	// TODO: Proper return/error handling
	return "SUCCESS"
}
