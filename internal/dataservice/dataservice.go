package dataservice

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
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
		log.Errorf("Deployment failed! DeploymentID --> %s, CAUSE --> %v", deploymentID, err)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	if dataServiceExists {
		log.Warn(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", man.ManifestUniqueID.ManifestName, man.ManifestUniqueID.VersionNumber))
		return nil
	}

	manifest.AddKnownManifest(man)

	//******** STEP 2 - Pull all images *************//
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for _, module := range man.Modules {
		// Check if image exist in local
		exists, err := docker.ImageExists(module.ImageName)
		if err != nil {
			log.Errorf("Deployment failed! DeploymentID --> %s, CAUSE --> %v", deploymentID, err)
			return errors.Wrap(err, traceutility.FuncTrace())
		}
		if exists { // Image already exists, continue
			log.Info(deploymentID, fmt.Sprintf("Image %v, already exists on host", module.ImageName))
		} else { // Pull this image
			log.Info(deploymentID, fmt.Sprintf("Image %v, does not exist on host", module.ImageName))
			log.Info(deploymentID, "Pulling ", module.ImageName)
			err = docker.PullImage(module.AuthConfig, module.ImageName)
			if err != nil {
				log.Error(deploymentID, "Unable to pull image/s, "+err.Error())
				setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
				log.Info(deploymentID, "Initiating rollback ...")
				RemoveDataService(man.ManifestUniqueID)
				return errors.New("unable to pull image/s")
			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.ManifestUniqueID.ManifestName, man.Labels)
	if err != nil {
		log.Error("CreateNetwork failed! CAUSE --> ", err)
		setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
		log.Info(deploymentID, "Initiating rollback ...")
		RemoveDataService(man.ManifestUniqueID)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	man.UpdateManifest(networkName)

	log.Info(deploymentID, "Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid containers in Manifest")
		setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
		log.Info(deploymentID, "Initiating rollback ...")
		RemoveDataService(man.ManifestUniqueID)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error("CreateAndStartContainer failed! CAUSE --> ", deploymentID, err)
			log.Error(deploymentID, "Failed to create and start container ", containerConfig.ContainerName)
			log.Info(deploymentID, "Initiating rollback ...")
			RemoveDataService(man.ManifestUniqueID)
			setAndSendStatus(man.ManifestUniqueID, model.EdgeAppError)
			return errors.Wrap(err, traceutility.FuncTrace())
		}
		log.Info(deploymentID, "Successfully created container ", containerID)
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
		log.Error("Failed to read data service containers! CAUSE --> ", err)
		return errors.Wrap(err, traceutility.FuncTrace())
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
				log.Error("Could not stop a container! CAUSE --> ", err)
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)

				return errors.Wrap(err, traceutility.FuncTrace())
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
		log.Error("Unable to resume data service! CAUSE --> ", err)
		log.Error("Failed to read data service containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.Wrap(err, traceutility.FuncTrace())
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
				return errors.Wrap(err, traceutility.FuncTrace())
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
		log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	if !dataServiceExists {
		log.Warnln(deploymentID, "Trying to undeploy a non-existant edge application with ManifestName: ", manifestUniqueID.ManifestName, " and VersionNumber: ", manifestUniqueID.VersionNumber)
		return nil
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	//******** STEP 1 - Stop and Remove Containers *************//
	dsContainers, err := docker.ReadDataServiceContainers(manifestUniqueID)
	if err != nil {
		log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
		log.Error(deploymentID, "Failed to read data service containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
			setAndSendStatus(manifestUniqueID, model.EdgeAppError)
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestUniqueID)
	if err != nil {
		log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
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
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	usedImageNames, err := manifest.GetUsedImages(manifestUniqueID)
	if err != nil {
		log.Errorf("Data service removal failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	usedImageIDs, err := docker.GetImagesByName(usedImageNames)
	if err != nil {
		log.Error("Unable to get images! CAUSE --> ", err)
		log.Error(deploymentID, "Failed to read the used images.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	numContainersPerImage := make(map[string]int) // map { imageID: number_of_allocated_containers }
	for _, image := range usedImageIDs {
		numContainersPerImage[image.ID] = 0
	}
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error("Unable to read containers! CAUSE --> ", err)
		log.Error(deploymentID, "Failed to read all containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.Wrap(err, traceutility.FuncTrace())
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
				log.Errorf("Data service removal failed! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
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
		log.Errorf("Failed to delete known manifest! UndeploymentID --> %s, CAUSE --> %v", deploymentID, err)
		return errors.Wrap(err, traceutility.FuncTrace())
	}

	return nil
}

func UndeployAll() error {
	log.Info("Undeploying all edge apps")

	for uniqueID := range manifest.GetKnownManifests() {
		err := RemoveDataService(uniqueID)
		if err != nil {
			return errors.Wrap(err, traceutility.FuncTrace())
		}
	}

	return nil
}

func DataServiceExist(manifestUniqueID model.ManifestUniqueID) (bool, error) {
	networks, err := docker.ReadDataServiceNetworks(manifestUniqueID)
	if err != nil {
		return false, errors.Wrap(err, traceutility.FuncTrace())
	}
	if len(networks) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func setAndSendStatus(manifestUniqueID model.ManifestUniqueID, status string) {
	log.Debug("Setting and sending data service status...")

	err := manifest.SetStatus(manifestUniqueID, status)
	if err != nil {
		log.Error("SetStatus failed! CAUSE --> ", err)
		return
	}

	err = SendStatus()
	if err != nil {
		log.Error("SendStatus failed! CAUSE --> ", err)
	}
}
