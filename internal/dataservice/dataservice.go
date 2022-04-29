package dataservice

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const ManifestFile = "manifests.jsonl"
const ManifestLogFile = "manifests_log.jsonl"

func DeployDataService(manifest model.Manifest, command string) error {
	//******** STEP 1 - Check if Data Service is already deployed *************//
	deploymentID := manifest.ID + "-" + manifest.Version + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(manifest.ID, manifest.Version)
	if err != nil {
		log.Error(deploymentID, err)
		model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"_FAILED")
		return err
	}

	if dataServiceExists {
		if command == "deploy" {
			log.Info(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", manifest.ID, manifest.Version))
			return errors.New("data service already exists")

		} else if command == "redeploy" {
			// Clean old data service resources
			err := UndeployDataService(manifest.ID, manifest.Version)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"_FAILED")
				return errors.New("redeployment failed")
			}
		}
	}

	//******** STEP 2 - Pull all images *************//
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for _, module := range manifest.Modules {
		imgDetails := module.Registry
		// Check if image exist in local
		exists, err := docker.ImageExists(imgDetails.ImageName)
		if err != nil {
			log.Error(deploymentID, err)
			return err
		}
		if exists { // Image already exists, continue
			log.Info(deploymentID, fmt.Sprintf("Image %v, already exists on host", imgDetails.ImageName))
		} else { // Pull this image
			log.Info(deploymentID, fmt.Sprintf("Image %v, does not exist on host", imgDetails.ImageName))
			log.Info(deploymentID, "Pulling ", imgDetails.ImageName, imgDetails)
			err = docker.PullImage(imgDetails)
			if err != nil {
				msg := "Unable to pull image/s, " + err.Error()
				log.Error(deploymentID, msg)
				model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command))
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(manifest.Name, manifest.Labels)
	if err != nil {
		log.Error(err)
		model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"_FAILED")
		return err
	}

	manifest.UpdateManifest(networkName)

	log.Info("deploymentID, Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := manifest.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"_FAILED")
		log.Info(deploymentID, "Initiating rollback ...")
		UndeployDataService(manifest.ID, manifest.Version)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag, " ", containerConfig)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container", containerConfig.ContainerName)
			model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"_FAILED")
			log.Info(deploymentID, "Initiating rollback ...")
			UndeployDataService(manifest.ID, manifest.Version)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	model.SetStatus(manifest.ID, manifest.Version, strings.ToUpper(command)+"ED")

	return nil
}

func StopDataService(manifestID string, version string) error {
	log.Info("Stopping data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		model.SetStatus(manifestID, version, "STOP_SERVICE_FAILED")
		return err
	}

	for _, container := range containers {
		if container.State == "running" {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				model.SetStatus(manifestID, version, "STOP_CONTAINER_FAILED")
				return err

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State)
		}
	}

	model.SetStatus(manifestID, version, "STOPPED")

	return nil
}

func StartDataService(manifestID string, version string) error {
	log.Info("Starting data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	if len(containers) == 0 {
		model.SetStatus(manifestID, version, "START_FAILED")
		return errors.New("no data service containers found")
	}

	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Error("Could not start a container", err)
				model.SetStatus(manifestID, version, "START_FAILED")
				return err
			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	model.SetStatus(manifestID, version, "STARTED")

	return nil
}

func UndeployDataService(manifestID string, version string) error {
	log.Info("Undeploying data service ...", manifestID, version)

	deploymentID := manifestID + "-" + version + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	if !dataServiceExists {
		log.Errorf(deploymentID, "Data service %v, %v does not exist", manifestID, version)
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return nil
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	numContainersPerImage := make(map[string]int)

	dsContainers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		numContainersPerImage[dsContainer.ImageID] = 0

		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	for imageID := range numContainersPerImage {
		for _, container := range containers {
			if container.ImageID == imageID {
				numContainersPerImage[imageID]++
			}
		}

		if numContainersPerImage[imageID] == 0 {
			log.Info(deploymentID, "Remove Image - ", imageID)
			err := docker.ImageRemove(imageID)
			if err != nil {
				log.Error(deploymentID, err)
				model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		model.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	model.SetStatus(manifestID, version, "UNDEPLOYED")

	if errorlist != "" {
		log.Error(deploymentID, err)
		return errors.New("Data Service could not be undeployed completely. Cause(s): " + errorlist)
	} else {
		return nil
	}
}

func DataServiceExist(manifestID string, version string) (bool, error) {
	networks, err := docker.ReadDataServiceNetworks(manifestID, version)
	if err != nil {
		return false, err
	}

	if len(networks) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}
