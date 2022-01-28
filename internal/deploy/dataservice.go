package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

func DeployManifest(man model.Manifest, command string) error {

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return err
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(ManifestLogFile, man.Manifest.String())

	//******** STEP 1 - Deploy or Redeploy process *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	manifestName := man.Manifest.Search("name").Data().(string)
	deploymentID := manifestID + "-" + version + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return err
	}
	// Check if data service already exist
	if dataServiceExists {
		if command == "deploy" {
			// msg := "Data service already Exist!"
			// log.Error(msg)
			log.Info(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", manifestID, version))
			return errors.New("data service already exist")

		} else if command == "redeploy" {
			// Clean old data service resources
			err := UndeployDataService(manifestID, version)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				LogStatus(manifestID, version, "REDEPLOY_FAILED", "Undeployment failed")
				return errors.New("redeployment failed")

			}
		}
	}

	filter := map[string]string{"id": man.Manifest.Search("id").Data().(string), "version": man.Manifest.Search("version").Data().(string)}
	jsonlines.Delete(ManifestFile, "", "", filter, true)

	// need to set some default manifest in manifest.jsonl so later could log without errors
	man.Manifest.Set("DEPLOYING_IN_PROGRESS", "status")
	jsonlines.Insert(ManifestFile, man.Manifest.String())

	//******** STEP 2 - Pull all images *************//
	// Pull all images as required
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for i, imgDetails := range man.ImageNamesWithRegList() {
		// Check if image exist in local
		exists, err := docker.ImageExists(imgDetails.ImageName)
		if err != nil {
			log.Error(deploymentID, err)
			return errors.New("images exists")

		}
		if exists { // Image already exists, continue
			log.Info(deploymentID, fmt.Sprintf("Image %v %v, already exists on host", i, imgDetails.ImageName))
		} else { // Pull this image
			log.Info(deploymentID, fmt.Sprintf("Image %v %v, does not exist on host", i, imgDetails.ImageName))
			log.Info(deploymentID, "Pulling ", imgDetails.ImageName, imgDetails)
			err = docker.PullImage(imgDetails)
			if err != nil {
				msg := "404 - Unable to pull image/s, one or more image/s not found"
				log.Error(deploymentID, msg)
				LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", msg)
				return errors.New("unable to pull image/s, one or more image/s not found")

			}
		}
	}

	if failed {
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Process failed")
		return errors.New("failed to pull images")
	}

	//******** STEP 3 - Create the network *************//
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return err
	}

	log.Info(deploymentID, "Creating network ...")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	networkCreateOptions.Labels = man.GetLabels()

	networkName := docker.GetNetworkName(manifestName)
	if networkName == "" {
		log.Error(deploymentID, "Failed to generate Network Name")
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Failed to generate Network Name")
		return errors.New("failed to generate network name")

	}

	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return errors.New("failed to create network")
	}
	log.Info("deploymentID, Created network >> ", networkName)

	_ = resp
	// log.Info(resp.ID, resp.Warning)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	var contianers_cmd = man.GetContainerStart(networkName)

	if contianers_cmd == nil || len(contianers_cmd) <= 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		log.Debug(deploymentID, "Initiating rollback ...")
		UndeployDataService(manifestID, version)
		return err
	}
	for _, startCommand := range contianers_cmd {
		log.Info(deploymentID, "Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag, startCommand)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand)
		if err != nil {
			failed = true

			log.Error(deploymentID, "Failed to create and start container: "+imageAndTag)
			LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
			UndeployDataService(manifestID, version)
			return err

		}
		log.Info(deploymentID, "Successfully created with args: ", startCommand.EntryPointArgs, containerCreateResponse)
		log.Info(deploymentID, "Started!")
	}

	if failed {
		LogStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", "Process failed")
		UndeployDataService(manifestID, version)
		return errors.New("failed to start and create containers")
	}
	LogStatus(manifestID, version, strings.ToUpper(command)+"ED", strings.Title(command)+"ed successfully")

	return nil
}

func StopDataService(manifestID string, version string) error {
	log.Info("Stopping data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)

	if err != nil {
		log.Error("Failed to read data service containers.")
		LogStatus(manifestID, version, "STOP_SERVICE_FAILED", "Failed to read data service containers")
		return errors.New("failed to read data service containers")
	}
	if len(containers) == 0 {
		LogStatus(manifestID, version, "STOPPED", "Stopped successfully")

	}
	for _, container := range containers {
		if container.State == "running" {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				LogStatus(manifestID, version, "STOP_CONTAINER_FAILED", "Could not stop a container")
				return errors.New("failed to stop container")

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		}
	}

	LogStatus(manifestID, version, "STOPPED", "Stopped successfully")

	return nil
}

func StartDataService(manifestID string, version string) error {
	log.Info("Starting data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)

	if err != nil {
		log.Error("Failed to read data service containers.")
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read data service containers")
		return errors.New("failed to read data service container")
	}
	if len(containers) == 0 {
		LogStatus(manifestID, version, "START_FAILED", "No data service containers found")
		return errors.New("no data service containers found")
	}
	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			status := docker.StartContainer(container.ID)
			if !status {
				log.Error("Could not start a container")
				LogStatus(manifestID, version, "START_FAILED", "Could not start a container")
				return errors.New("failed to start container")

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	LogStatus(manifestID, version, "STARTED", "Started successfully")

	return nil
}

func UndeployDataService(manifestID string, version string) error {
	var errorlist string
	log.Info("Undeploying data service ...", manifestID, version)

	deploymentID := manifestID + "-" + version + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}
	if !dataServiceExists {
		log.Error(fmt.Sprintf(deploymentID, "Data service %v, %v does not exist", manifestID, version))
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", fmt.Sprintf("Data service %v, %v does not exist", manifestID, version))
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	// Set up Background Context and Client for Docker API calls
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	imageContainers := make(map[string]int)

	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read all containers")
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}
	dsContainers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read data service containers")
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}
	for _, container := range containers {

		imageContainers[container.ImageID] = imageContainers[container.ImageID] + 1

		for _, dsContainer := range dsContainers {
			if container.ID == dsContainer.ID {
				log.Info(deploymentID, "Stop And Remove Container - ", dsContainer.ID)
				// Stop and delete container
				err := docker.StopAndRemoveContainer(dsContainer.ID)
				if err != nil {
					log.Error(deploymentID, err)
					LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
					errorlist = fmt.Sprintf("%v,%v", errorlist, err)
				}

				imageContainers[container.ImageID] = imageContainers[container.ImageID] - 1
			}
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	for imageID, containersCount := range imageContainers {
		if containersCount == 0 {
			log.Info(deploymentID, "Remove Image - ", imageID)
			_, err := cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{})
			if err != nil {
				log.Error(deploymentID, err)
				LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")
	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	if err != nil {
		log.Error(deploymentID, err)
		LogStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}
	log.Info(deploymentID, "Pruned networks:", pruneReport, "!")

	LogStatus(manifestID, version, "UNDEPLOYED", "Undeployed successfully")

	// Remove records from manifest.jsonl
	filterLog := map[string]string{"id": manifestID, "version": version}
	deleted := jsonlines.Delete(ManifestFile, "", "", filterLog, true)
	if !deleted {
		log.Error(deploymentID, "Could not remove old records from manifest.jsonl")
	}
	if errorlist != "" {
		log.Error(deploymentID, err)
	}
	return err
}

// DataServiceExist returns status of data service existance as true or false
func DataServiceExist(manifestID string, version string) (bool, error) {
	var networks []types.NetworkResource

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, err
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err = dockerClient.NetworkList(context.Background(), options)
	if err != nil {
		return false, err
	}

	if len(networks) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func LogStatus(manifestID string, manifestVersion string, statusVal string, statusReason string) {
	filter := map[string]string{"id": manifestID, "version": manifestVersion}
	mani := jsonlines.Read(ManifestFile, "", "", filter, false)
	deleted := jsonlines.Delete(ManifestFile, "", "", filter, true)
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
		inserted := jsonlines.Insert(ManifestFile, string(maniJSON))
		if !inserted {
			log.Error("Could not log manifest status")
		}
	}
}
