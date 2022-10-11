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

const (
	CMDDeploy        = "DEPLOY"
	CMDDeployLocal   = "LOCAL_DEPLOY"
	CMDStopService   = "STOP"
	CMDResumeService = "RESUME"
	CMDUndeploy      = "UNDEPLOY"
	CMDRemove        = "REMOVE"
)

func DeployDataService(man manifest.Manifest, command string) error {
	//******** STEP 1 - Check if Data Service is already deployed *************//
	containerCount := len(man.Modules)

	deploymentID := man.ManifestUniqueID.ManifestName + "-" + man.ManifestUniqueID.VersionNumber + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(man.ManifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
		return err
	}

	if dataServiceExists {
		if command == CMDDeploy {
			log.Warn(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", man.ManifestUniqueID.ManifestName, man.ManifestUniqueID.VersionNumber))
			return nil

		} else if command == CMDDeployLocal {
			// Clean old data service resources
			err := UndeployDataService(man.ManifestUniqueID, CMDUndeploy)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
				return errors.New("local deployment failed")
			}
		}
	}

	setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppInitiated, false)

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
				setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.ManifestUniqueID.ManifestName, man.Labels)
	if err != nil {
		log.Error(err)
		setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
		return err
	}

	man.UpdateManifest(networkName)

	log.Info(deploymentID, "Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
		log.Info(deploymentID, "Initiating rollback ...")
		UndeployDataService(man.ManifestUniqueID, CMDRemove)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container ", containerConfig.ContainerName)
			log.Info(deploymentID, "Initiating rollback ...")
			UndeployDataService(man.ManifestUniqueID, CMDRemove)
			setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppError, false)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	setAndSendStatus(man.ID, containerCount, man.ManifestUniqueID, model.EdgeAppRunning, false)

	return nil
}

func StopDataService(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Stopping data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)

	status := manifest.GetEdgeAppStatus(manifestUniqueID)
	if status != model.EdgeAppRunning {
		log.Warn("Can't stop edge application with ManifestName: ", manifestUniqueID.ManifestName, " and VersionNumber: ", manifestUniqueID.VersionNumber, " with status ", status)
		return nil
	}

	containers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error("Failed to read data service containers.")
		return err
	}

	if len(containers) == 0 {
		setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
		return errors.New("no data service containers found")
	}

	setAndSendStatus("", 0, manifestUniqueID, "", true)

	for _, container := range containers {
		if container.State == strings.ToLower(model.ModuleRunning) {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)

				return err
			}

			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppStopped, false)

	return nil
}

func ResumeDataService(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Resuming data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)

	status := manifest.GetEdgeAppStatus(manifestUniqueID)
	if status != model.EdgeAppStopped {
		log.Warn("Can't resume edge application with ManifestName: ", manifestUniqueID.ManifestName, " and VersionNumber: ", manifestUniqueID.VersionNumber, " with status ", status)
		return nil
	}

	containers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error("Failed to read data service containers.")
		setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
		return err
	}

	if len(containers) == 0 {
		setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
		return errors.New("no data service containers found")
	}

	setAndSendStatus("", 0, manifestUniqueID, "", true)

	for _, container := range containers {
		if container.State != strings.ToLower(model.ModuleRunning) {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Errorln("Could not start a container", err)
				setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
				return err
			}

			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppRunning, false)

	return nil
}

func UndeployDataService(manifestUniqueID model.ManifestUniqueID, command string) error {
	log.Infoln("Undeploying data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)

	deploymentID := manifestUniqueID.ManifestName + "-" + manifestUniqueID.VersionNumber + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		return err
	}

	if !dataServiceExists {
		log.Warnln(deploymentID, "Trying to undeploy a non-existant edge application with ManifestName: ", manifestUniqueID.ManifestName, " and VersionNumber: ", manifestUniqueID.VersionNumber)
		return nil
	}

	setAndSendStatus("", 0, manifestUniqueID, "", true)

	//******** STEP 1 - Stop and Remove Containers *************//
	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	numContainersPerImage := make(map[string]int)

	dsContainers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		numContainersPerImage[dsContainer.ImageID] = 0

		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	if command == CMDRemove {
		//******** STEP 2 - Remove Images WITHOUT Containers *************//
		containers, err := docker.ReadAllContainers()
		if err != nil {
			log.Error(deploymentID, "Failed to read all containers.")
			setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
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
					setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
					errorlist = fmt.Sprintf("%v,%v", errorlist, err)
				}
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		setAndSendStatus("", 0, manifestUniqueID, model.EdgeAppError, false)
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	if errorlist != "" {
		return errors.New("Data Service could not be undeployed completely. Cause(s): " + errorlist)
	}

	manifest.DeleteKnownManifest(manifestUniqueID)
	err = SendStatus("")
	if err != nil {
		log.Error(deploymentID, err)
		return err
	}

	return nil
}

func UndeployAll() error {
	knownManifests := manifest.GetKnownManifests()
	log.Info(knownManifests)

	for _, manif := range knownManifests {
		err := UndeployDataService(manif.ManifestUniqueID, CMDRemove)
		if err != nil {
			return err
		}
	}

	return nil
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

func setAndSendStatus(manifestID string, containerCount int, manifestUniqueID model.ManifestUniqueID, status string, inTransition bool) {
	manifest.SetStatus(manifestID, containerCount, manifestUniqueID, status, inTransition)
	err := SendStatus("")
	if err != nil {
		log.Error(err)
	}
}
