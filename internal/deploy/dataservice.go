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

	// need to set some default manifest in manifest.jsonl so later could log without errors
	man.Manifest.Set("DEPLOYING_IN_PROGRESS", "status")
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

	//******** STEP 1 - Deploy or Redeploy process *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	manifestName := man.Manifest.Search("name").Data().(string)
	if command == "deploy" {
		// Check if data service already exist
		containerExists := DataServiceExist(manifestID, version)
		if containerExists {
			log.Error(fmt.Sprintf("\tData service %v, %v already exist", manifestID, version))
			LogStatus(manifestID, "DEPLOY_FAILED", fmt.Sprintf("Data service %v, %v already exist", manifestID, version))
			return false
		}
	} else if command == "redeploy" {
		// Clean old data service resources
		result := UndeployDataService(manifestID, version)
		if !result {
			log.Error("\tError while cleaning old data service - ", result)
			LogStatus(manifestID, "REDEPLOY_FAILED", "Undeployment failed")
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
				LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", msg)
				return false
			}
		}
	}

	if failed {
		LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", "Process failed")
		return false
	}

	//******** STEP 3 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", err.Error())
		return false
	}

	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = man.GetLabels()

	networkName := docker.GetNetworkName(manifestName)
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", err.Error())
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
		LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", err.Error())
		return false
	}
	for _, startCommand := range contianers_cmd {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag, startCommand)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand)
		if err != nil {
			failed = true
			log.Error("Failed to create and start container: " + imageAndTag)
			LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", err.Error())
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
		LogStatus(manifestID, strings.ToUpper(command)+"_FAILED", "Process failed")
		return false
	}
	LogStatus(manifestID, strings.ToUpper(command)+"ED", strings.Title(command)+"ed successfully")

	log.Info("Completed")

	// TODO: Proper return/error handling
	return true
}

func StopDataService(manifestID string, version string) bool {
	log.Info("Stopping data service:", manifestID, version)
	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		LogStatus(manifestID, "STOPPED", "Stopped successfully")
		return true
	}
	for _, container := range containers {
		if container.State == "running" {
			log.Info("\tStopping container:", strings.Join(container.Names[:], ","))
			status := docker.StopContainer(container.ID)
			if !status {
				log.Error("\tCould not stop a container")
				LogStatus(manifestID, "STOP_FAILED", "Could not stop a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		}
	}
	LogStatus(manifestID, "STOPPED", "Stopped successfully")
	return true
}

func StartDataService(manifestID string, version string) bool {
	log.Info("Starting data service:", manifestID, version)
	containers := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) == 0 {
		LogStatus(manifestID, "START_FAILED", "No data service containers found")
		return false
	}
	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("\tStarting container:", strings.Join(container.Names[:], ","))
			status := docker.StartContainer(container.ID)
			if !status {
				log.Error("\tCould not start a container")
				LogStatus(manifestID, "START_FAILED", "Could not start a container")
				return false
			}
			log.Info("\t", strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}
	LogStatus(manifestID, "STARTED", "Started successfully")
	return true
}

func UndeployDataService(manifestID string, version string) bool {
	log.Info("Undeploying ", manifestID, version)

	// Set up Background Context and Client for Docker API calls
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		LogStatus(manifestID, "UNDEPLOY_FAILED", err.Error())
		return false
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	imageContainers := make(map[string]int)

	containers := docker.ReadAllContainers()
	if containers == nil {
		log.Error("Failed to read all containers.")
		LogStatus(manifestID, "UNDEPLOY_FAILED", "Failed to read all containers")
		return false
	}
	dsContainers := docker.ReadDataServiceContainers(manifestID, version)
	if dsContainers == nil {
		log.Error("Failed to read data service containers.")
		LogStatus(manifestID, "UNDEPLOY_FAILED", "Failed to read data service containers")
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
					LogStatus(manifestID, "UNDEPLOY_FAILED", err.Error())
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
				LogStatus(manifestID, "UNDEPLOY_FAILED", err.Error())
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
		LogStatus(manifestID, "UNDEPLOY_FAILED", err.Error())
		return false
	}
	log.Info("Pruned networks:", pruneReport)

	LogStatus(manifestID, "UNDEPLOYED", "Undeployed successfully")

	// Remove records from manifest.jsonl
	deleted := jsonlines.Delete(constants.ManifestFile, "", "")
	if !deleted {
		log.Error("Could not remove old records from manifest.jsonl")
	}

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

func LogStatus(manifestID string, statusVal string, statusReason string) {
	mani := jsonlines.Read(constants.ManifestFile, "id", manifestID, nil, false)
	deleted := jsonlines.Delete(constants.ManifestFile, "id", manifestID)
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
