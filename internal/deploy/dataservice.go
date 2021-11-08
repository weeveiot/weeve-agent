package deploy

import (
	"context"
	"encoding/json"
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

func DeployManifest(man model.Manifest, command string) (response bool) {
	// flag for exception handling
	response = true
	defer func() {
		// if response true means there was exception because last line of this method (response = false) was not executed
		if response {
			response = false
		} else {
			response = true
		}
	}()

	log.Info(fmt.Sprintf("Data service %v initiated", command))

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return false
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	//******** STEP 1 - Deploy or Redeploy process *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	manifestName := man.Manifest.Search("name").Data().(string)

	dataServiceExists := DataServiceExist(manifestID, version)
	// Check if data service already exist
	if dataServiceExists {
		if command == "deploy" {
			log.Info(fmt.Sprintf("\tData service %v, %v already exist", manifestID, version))
			return false
		} else if command == "redeploy" {
			// Clean old data service resources
			result := UndeployDataService(manifestID, version)
			if !result {
				log.Error("\tError while cleaning old data service - ", result)
				LogStatus(manifestID, version, "REDEPLOY_FAILED", "Undeployment failed")
				return false
			}
		}
	}

	filter := map[string]string{"id": man.Manifest.Search("id").Data().(string), "version": man.Manifest.Search("version").Data().(string)}
	jsonlines.Delete(constants.ManifestFile, "", "", filter, true)

	// need to set some default manifest in manifest.jsonl so later could log without errors
	man.Manifest.Set("DEPLOYING_IN_PROGRESS", "status")
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

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
				msg := "404 - Unable to pull image/s, one or more image/s not found"
				log.Error(msg)
				LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", msg)
				return false
			}
		}
	}

	if failed {
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Process failed")
		return false
	}

	//******** STEP 3 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return false
	}

	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = man.GetLabels()

	networkName := docker.GetNetworkName(manifestName)
	if networkName == "" {
		log.Error("Failed to generate Network Name")
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Failed to generate Network Name")
		return false
	}

	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
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
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return false
	}
	for _, startCommand := range contianers_cmd {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag, startCommand)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand)
		if err != nil {
			failed = true
			log.Error("Failed to create and start container: " + imageAndTag)
			LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
			return false
		}
		log.Info("\tSuccessfully created with args: ", startCommand.EntryPointArgs, containerCreateResponse)
		log.Info("Started")
	}

	if failed {
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Process failed")
		return false
	}
	LogStatus(manifestID, version, strings.ToUpper(command)+"ED", strings.Title(command)+"ed successfully")

	response = false

	return response
}

func StopDataService(manifestID string, version string) (response bool) {
	// flag for exception handling
	response = true
	defer func() {
		// if response true means there was exception because last line of this method (response = false) was not executed
		if response {
			response = false
		} else {
			response = true
		}
	}()

	log.Info("Stopping data service:", manifestID, version)

	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		LogStatus(manifestID, version, "STOPPED", "Stopped successfully")
		return true
	}
	for _, container := range containers {
		if container.State == "running" {
			log.Info("\tStopping container:", strings.Join(container.Names[:], ","))
			status := docker.StopContainer(container.ID)
			if !status {
				log.Error("\tCould not stop a container")
				LogStatus(manifestID, version, "STOP_FAILED", "Could not stop a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		}
	}

	LogStatus(manifestID, version, "STOPPED", "Stopped successfully")

	response = false

	return response
}

func StartDataService(manifestID string, version string) (response bool) {
	// flag for exception handling
	response = true
	defer func() {
		// if response true means there was exception because last line of this method (response = false) was not executed
		if response {
			response = false
		} else {
			response = true
		}
	}()

	log.Info("Starting data service:", manifestID, version)

	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		LogStatus(manifestID, version, "START_FAILED", "No data service containers found")
		return false
	}
	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("\tStarting container:", strings.Join(container.Names[:], ","))
			status := docker.StartContainer(container.ID)
			if !status {
				log.Error("\tCould not start a container")
				LogStatus(manifestID, version, "START_FAILED", "Could not start a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	LogStatus(manifestID, version, "STARTED", "Started successfully")

	response = false

	return response
}

func UndeployDataService(manifestID string, version string) (response bool) {
	// flag for exception handling
	response = true
	defer func() {
		// if response true means there was exception because last line of this method (response = false) was not executed
		if response {
			response = false
		} else {
			response = true
		}
	}()

	log.Info("Undeploying data service", manifestID, version)

	// Check if data service already exist
	dataServiceExists := DataServiceExist(manifestID, version)
	if !dataServiceExists {
		log.Error(fmt.Sprintf("\tData service %v, %v does not exist", manifestID, version))
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", fmt.Sprintf("\tData service %v, %v does not exist", manifestID, version))
		return false
	}

	// Set up Background Context and Client for Docker API calls
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		return false
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	imageContainers := make(map[string]int)

	containers := docker.ReadAllContainers()
	if containers == nil {
		log.Error("Failed to read all containers.")
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read all containers")
		return false
	}
	dsContainers := docker.ReadDataServiceContainers(manifestID, version)
	if dsContainers == nil {
		log.Error("Failed to read data service containers.")
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read data service containers")
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
					LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
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
				LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
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
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		return false
	}
	log.Info("Pruned networks:", pruneReport)

	LogStatus(manifestID, version, "UNDEPLOYED", "Undeployed successfully")

	// Remove records from manifest.jsonl
	filterLog := map[string]string{"id": manifestID, "version": version}
	deleted := jsonlines.Delete(constants.ManifestFile, "", "", filterLog, true)
	if !deleted {
		log.Error("Could not remove old records from manifest.jsonl")
	}

	response = false

	return response
}

// DataServiceExist returns status of data service existance as true or false
func DataServiceExist(manifestID string, version string) (response bool) {
	// flag for exception handling
	response = true
	var networks []types.NetworkResource
	defer func() {
		// if response true means there was exception because last line of this method (response = false) was not executed
		if response {
			response = false
		} else {
			response = (len(networks) > 0)
		}
	}()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err = dockerClient.NetworkList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}

	response = false

	return response
}

func LogStatus(manifestID string, manifestVersion string, statusVal string, statusReason string) {
	filter := map[string]string{"id": manifestID, "version": manifestVersion}
	mani := jsonlines.Read(constants.ManifestFile, "", "", filter, false)
	deleted := jsonlines.Delete(constants.ManifestFile, "", "", filter, true)
	if !deleted {
		log.Error("Could not remove old records from manifest.jsonl")
	}

	if len(mani) != 0 {
		mani[0]["status"] = statusVal
		mani[0]["reason"] = statusReason
		maniJSON, err := json.Marshal(mani[0])
		if err != nil {
			log.Error("Could not convert map to json when logging status of the manifest")
		}
		inserted := jsonlines.Insert(constants.ManifestFile, string(maniJSON))
		if !inserted {
			log.Error("Could not log manifest status")
		}
	}
}
