package controller

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

var apiURL = "http://localhost:8030/pipelines"

/* SUCCESS TEST
Create pipeline with 2 containers
Checck container status is Up or Exited
*/
func TestPostPipeline(t *testing.T) {
	filePath := "testdata/pipeline_integration_public/workingMVP.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusTeapot)
	} else {
		logrus.Info("Containers created wait 15 seconds before checking container status")
		time.Sleep(15 * time.Second)

		// Get all containers
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Error(err)
		}
		options := types.ContainerListOptions{All: true}
		containers, err := dockerClient.ContainerList(context.Background(), options)
		if err != nil {
			t.Error(err)
		}

		// Parse manifest
		man, er := model.ParseJSONManifest(json)
		if err != nil {
			t.Error(er)
		}

		// Check all containers status Up or Exited
		for _, containerName := range man.ContainerNamesList() {
			for _, container := range containers {
				findContainer := util.StringArrayContains(container.Names, "/"+containerName)
				if findContainer {
					if !strings.Contains(container.Status, "Up ") && !strings.Contains(container.Status, "Exited ") {
						t.Error("Container: " + containerName + " failed to start with status Status: " + container.Status)
					}
				}
			}
		}
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json)
}

/* SUCCESS TEST
Create pipeline with 3 inter connected containers
Checck container status is Up
*/
func TestInterCommunication(t *testing.T) {
	filePath := "testdata/pipeline_integration_public/workingInterCommunicationMVP.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusTeapot)
	} else {
		logrus.Info("Containers created wait 15 seconds before checking container status")
		time.Sleep(15 * time.Second)

		// Get all containers
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Error(err)
		}
		options := types.ContainerListOptions{All: true}
		containers, err := dockerClient.ContainerList(context.Background(), options)
		if err != nil {
			t.Error(err)
		}

		// Parse manifest
		man, er := model.ParseJSONManifest(json)
		if err != nil {
			t.Error(er)
		}

		// Check all containers status Up or Exited
		for _, containerName := range man.ContainerNamesList() {
			for _, container := range containers {
				findContainer := util.StringArrayContains(container.Names, "/"+containerName)
				if findContainer {
					if !strings.Contains(container.Status, "Up ") {
						t.Error("Container: " + containerName + " failed to start with status Status: " + container.Status)
					}
				}
			}
		}
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json)
}

/* SUCCESS TEST
Pull image from private docker registry using UserName and Password
*/
func TestPullPrivateDockerRegistryImage(t *testing.T) {
	filePath := "testdata/pipeline_integration_private/pullImageWithAuth.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusTeapot)
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json)
}

/* SUCCESS TEST
Don't pull image from docker hub if it's available in local
Recreate containers with new configuration (port changed from 2000 to 2020) if conainer already exists
*/
func TestContinerRemoveCreate(t *testing.T) {
	filePath1 := "testdata/pipeline_integration_public/recreateContainer1.json"
	json1 := LoadJSONBytes(filePath1)

	req1 := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json1)))
	res1 := httptest.NewRecorder()

	// Step-1: This will create containers
	POSTpipelines(res1, req1)

	if res1.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res1.Code, http.StatusTeapot)
	}

	filePath2 := "testdata/pipeline_integration_public/recreateContainer2.json"
	json2 := LoadJSONBytes(filePath2)

	req2 := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json2)))
	res2 := httptest.NewRecorder()

	// Step-2: It will remove containers created in Step-1 and creates again
	POSTpipelines(res2, req2)

	if res2.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res2.Code, http.StatusTeapot)
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json2)
}

/* FAIL TEST
Return 404 Status (Not found) if image from manifest is not exist in docker registry
*/
func TestImageNotFound(t *testing.T) {
	filePath := "testdata/pipeline_integration_public/failImageNotFound.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusNotFound)
	}

	logrus.Debug("Called post pipeline")
}

/* FAIL TEST
Return 404 Status (Not found) if any one of image from manifest is not exist in docker registry
*/
func TestPartialImagesPull(t *testing.T) {
	filePath := "testdata/pipeline_integration_public/failPartialImagesAvail.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusNotFound)
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json)
}

/* FAIL TEST
Create pipeline for one container with incorrect parameters
Checck container status is not Up or Exited
*/
func TestContainerWithIncorrectParameter(t *testing.T) {
	filePath := "testdata/pipeline_integration_private/failIncorrectParameters.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusTeapot)
	} else {
		logrus.Info("Containers created wait 15 seconds before checking container status")
		time.Sleep(15 * time.Second)

		// Get all containers
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Error(err)
		}
		options := types.ContainerListOptions{All: true}
		containers, err := dockerClient.ContainerList(context.Background(), options)
		if err != nil {
			t.Error(err)
		}

		// Parse manifest
		man, er := model.ParseJSONManifest(json)
		if err != nil {
			t.Error(er)
		}

		// Check all containers status Up or Exited
		for _, containerName := range man.ContainerNamesList() {
			for _, container := range containers {
				findContainer := util.StringArrayContains(container.Names, "/"+containerName)
				if findContainer {
					if !strings.Contains(container.Status, "Restarting ") {
						t.Error("Container: " + containerName + " expected Status is Restarting but received Status: " + container.Status)
					}
				}
			}
		}
	}

	// Cleanup resources creaetd by test
	CleanDockerResources(json)
}

// CleanDockerResources cleans all docker resources as per input manifest
func CleanDockerResources(manifest []byte) {
	logrus.Info("Cleaning docker resources")

	man, err := model.ParseJSONManifest(manifest)
	if err != nil {
		log.Printf("Unable to stop container: %s", err)
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Error(err)
	}

	// Delete containers
	for _, containerName := range man.ContainerNamesList() {
		if err := cli.ContainerStop(ctx, containerName, nil); err != nil {
			log.Printf("Unable to stop container %s: %s", containerName, err)
		}

		removeOptions := types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}

		if err := cli.ContainerRemove(ctx, containerName, removeOptions); err != nil {
			log.Printf("Unable to remove container: %s", err)
		}
	}

	// Delete images
	for _, imgName := range man.ImageNamesList() {
		removeOptions := types.ImageRemoveOptions{
			Force: true,
		}

		if _, err := cli.ImageRemove(ctx, imgName, removeOptions); err != nil {
			log.Printf("Unable to remove image: %s", err)
		}
	}

	// Delete network
	networkName := man.GetNetworkName()
	errN := cli.NetworkRemove(ctx, networkName)
	if errN != nil {
		log.Printf("Unable to remove image: %s", errN)
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
