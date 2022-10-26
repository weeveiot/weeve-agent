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
	CMDStopService   = "STOP"
	CMDResumeService = "RESUME"
	CMDUndeploy      = "UNDEPLOY"
	CMDRemove        = "REMOVE"
)

func DeployDataService(man manifest.Manifest) error {
	deploymentID := man.ManifestUniqueID.ManifestName + "-" + man.ManifestUniqueID.VersionNumber + " | "

	log.Info(deploymentID, "Deploying data service ...")

	//******** STEP 1 - Check if Data Service is already deployed *************//
	dataServiceExists, err := DataServiceExist(man.ManifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		return err
	}

	if dataServiceExists {
		log.Warn(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", man.ManifestUniqueID.ManifestName, man.ManifestUniqueID.VersionNumber))
		return nil
	}

	manifest.AddKnownManifest(man)

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
				setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.ManifestUniqueID.ManifestName, man.Labels)
	if err != nil {
		log.Error(err)
		setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
		return err
	}

	man.UpdateManifest(networkName)

	log.Info(deploymentID, "Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
		log.Info(deploymentID, "Initiating rollback ...")
		RemoveDataService(man.ManifestUniqueID)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container ", containerConfig.ContainerName)
			log.Info(deploymentID, "Initiating rollback ...")
			RemoveDataService(man.ManifestUniqueID)
			setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	setAndSendStatus(man.ManifestUniqueID, model.EdgeAppRunning)

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
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.New("no data service containers found")
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	for _, container := range containers {
		if container.State == strings.ToLower(model.ModuleRunning) {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)

				return err
			}

			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppStopped)

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
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return err
	}

	if len(containers) == 0 {
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.New("no data service containers found")
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	for _, container := range containers {
		if container.State != strings.ToLower(model.ModuleRunning) {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Errorln("Could not start a container", err)
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)
				return err
			}

			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppRunning)

	return nil
}

func UndeployDataService(manifestUniqueID model.ManifestUniqueID) error {
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

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	//******** STEP 1 - Stop and Remove Containers *************//
	dsContainers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			setAndSendStatus(manifestUniqueID, model.EdgeAppError)
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestUniqueID)
	if err != nil {
		log.Error(deploymentID, err)
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	if errorlist != "" {
		return errors.New("Data Service could not be undeployed completely. Cause(s): " + errorlist)
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppUndeployed)

	return nil
}

func RemoveDataService(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Removing data service:", manifestUniqueID.ManifestName, manifestUniqueID.VersionNumber)

	deploymentID := manifestUniqueID.ManifestName + "-" + manifestUniqueID.VersionNumber + " | "

	//******** STEP 1 - Undeploy the data service *************//
	err := UndeployDataService(manifestUniqueID)
	if err != nil {
		return err
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	usedImages, err := docker.GetImagesByName(manifest.GetUsedImages(manifestUniqueID))
	if err != nil {
		log.Error(deploymentID, "Failed to read the used images.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return err
	}

	numContainersPerImage := make(map[string]int) // map { imageID: number_of_allocated_containers }
	for _, image := range usedImages {
		numContainersPerImage[image.ID] = 0
	}
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return err
	}

	var errorlist string
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
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	if errorlist != "" {
		return errors.New("Data Service could not be removed completely. Cause(s): " + errorlist)
	}

	manifest.DeleteKnownManifest(manifestUniqueID)
	err = SendStatus()
	if err != nil {
		log.Error(deploymentID, err)
		return err
	}

	return nil
}

func UndeployAll() error {
	log.Info("Undeploying all edge apps")

	for uniqueID := range manifest.GetKnownManifests() {
		err := RemoveDataService(uniqueID)
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

func setAndSendStatus(manifestUniqueID model.ManifestUniqueID, status string) {
	err := manifest.SetStatus(manifestUniqueID, status)
	if err != nil {
		log.Error(err)
		return
	}

	err = SendStatus()
	if err != nil {
		log.Error(err)
	}
}
