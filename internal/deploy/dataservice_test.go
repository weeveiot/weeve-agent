/*
	These unit tests assume that there exist docker containers that
	can be started and stopped using the tested functions.
*/

package deploy_test

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/deploy"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/model"
	"github.com/weeveiot/weeve-agent/internal/util"
)

var manifestID = "PLACEHOLDER"
var version = "PLACEHOLDER"
var manifestID2 = "PLACEHOLDER"
var version2 = "PLACEHOLDER"

const manifestPath = "testdata/manifest/test_manifest.json"
const manifestPath2 = "testdata/manifest/test_manifest_copy.json"
const manifestPathRedeploy = "testdata/manifest/test_manifest_redeploy.json"

func TestDeployManifest(t *testing.T) {
	log.Info("TESTING DEPLOYMENT...")

	// Load Manifest JSON from file.
	json := LoadJSONBytes(manifestPath)

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	var thisManifest = model.Manifest{}
	thisManifest.Manifest = *jsonParsed

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.Manifest.Search("id").Data().(string)
	log.Info(manifestID)
	version = thisManifest.Manifest.Search("version").Data().(string)
	log.Info(version)

	err = deploy.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// Check if network exists
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
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
		containerName := thisManifest.ContainerNamesList(networks[0].Name)
		for _, dsContainer := range dsContainers {
			containersExist := false
			for _, dsContainerName := range dsContainer.Names {
				for _, containerName := range containerName {
					if dsContainerName == containerName {
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
	err := deploy.StopDataService(wrongManifesetID, wrongVersion)
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
	err := deploy.StopDataService(manifestID, version)
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
	err := deploy.StartDataService(wrongServiceID, wrongServiceName)
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
	err := deploy.StartDataService(manifestID, version)
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
	err := deploy.UndeployDataService(manifestID, version)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

	// check if containers are removed
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) > 0 {
		t.Errorf("The following containers should have been removed: %v", containers)
	}

	// Check if the network is removed
	result, _ := deploy.DataServiceExist(manifestID, version)
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

	json := LoadJSONBytes(manifestPath)

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	var thisManifest = model.Manifest{}
	thisManifest.Manifest = *jsonParsed

	// Fill the placeholders for Start and Stop tests
	manifestID = thisManifest.Manifest.Search("id").Data().(string)
	log.Info(manifestID)
	version = thisManifest.Manifest.Search("version").Data().(string)
	log.Info(version)

	err = deploy.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***** DEPLOY SECOND IDENTICAL DATA SERVICE ********* //
	// Load Manifest JSON from file.
	log.Info("Loading second manifest...")

	json = LoadJSONBytes(manifestPath2)

	// Parse to gabs Container type
	jsonParsed, err = gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	var thisManifest2 = model.Manifest{}
	thisManifest2.Manifest = *jsonParsed

	// Fill the placeholders for Start and Stop tests
	manifestID2 = thisManifest2.Manifest.Search("id").Data().(string)
	log.Info(manifestID2)
	version2 = thisManifest2.Manifest.Search("version").Data().(string)
	log.Info(version2)

	err = deploy.DeployDataService(thisManifest2, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***** TEST UNDEPLOY FOR ORIGINAL DATA SERVICE ********* //

	// run tested method
	err = deploy.UndeployDataService(manifestID, version)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

	// check if containers are removed
	containers, _ := docker.ReadDataServiceContainers(manifestID, version)
	if len(containers) > 0 {
		t.Errorf("The following containers should have been removed: %v", containers)
	}

	// Check if the network is removed
	result, _ := deploy.DataServiceExist(manifestID, version)
	if result {
		t.Errorf("Network was not pruned (Data Service not removed)")
	}

	// ***** CHECK IF SECOND IDENTICAL DATA SERVICE STILL EXISTS ********* //
	expectedNumberContainers := len(thisManifest2.Manifest.S("services").Children())
	dsContainers, _ := docker.ReadDataServiceContainers(manifestID2, version2)

	if len(dsContainers) != expectedNumberContainers {
		t.Errorf("Some containers from the second identical network were removed.")
	}

	result2, _ := deploy.DataServiceExist(manifestID2, version2)
	if !result2 {
		t.Errorf("Second identical network is removed.")
	}

	// clean up and remove second data service
	err = deploy.UndeployDataService(manifestID2, version2)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}

}

func TestRedeployDataService(t *testing.T) {
	log.Info("TESTING REDEPLOYMENT...")

	// ***************** LOAD ORIGINAL MANIFEST AND DEPLOY DATA SERVICE ******************** //
	log.Info("Loading original manifest...")
	json := LoadJSONBytes(manifestPath)

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	var thisManifest = model.Manifest{}
	thisManifest.Manifest = *jsonParsed

	// Fill the placeholders for data service
	manifestID = thisManifest.Manifest.Search("id").Data().(string)
	log.Info(manifestID)
	version = thisManifest.Manifest.Search("version").Data().(string)
	log.Info(version)

	err = deploy.DeployDataService(thisManifest, "deploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***************** SAVE ORIGINAL DATA SERVICE TIMESTAMP AND ID ******************** //
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	filter := filters.NewArgs()
	filter.Add("label", "manifestID="+manifestID)
	filter.Add("label", "version="+version)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := cli.NetworkList(ctx, options)
	if err != nil {
		log.Error(err)
	}

	originalServiceTimestamp := networks[0].Created
	originalServiceID := networks[0].ID

	// ***************** LOAD ORIGINAL MANIFEST AND DEPLOY DATA SERVICE ******************** //
	log.Info("Loading redeployment manifest...")
	json = LoadJSONBytes(manifestPathRedeploy)

	// Parse to gabs Container type
	jsonParsed, err = gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	var thisManifestRedeploy = model.Manifest{}
	thisManifestRedeploy.Manifest = *jsonParsed

	err = deploy.DeployDataService(thisManifestRedeploy, "redeploy")
	if err != nil {
		t.Errorf("DeployDataService returned %v status", err)
	}

	// ***************** CHECK REDEPLOYMENT's SUCCESS ******************** //
	// compare new and old networks
	networksRe, err := cli.NetworkList(ctx, options)
	if err != nil {
		log.Info(err)
	}

	if originalServiceID == networksRe[0].ID || originalServiceTimestamp == networksRe[0].Created {
		t.Errorf("New network was not created.")
	}

	// ***************** CLEANING AFTER TESTING ******************** //
	log.Info("Cleaning after testing...")
	redeployedManifestID := thisManifestRedeploy.Manifest.Search("id").Data().(string)
	redeployedVersion := thisManifestRedeploy.Manifest.Search("version").Data().(string)
	err = deploy.UndeployDataService(redeployedManifestID, redeployedVersion)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}
}

// LoadJsonBytes reads file containts into byte[]
func LoadJSONBytes(manName string) []byte {
	manifestPath := path.Join(util.GetExeDir(), manName)

	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}
