package dataservice

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const CMDDeploy = "deploy"
const CMDReDeploy = "redeploy"
const CMDDeployLocal = "local_deploy"
const CMDStopService = "stopservice"
const CMDStartService = "startservice"
const CMDUndeploy = "undeploy"

func DeployDataService(man manifest.Manifest, command string) error {
	//******** STEP 1 - Check if Data Service is already deployed *************//
	containerCount := len(man.Modules)

	deploymentID := man.ManifestUniqueID.ManifestName + "-" + man.ManifestUniqueID.VersionName + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(man.ManifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
		return err
	}

	if dataServiceExists {
		if command == CMDDeploy {
			log.Info(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", man.ManifestUniqueID.ManifestName, man.ManifestUniqueID.VersionName))
			return errors.New("data service already exists")

		} else if command == CMDReDeploy || command == CMDDeployLocal {
			// Clean old data service resources
			err := UndeployDataService(man.ManifestUniqueID)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
				return errors.New("redeployment failed")
			}
		}
	}

	manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Initiated, true)

	//******** STEP 2 - Pull all images *************//
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for _, module := range man.Modules {
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
			log.Info(deploymentID, "Pulling ", imgDetails.ImageName)
			err = docker.PullImage(imgDetails)
			if err != nil {
				msg := "Unable to pull image/s, " + err.Error()
				log.Error(deploymentID, msg)
				manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.ManifestUniqueID.ManifestName, man.Labels)
	if err != nil {
		log.Error(err)
		manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
		return err
	}

	man.UpdateManifest(networkName)

	log.Info(deploymentID, "Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
		log.Info(deploymentID, "Initiating rollback ...")
		UndeployDataService(man.ManifestUniqueID)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container ", containerConfig.ContainerName)
			manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Error, false)
			log.Info(deploymentID, "Initiating rollback ...")
			UndeployDataService(man.ManifestUniqueID)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	manifest.SetStatus(man.ID, containerCount, man.ManifestUniqueID, manifest.Running, false)

	return nil
}

func StopDataService(manifestUniqueID model.ManifestUniqueID) error {
	const stateRunning = "running"

	log.Info("Stopping data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionName)

	containers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error("Failed to read data service containers.")
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return err
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Paused, true)

	for _, container := range containers {
		if container.State == stateRunning {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
				return err

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Paused, false)

	return nil
}

func StartDataService(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Starting data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionName)

	const stateExited = "exited"
	const stateCreated = "created"
	const statePaused = "paused"

	containers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error("Failed to read data service containers.")
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return err
	}

	if len(containers) == 0 {
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return errors.New("no data service containers found")
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Running, true)

	for _, container := range containers {
		if container.State == stateExited || container.State == stateCreated || container.State == statePaused {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Errorln("Could not start a container", err)
				manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
				return err
			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Running, false)

	return nil
}

func UndeployDataService(manifestUniqueID model.ManifestUniqueID) error {
	log.Info("Undeploying data service ...", manifestUniqueID.ManifestName, manifestUniqueID.VersionName)

	deploymentID := manifestUniqueID.ManifestName + "-" + manifestUniqueID.VersionName + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return err
	}

	if !dataServiceExists {
		log.Errorln(deploymentID, "Data service", manifestUniqueID.ManifestName, manifestUniqueID.VersionName, "does not exist")
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return nil
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Deleted, true)

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	numContainersPerImage := make(map[string]int)

	dsContainers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		numContainersPerImage[dsContainer.ImageID] = 0

		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
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
				manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus("", 0, manifestUniqueID, manifest.Error, false)
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	manifest.SetStatus("", 0, manifestUniqueID, manifest.Deleted, false)

	if errorlist != "" {
		log.Error(deploymentID, err)
		return errors.New("Data Service could not be undeployed completely. Cause(s): " + errorlist)
	} else {
		return nil
	}
}

func DataServiceExist(manifestUniqueID model.ManifestUniqueID) (bool, error) {
	networks, err := docker.ReadDataServiceNetworks(manifestUniqueID)
	if err != nil {
		return false, err
	}
	if len(networks) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}
