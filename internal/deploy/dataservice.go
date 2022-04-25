package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util/jsonlines"
)

const ManifestFile = "manifests.jsonl"
const ManifestLogFile = "manifests_log.jsonl"

func DeployDataService(man model.Manifest, command string, local bool) error {

	var err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		return err
	}

	jsonlines.Insert(ManifestLogFile, man.Manifest.String())

	//******** STEP 1 - Check if Data Service is already deployed *************//
	manifestID := man.Manifest.Search("id").Data().(string)
	version := man.Manifest.Search("version").Data().(string)
	manifestName := man.Manifest.Search("name").Data().(string)
	deploymentID := manifestID + "-" + version + " | "

	log.Info(deploymentID, fmt.Sprintf("%ving data service ...", command))

	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		logStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return err
	}

	if dataServiceExists {
		if command == "deploy" && !local {
			log.Info(deploymentID, fmt.Sprintf("Data service %v, %v already exist!", manifestID, version))
			return errors.New("data service already exists")

		} else if command == "redeploy" {
			// Clean old data service resources
			err := UndeployDataService(manifestID, version)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				logStatus(manifestID, version, "REDEPLOY_FAILED", "Undeployment failed")
				return errors.New("redeployment failed")
			}
		} else if command == "deploy" && local {
			// Clean old data service resources
			err := UndeployDataService(manifestID, version)
			if err != nil {
				log.Error(deploymentID, "Error while cleaning old data service -> ", err)
				logStatus(manifestID, version, "REDEPLOY_FAILED", "Undeployment failed")
				return errors.New("redeployment failed")
			}
		}
	}

	filter := map[string]string{"id": manifestID, "version": version}
	jsonlines.Delete(ManifestFile, filter, true)

	// need to set some default manifest in manifest.jsonl so later could log without errors
	man.Manifest.Set("DEPLOYING_IN_PROGRESS", "status")
	jsonlines.Insert(ManifestFile, man.Manifest.String())

	//******** STEP 2 - Pull all images *************//
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for _, imgDetails := range man.ImageNamesWithRegList() {
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
				logStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", msg)
				return errors.New("unable to pull image/s")

			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(manifestName, man.GetLabels())
	if err != nil {
		log.Error(err)
		logStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		return err
	}

	log.Info("deploymentID, Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.GetContainerConfig(networkName)

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid contianers in Manifest")
		logStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
		log.Info(deploymentID, "Initiating rollback ...")
		UndeployDataService(manifestID, version)
		return errors.New("no valid contianers in manifest")
	}

	for _, containerConfig := range containerConfigs {
		log.Info(deploymentID, "Creating ", containerConfig.ContainerName, " from ", containerConfig.ImageName, ":", containerConfig.ImageTag, " ", containerConfig)
		containerID, err := docker.CreateAndStartContainer(containerConfig)
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container", containerConfig.ContainerName)
			logStatus(manifestID, version, strings.ToUpper(command)+"_FAILED", err.Error())
			log.Info(deploymentID, "Initiating rollback ...")
			UndeployDataService(manifestID, version)
			return err
		}
		log.Info(deploymentID, "Successfully created container ", containerID, " with args: ", containerConfig.EntryPointArgs)
		log.Info(deploymentID, "Started!")
	}

	logStatus(manifestID, version, strings.ToUpper(command)+"ED", strings.Title(command)+"ed successfully")

	return nil
}

func StopDataService(manifestID string, version string) error {
	log.Info("Stopping data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		logStatus(manifestID, version, "STOP_SERVICE_FAILED", "Failed to read data service containers")
		return err
	}

	for _, container := range containers {
		if container.State == "running" {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container")
				logStatus(manifestID, version, "STOP_CONTAINER_FAILED", "Could not stop a container")
				return err

			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State)
		}
	}

	logStatus(manifestID, version, "STOPPED", "Stopped successfully")

	return nil
}

func StartDataService(manifestID string, version string) error {
	log.Info("Starting data service:", manifestID, version)

	containers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error("Failed to read data service containers.")
		logStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read data service containers")
		return err
	}

	if len(containers) == 0 {
		logStatus(manifestID, version, "START_FAILED", "No data service containers found")
		return errors.New("no data service containers found")
	}

	for _, container := range containers {
		if container.State == "exited" || container.State == "created" || container.State == "paused" {
			log.Info("Starting container:", strings.Join(container.Names[:], ","))
			err := docker.StartContainer(container.ID)
			if err != nil {
				log.Error("Could not start a container", err)
				logStatus(manifestID, version, "START_FAILED", "Could not start a container")
				return err
			}
			log.Info(strings.Join(container.Names[:], ","), ": ", container.State, "--> running")
		}
	}

	logStatus(manifestID, version, "STARTED", "Started successfully")

	return nil
}

func UndeployDataService(manifestID string, version string) error {
	log.Info("Undeploying data service ...", manifestID, version)

	deploymentID := manifestID + "-" + version + " | "

	// Check if data service already exist
	dataServiceExists, err := DataServiceExist(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		logStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		return err
	}

	if !dataServiceExists {
		log.Error(fmt.Sprintf(deploymentID, "Data service %v, %v does not exist", manifestID, version))
		logStatus(manifestID, version, "UNDEPLOY_FAILED", fmt.Sprintf("Data service %v, %v does not exist", manifestID, version))
		return nil
	}

	//******** STEP 1 - Stop and Remove Containers *************//

	// map { imageID: number_of_allocated_containers }, needed for removing images as not supported by Go-Docker SDK
	numContainersPerImage := make(map[string]int)

	dsContainers, err := docker.ReadDataServiceContainers(manifestID, version)
	if err != nil {
		log.Error(deploymentID, "Failed to read data service containers.")
		logStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read data service containers")
		return err
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		numContainersPerImage[dsContainer.ImageID] = 0

		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Error(deploymentID, err)
			logStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	containers, err := docker.ReadAllContainers()
	if err != nil {
		log.Error(deploymentID, "Failed to read all containers.")
		logStatus(manifestID, version, "UNDEPLOY_FAILED", "Failed to read all containers")
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
				logStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
				errorlist = fmt.Sprintf("%v,%v", errorlist, err)
			}
		}
	}

	//******** STEP 3 - Remove Network *************//
	log.Info(deploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestID, version)
	if err != nil {
		log.Error(deploymentID, err)
		logStatus(manifestID, version, "UNDEPLOY_FAILED", err.Error())
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	logStatus(manifestID, version, "UNDEPLOYED", "Undeployed successfully")

	// Remove records from manifest.jsonl
	filterLog := map[string]string{"id": manifestID, "version": version}
	deleted := jsonlines.Delete(ManifestFile, filterLog, true)
	if !deleted {
		log.Error(deploymentID, "Could not remove old records from ", ManifestFile)
	}

	if errorlist != "" {
		log.Error(deploymentID, err)
		return errors.New("Data Service could not be undeployed completely. Cause(s): " + errorlist)
	} else {
		return nil
	}
}

// DataServiceExist returns status of data service existance as true or false
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

func logStatus(manifestID string, manifestVersion string, statusVal string, statusReason string) {
	filter := map[string]string{"id": manifestID, "version": manifestVersion}
	mani, err := jsonlines.Read(ManifestFile, filter, false)
	if err == nil {
		deleted := jsonlines.Delete(ManifestFile, filter, true)
		if !deleted {
			log.Error("Could not remove old records from ", ManifestFile)
		}
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
