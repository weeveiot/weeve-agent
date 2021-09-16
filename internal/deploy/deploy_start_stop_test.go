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

const manifestPath = "testdata/deploy_start_stop/deploy_manifest.json"

func TestDeployManifest(t *testing.T) {
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
	serviceID = strings.ReplaceAll(thisManifest.Manifest.Search("id").Data().(string), "-", "")
	log.Info(serviceID)
	serviceName = thisManifest.Manifest.Search("compose").Search("network").Search("name").Data().(string)
	log.Info(serviceName)

	// Get list of containers in a dataservice
	serviceContainerList := thisManifest.ContainerNamesList()

	resp := deploy.DeployManifest(thisManifest)

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
