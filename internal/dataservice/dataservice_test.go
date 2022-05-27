/*
	These unit tests assume that there exist docker containers that
	can be started and stopped using the tested functions.
*/

package dataservice_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

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
var originalServiceTimestamp time.Time
var originalServiceID string

const manifestPath = "../../testdata/manifest/test_manifest.json"

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
	version = thisManifest.Version
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
		originalServiceTimestamp = networks[0].Created
		originalServiceID = networks[0].ID

		// Check if containers exist
		dsContainers, _ := docker.ReadDataServiceContainers(manifestID, version)
		containers := thisManifest.Modules
		for _, dsContainer := range dsContainers {
			containersExist := false
			for _, dsContainerName := range dsContainer.Names {
				for _, container := range containers {
					if dsContainerName == "/"+container.ContainerName {
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

func TestRedeployDataService(t *testing.T) {
	log.Info("TESTING REDEPLOYMENT...")

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

	// ***************** LOAD ORIGINAL MANIFEST AND DEPLOY DATA SERVICE ******************** //
	log.Info("Loading redeployment manifest...")
	json, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		t.Error("Error on parsing message: ", err)
	}

	thisManifestRedeploy, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

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
		t.Error("New network was not created.")
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
