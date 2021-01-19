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
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
)

var apiURL = "http://localhost:8030/pipelines"

func TestPostPipeline(t *testing.T) {
	logrus.Debug("Running test Pipeline POST")
	filePath := "testdata/newFormat020/workingMVP.json"
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

func TestImageNotFound(t *testing.T) {
	filePath := "testdata/newFormat020/failImageNotFound.json"
	json := LoadJSONBytes(filePath)

	req := httptest.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer([]byte(json)))
	res := httptest.NewRecorder()

	POSTpipelines(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("got status %d but wanted %d", res.Code, http.StatusNotFound)
	}

	logrus.Debug("Called post pipeline")
}

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
