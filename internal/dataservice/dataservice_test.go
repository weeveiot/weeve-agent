/*
	These unit tests assume that there exist docker containers that
	can be started and stopped using the tested functions.
*/

package dataservice_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

var manifestID = "PLACEHOLDER"
var version = "PLACEHOLDER"
var manifestID2 = "PLACEHOLDER"
var version2 = "PLACEHOLDER"

const manifestPath = "../../testdata/manifest/test_manifest.json"
const manifestPath2 = "../../testdata/manifest/test_manifest_copy.json"
const manifestPathRedeploy = "../../testdata/manifest/test_manifest_redeploy.json"

func init() {
	docker.SetupDockerClient()
}

func TestDeployManifest(t *testing.T) {
	log.Info("TESTING DEPLOYMENT...")

	// Load Manifest JSON from file.
	json, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifest, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.ID
	log.Info(manifestID)
	version = thisManifest.VersionName
	log.Info(version)

	err = dataservice.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// Check if network exists
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Error(err)
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := cli.NetworkList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}

	if len(networks) > 0 {
		// Check if containers exist
		dsContainers, _ := docker.ReadDataServiceContainers(manifestID, version)
		containers := thisManifest.Modules
		for _, dsContainer := range dsContainers {
			containersExist := false
			for _, dsContainerName := range dsContainer.Names {
				for _, container := range containers {
					if dsContainerName == container.ContainerName {
						containersExist = true
					}
				}
			}

			if !containersExist {
				t.Error("Container/s missing")
			}
		}
	} else {
		t.Error("Network not created")
	}
}

// Test Stop Service method
func TestStopDataServiceWrongDetails(t *testing.T) {
	// IMPORTANT: Assume all containers are exited at the beginning

	log.Info("TESTING STOP DATA SERVICE WITH WRONG DETAILS...")

	var wrongManifesetID = manifestID + "WRONG"
	var wrongVersion = version + "WRONG"
	var wrongStatusContainerList []string

	// check container status before executing tested function
	statusBefore := make(map[string]string)
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		statusBefore[container.ID] = container.State
	}

	// run tested method
	err := dataservice.StopDataService(wrongManifesetID, wrongVersion)
	if err != nil {
		t.Errorf("StopDataService returned True status, but should return False as manifestID is wrong")
	}

	// check container status after executing tested function
	containers, _ = docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State != statusBefore[container.ID] {
			wrongStatusContainerList = append(wrongStatusContainerList, container.ID)
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers status changed when should have not: %v", wrongStatusContainerList)
	}
}

func TestStopDataService(t *testing.T) {
	// IMPORTANT: Assume all containers are exited at the beginning

	log.Info("TESTING STOP DATA SERVICE...")

	var wrongStatusContainerList []string

	// run tested method
	err := dataservice.StopDataService(manifestID, version)
	if err != nil {
		t.Errorf("StopDataService returned %v status", err)
	}

	// check container status
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State != "exited" {
			wrongStatusContainerList = append(wrongStatusContainerList, container.ID)
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers SHOULD be 'exited': %v", wrongStatusContainerList)
	}
}

// Test Start Service method
func TestStartDataServiceWrongDetails(t *testing.T) {
	// IMPORTANT: Assume all containers are exited at the beginning

	log.Info("TESTING START DATA SERVICE WITH WRONG DETAILS...")

	var wrongServiceID = manifestID + "WRONG"
	var wrongServiceName = version + "WRONG"
	var wrongStatusContainerList []string

	// check container status before executing tested function
	statusBefore := make(map[string]string)
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		statusBefore[container.ID] = container.State
	}

	// run tested method
	err := dataservice.StartDataService(wrongServiceID, wrongServiceName)
	if err != nil {
		t.Errorf("StartDataService returned %v status", err)
	}

	// check container status after executing tested function
	containers, _ = docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State != statusBefore[container.ID] {
			wrongStatusContainerList = append(wrongStatusContainerList, container.ID)
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers status changed when should have not: %v", wrongStatusContainerList)
	}
}

func TestStartDataService(t *testing.T) {
	// IMPORTANT: Assume all containers are exited at the beginning

	log.Info("TESTING START DATA SERVICE...")

	var wrongStatusContainerList []string

	// run tested method
	err := dataservice.StartDataService(manifestID, version)
	if err != nil {
		t.Errorf("StartDataService returned %v status", err)
	}

	// check container status
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	for _, container := range containers {
		if container.State != "running" {
			wrongStatusContainerList = append(wrongStatusContainerList, container.ID)
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers SHOULD be 'running': %v", wrongStatusContainerList)
	}
}

func TestUndeployDataService(t *testing.T) {
	log.Info("TESTING UNDEPLOYMENT...")

	// run tested method
	err := dataservice.UndeployDataService(manifestID, version)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

	// check if containers are removed
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) > 0 {
		t.Errorf("The following containers should have been removed: %v", containers)
	}

	// Check if the network is removed
	result, _ := dataservice.DataServiceExist(manifestID, version)
	if result {
		t.Errorf("Network %v was not pruned (Data Service not removed)", version)
	}
}

func TestUndeployDataService2SameServices(t *testing.T) {
	// testing 2 identical services, one should be later undeployed and another should still run
	log.Info("TESTING UNDEPLOYMENT WHEN SECOND IDENTICAL DATA SERVICE EXISTS...")

	// ***** DEPLOY ORIGINAL DATA SERVICE ********* //
	// Load Manifest JSON from file.
	log.Info("Loading original manifest...")

	json, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifest, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.ID
	log.Info(manifestID)
	version = thisManifest.VersionName
	log.Info(version)

	err = dataservice.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***** DEPLOY SECOND IDENTICAL DATA SERVICE ********* //
	// Load Manifest JSON from file.
	log.Info("Loading second manifest...")

	json, err = ioutil.ReadFile(manifestPath2)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err = gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifest2, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// Fill the placeholders for Start and Stop tests
	manifestID2 = thisManifest2.ID
	log.Info(manifestID2)
	version2 = thisManifest2.VersionName
	log.Info(version2)

	err = dataservice.DeployDataService(thisManifest2, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***** TEST UNDEPLOY FOR ORIGINAL DATA SERVICE ********* //

	// run tested method
	err = dataservice.UndeployDataService(manifestID, version)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

	// check if containers are removed
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) > 0 {
		t.Errorf("The following containers should have been removed: %v", containers)
	}

	// Check if the network is removed
	result, _ := dataservice.DataServiceExist(manifestID, version)
	if result {
		t.Errorf("Network was not pruned (Data Service not removed)")
	}

	// ***** CHECK IF SECOND IDENTICAL DATA SERVICE STILL EXISTS ********* //
	expectedNumberContainers := len(thisManifest2.Modules)
	dsContainers, _ := docker.ReadDataServiceContainers(manifestID2, version2)

	if len(dsContainers) != expectedNumberContainers {
		t.Errorf("Some containers from the second identical network were removed.")
	}

	result2, _ := dataservice.DataServiceExist(manifestID2, version2)
	if !result2 {
		t.Errorf("Second identical network is removed.")
	}

	// clean up and remove second data service
	err = dataservice.UndeployDataService(manifestID2, version2)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

}

func TestRedeployDataService(t *testing.T) {
	log.Info("TESTING REDEPLOYMENT...")

	// ***************** LOAD ORIGINAL MANIFEST AND DEPLOY DATA SERVICE ******************** //
	log.Info("Loading original manifest...")
	json, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifest, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.ID
	log.Info(manifestID)
	version = thisManifest.VersionName
	log.Info(version)

	err = dataservice.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***************** SAVE ORIGINAL DATA SERVICE TIMESTAMP AND ID ******************** //
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Error(err)
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := cli.NetworkList(ctx, options)
	if err != nil {
		t.Error(err)
	}

	originalServiceTimestamp := networks[0].Created
	originalServiceID := networks[0].ID

	// ***************** LOAD ORIGINAL MANIFEST AND DEPLOY DATA SERVICE ******************** //
	log.Info("Loading redeployment manifest...")
	json, err = ioutil.ReadFile(manifestPathRedeploy)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err = gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifestRedeploy, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.ID
	log.Info(manifestID)
	version = thisManifest.VersionName
	log.Info(version)

	err = dataservice.DeployDataService(thisManifestRedeploy, "redeploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***************** CHECK REDEPLOYMENT's SUCCESS ******************** //
	// compare new and old networks
	networksRe, err := cli.NetworkList(ctx, options)
	if err != nil {
		t.Error(err)
	}

	if originalServiceID == networksRe[0].ID || originalServiceTimestamp == networksRe[0].Created {
		t.Errorf("New network was not created.")
	}

	// ***************** CLEANING AFTER TESTING ******************** //
	log.Info("Cleaning after testing...")
	redeployedManifestID := thisManifestRedeploy.ID
	redeployedVersion := thisManifestRedeploy.VersionName
	err = dataservice.UndeployDataService(redeployedManifestID, redeployedVersion)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}
}
