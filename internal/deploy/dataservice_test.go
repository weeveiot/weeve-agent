/*
	These unit tests assume that there exist docker containers that
	can be started and stopped using the tested functions.
*/

package deploy_test

import (
	"context"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/deploy"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
)

var serviceID = "PLACEHOLDER"
var serviceName = "PLACEHOLDER"
var serviceID2 = "PLACEHOLDER"
var serviceName2 = "PLACEHOLDER"

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
	serviceID = strings.ReplaceAll(thisManifest.Manifest.Search("id").Data().(string), " ", "")
	serviceID = strings.ReplaceAll(serviceID, "-", "")

	log.Info(serviceID)
	serviceName = thisManifest.Manifest.Search("compose").Search("network").Search("name").Data().(string)
	log.Info(serviceName)

	// Get list of containers in a dataservice
	serviceContainerList := thisManifest.ContainerNamesList()

	resp := deploy.DeployManifest(thisManifest, "deploy")

	if resp != "SUCCESS" {
		t.Errorf("DeployManifest returned %v status", resp)
	}

	// Check if network exists
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	found := false
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}
	for _, network := range networks {
		if network.Name == serviceName {
			found = true

			networkID := network.ID

			networkDetails, err := cli.NetworkInspect(ctx, networkID, types.NetworkInspectOptions{})
			if err != nil {
				log.Error(err)
			}

			var networkContainers []string
			for _, container := range networkDetails.Containers {
				networkContainers = append(networkContainers, container.Name)
			}

			sort.Strings(serviceContainerList)
			sort.Strings(networkContainers)

			for i, element := range serviceContainerList {
				if element != networkContainers[i] {
					t.Errorf("Container %v does not belong to the network", element)
				}
			}
		}
	}

	if !found {
		t.Errorf("Network not found.")
	}

}

// Test Stop Service method
func TestStopDataServiceWrongDetails(t *testing.T) {
	// IMPORTANT: Assume all containers are exited at the beginning

	log.Info("TESTING STOP DATA SERVICE WITH WRONG DETAILS...")

	var wrongServiceID = serviceID + "WRONG"
	var wrongServiceName = serviceName + "WRONG"
	var wrongStatusContainerList []string

	// check container status before executing tested function
	statusBefore := make(map[string]string)
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				statusBefore[name[1:]] = container.State
			}
		}
	}

	// run tested method
	deploy.StopDataService(wrongServiceID, wrongServiceName)

	// check container status after executing tested function
	containers = docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if container.State != statusBefore[name[1:]] {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
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
	deploy.StopDataService(serviceID, serviceName)

	// check container status
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if container.State != "exited" {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
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

	var wrongServiceID = serviceID + "WRONG"
	var wrongServiceName = serviceName + "WRONG"
	var wrongStatusContainerList []string

	// check container status before executing tested function
	statusBefore := make(map[string]string)
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				statusBefore[name[1:]] = container.State
			}
		}
	}

	// run tested method
	deploy.StartDataService(wrongServiceID, wrongServiceName)

	// check container status after executing tested function
	containers = docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if container.State != statusBefore[name[1:]] {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
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
	deploy.StartDataService(serviceID, serviceName)

	// check container status
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if container.State != "running" {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers SHOULD be 'running': %v", wrongStatusContainerList)
	}
}

func TestUndeployDataService(t *testing.T) {
	log.Info("TESTING UNDEPLOYMENT...")

	// run tested method
	deploy.UndeployDataService(serviceID, serviceName)

	// check if containers are removed
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				t.Errorf("The following container should have been removed: %v", name)
			}
		}
	}

	// Check if the network is removed
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}
	for _, network := range networks {
		if network.Name == serviceName {
			t.Errorf("Network %v was not pruned (Data Service not removed)", serviceName)
		}
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
	serviceID = strings.ReplaceAll(thisManifest.Manifest.Search("id").Data().(string), " ", "")
	serviceID = strings.ReplaceAll(serviceID, "-", "")

	log.Info(serviceID)
	serviceName = thisManifest.Manifest.Search("compose").Search("network").Search("name").Data().(string)
	log.Info(serviceName)

	resp := deploy.DeployManifest(thisManifest, "deploy")

	if resp != "SUCCESS" {
		t.Errorf("DeployManifest returned %v status", resp)
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
	serviceID2 = strings.ReplaceAll(thisManifest2.Manifest.Search("id").Data().(string), " ", "")
	serviceID2 = strings.ReplaceAll(serviceID2, "-", "")

	log.Info(serviceID2)
	serviceName2 = thisManifest2.Manifest.Search("compose").Search("network").Search("name").Data().(string)
	log.Info(serviceName2)

	resp = deploy.DeployManifest(thisManifest2, "deploy")

	if resp != "SUCCESS" {
		t.Errorf("DeployManifest returned %v status", resp)
	}

	// ***** TEST UNDEPLOY FOR ORIGINAL DATA SERVICE ********* //

	// run tested method
	deploy.UndeployDataService(serviceID, serviceName)

	// check if containers are removed
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				t.Errorf("The following container should have been removed: %v", name)
			}
		}
	}

	// Check if the network is removed
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}
	for _, network := range networks {
		if network.Name == serviceName {
			t.Errorf("Network %v was not pruned (Data Service not removed)", serviceName)
		}
	}

	// ***** CHECK IF SECOND IDENTICAL DATA SERVICE STILL EXISTS ********* //
	expectedNumberContainers := len(thisManifest2.Manifest.Search("compose").S("services").Children())
	containersCount := 0
	for _, container := range docker.ReadAllContainers() {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID2) {
				containersCount++
			}
		}
	}
	if containersCount != expectedNumberContainers {
		t.Errorf("Some containers from the second identical network were removed.")
	}

	secondNetworkExists := false
	networks, err = cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}
	for _, network := range networks {
		if network.Name == serviceName2 {
			secondNetworkExists = true
		}
	}
	if !secondNetworkExists {
		t.Errorf("Second identical network is removed.")
	}

	// clean up and remove second data service
	deploy.UndeployDataService(serviceID2, serviceName2)

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

	originalServiceName := thisManifest.Manifest.Search("compose").Search("network").Search("name").Data().(string)

	resp := deploy.DeployManifest(thisManifest, "deploy")
	if resp != "SUCCESS" {
		t.Errorf("DeployManifest returned %v status", resp)
	}

	// ***************** SAVE ORIGINAL DATA SERVICE TIMESTAMP AND ID ******************** //
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Error(err)
	}

	originalServiceTimestamp := time.Now()
	originalServiceMachineID := "placeholder"
	for _, network := range networks {
		if network.Name == originalServiceName {
			originalServiceTimestamp = network.Created
			originalServiceMachineID = network.ID
		}
	}

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

	redeployedServiceName := thisManifestRedeploy.Manifest.Search("compose").Search("network").Search("name").Data().(string)

	resp = deploy.DeployManifest(thisManifestRedeploy, "redeploy")
	if resp != "SUCCESS" {
		t.Errorf("DeployManifest returned %v status", resp)
	}

	// ***************** CHECK REDEPLOYMENT's SUCCESS ******************** //
	// compare new and old networks
	networks, err = cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Info(err)
	}

	for _, network := range networks {
		if network.Name == redeployedServiceName {
			if redeployedServiceName != originalServiceName {
				t.Errorf("Wrong networks names. Old network: %v. New network: %v", originalServiceName, redeployedServiceName)
			} else if originalServiceMachineID == network.ID || originalServiceTimestamp == network.Created {
				t.Errorf("New network was not created.")
			}
		}
	}

	// ***************** CLEANING AFTER TESTING ******************** //
	log.Info("Cleaning after testing...")
	redeployedServiceID := strings.ReplaceAll(thisManifestRedeploy.Manifest.Search("id").Data().(string), " ", "")
	redeployedServiceID = strings.ReplaceAll(redeployedServiceID, "-", "")
	deploy.UndeployDataService(redeployedServiceID, redeployedServiceName)
}

// LoadJsonBytes reads file containts into byte[]
func LoadJSONBytes(manName string) []byte {

	_, b, _, _ := runtime.Caller(0)
	// Root folder of this project
	Root := filepath.Join(filepath.Dir(b), "../..")
	manifestPath := path.Join(Root, manName)

	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}
