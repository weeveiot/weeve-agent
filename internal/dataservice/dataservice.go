package dataservice

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

const CMDDeploy = "deploy"
const CMDReDeploy = "redeploy"
const CMDCheckVersion = "CheckVersion"
const CMDStopService = "stopservice"
const CMDStartService = "startservice"
const CMDUndeploy = "undeploy"

func DeployDataService(man manifest.Manifest, command string) error {
	//******** STEP 1 - Check if Data Service is already deployed *************//
	deploymentID := man.ID + "-" + man.Version + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(man.ID, man.Version)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"_FAILED")
		return err
	}

	if dataServiceExists {
		if command == CMDDeploy {
			log.Info(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", man.ID, man.Version))
			return errors.New("data service already exists")

		} else if command == CMDReDeploy {
			// Clean old data service resources
			err := UndeployDataService(man.ID, man.Version)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"_FAILED")
				return errors.New("redeployment failed")
			}
		}
	}

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
				manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command))
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.Name, man.Labels)
	if err != nil {
		log.Error(err)
		manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"_FAILED")
		return err
	}

	man.UpdateManifest(networkName)

	log.Infoln(deploymentID, "Created network >>", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"_FAILED")
		log.Info(deploymentID, "Initiating rollback ...")
		UndeployDataService(man.ID, man.Version)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container", containerConfig.ContainerName)
			manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"_FAILED")
			log.Info(deploymentID, "Initiating rollback ...")
			UndeployDataService(man.ID, man.Version)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	manifest.SetStatus(man.ID, man.Version, strings.ToUpper(command)+"ED")

	return nil
}

func StopDataService(manifestID string, version string) error {
	const stateRunning = "running"

	log.Info("Stopping data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		manifest.SetStatus(manifestID, version, "STOP_SERVICE_FAILED")
		return err
	}

	for _, container := range containers {
		if container.State == stateRunning {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				manifest.SetStatus(manifestID, version, "STOP_CONTAINER_FAILED")
				return err

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	manifest.SetStatus(manifestID, version, "STOPPED")

	return nil
}

func StartDataService(manifestID string, version string) error {
	log.Infoln("Starting data service:", manifestID, version)

	const stateExited = "exited"
	const stateCreated = "created"
	const statePaused = "paused"

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	if len(containers) == 0 {
		manifest.SetStatus(manifestID, version, "START_FAILED")
		return errors.New("no data service containers found")
	}

	for _, container := range containers {
		if container.State == stateExited || container.State == stateCreated || container.State == statePaused {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Errorln("Could not start a container", err)
				manifest.SetStatus(manifestID, version, "START_FAILED")
				return err
			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	manifest.SetStatus(manifestID, version, "STARTED")

	return nil
}

func UndeployDataService(manifestID string, version string) error {
	log.Info("Undeploying data service ...", manifestID, version)

	deploymentID := manifestID + "-" + version + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	if !dataServiceExists {
		log.Errorln(deploymentID, "Data service", manifestID, version, "does not exist")
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return nil
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	numContainersPerImage := make(map[string]int)

	dsContainers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		numContainersPerImage[dsContainer.ImageID] = 0

		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
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
				manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		manifest.SetStatus(manifestID, version, "UNDEPLOY_FAILED")
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	manifest.SetStatus(manifestID, version, "UNDEPLOYED")

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
