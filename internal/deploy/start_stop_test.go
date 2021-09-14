/*
	These unit tests assume that there exist docker containers that
	can be started and stopped using the tested functions.
*/

package deploy

import (
	"strings"
	"testing"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
)

var serviceID = "kuba_test_go"
var serviceName = "test"

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
				statusBefore[name[1:]] = docker.ContainerStatus(container.ID)
			}
		}
	}

	// run tested method
	StartDataService(wrongServiceID, wrongServiceName)

	// check container status after executing tested function
	containers = docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if docker.ContainerStatus(container.ID) != statusBefore[name[1:]] {
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
	StartDataService(serviceID, serviceName)

	// check container status
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				containerStatus := docker.ContainerStatus(container.ID)
				if containerStatus != "running" {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers SHOULD be 'running': %v", wrongStatusContainerList)
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
				statusBefore[name[1:]] = docker.ContainerStatus(container.ID)
			}
		}
	}

	// run tested method
	StopDataService(wrongServiceID, wrongServiceName)

	// check container status after executing tested function
	containers = docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				if docker.ContainerStatus(container.ID) != statusBefore[name[1:]] {
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
	StopDataService(serviceID, serviceName)

	// check container status
	containers := docker.ReadAllContainers()
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name[1:], serviceID) {
				containerStatus := docker.ContainerStatus(container.ID)
				if containerStatus != "exited" {
					wrongStatusContainerList = append(wrongStatusContainerList, name[1:])
				}
			}
		}
	}
	if len(wrongStatusContainerList) > 0 {
		t.Errorf("The following containers SHOULD be 'exited': %v", wrongStatusContainerList)
	}
}
