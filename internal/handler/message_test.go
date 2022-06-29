package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func init() {
	docker.SetupDockerClient()
}

func TestProcessMessagePass(t *testing.T) {
	log.Info("TESTING DEPLOYMENT...")

	var manCmd struct {
		ManifestName  string `json:"manifestName"`
		VersionNumber string `json:"versionNumber"`
		Command       string `json:"command"`
	}

	// 1 Prepare test data
	manifestPath := "../../testdata/manifest/mvp-manifest-deploy.json"
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	jsonParsed, err := gabs.ParseJSON(jsonBytes)
	if err != nil {
		t.Error(err)
	}

	man, err := ParseManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	// you data
	manCmd.ManifestName = man.ManifestUniqueID.ManifestName
	manCmd.VersionNumber = man.ManifestUniqueID.VersionNumber

	// 2 Process deploy manifest
	err = handler.ProcessMessage(jsonBytes)
	if err != nil {
		t.Errorf("ProcessMessage returned %v status", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Error(err)
	}

	// 3 Verify deployment
	exist, err := CheckNetworkExist(man.ManifestUniqueID, cli)
	if err != nil {
		t.Error(err)
	}

	if exist {
		_, err := CheckContainersWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Running, cli)
		if err != nil {
			t.Error(err)
		}
	} else {
		t.Error("Network not created")
	}

	// JSON encoding
	manCmd.Command = dataservice.CMDStopService
	jsonB, err := json.Marshal(manCmd)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		t.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = CheckContainersWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Paused, cli)
	if err != nil {
		t.Error(err)
	}

	// JSON encoding
	manCmd.Command = dataservice.CMDStartService
	jsonB, err = json.Marshal(manCmd)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		t.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = CheckContainersWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Running, cli)
	if err != nil {
		t.Error(err)
	}

	// JSON encoding
	manCmd.Command = dataservice.CMDUndeploy
	jsonB, err = json.Marshal(manCmd)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		t.Errorf("ProcessMessage returned %v status", err)
	}

	exist, err = CheckNetworkExist(man.ManifestUniqueID, cli)
	if err != nil {
		t.Error(err)
	}

	if !exist {
		dsContainers, _ := docker.ReadDataServiceContainers(man.ManifestUniqueID)
		if len(dsContainers) > 0 {
			t.Error("Edge application undeployment failed")
		}
	}
}

func TestReadDeployManifestLocalPass(t *testing.T) {
	msg, err := handler.GetStatusMessage()
	if err != nil {
		t.Error("Expected status message, but got error! CAUSE --> ", err)
	}

	assert.Nil(t, msg)
	assert.NotEqual(t, nil, msg)
}

func ParseManifest(jsonParsed *gabs.Container) (manifest.Manifest, error) {
	manifestName := jsonParsed.Search("manifestName").Data().(string)
	versionNumber := jsonParsed.Search("versionNumber").Data().(float64)
	command := jsonParsed.Search("command").Data().(string)

	var containerConfigs []manifest.ContainerConfig

	modules := jsonParsed.Search("modules").Children()
	for _, module := range modules {
		var containerConfig manifest.ContainerConfig

		containerConfig.ImageName = module.Search("image").Search("name").Data().(string)
		containerConfig.ImageTag = module.Search("image").Search("tag").Data().(string)

		imageName := containerConfig.ImageName
		if containerConfig.ImageTag != "" {
			imageName = imageName + ":" + containerConfig.ImageTag
		}

		containerConfig.Registry = manifest.RegistryDetails{ImageName: imageName}
		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := manifest.Manifest{
		ManifestUniqueID: model.ManifestUniqueID{ManifestName: manifestName, VersionNumber: fmt.Sprint(versionNumber)},
		VersionNumber:    versionNumber,
		Modules:          containerConfigs,
		Command:          command,
	}

	return manifest, nil
}

func CheckNetworkExist(manID model.ManifestUniqueID, cli *client.Client) (bool, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manID.ManifestName)
	filter.Add("label", "versionNumber="+manID.VersionNumber)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := cli.NetworkList(context.Background(), options)
	if err != nil {
		return false, err
	}

	return len(networks) > 0, nil
}

func CheckContainersWithStatus(manID model.ManifestUniqueID, containerCount int, status string, cli *client.Client) (bool, error) {
	dsContainers, _ := docker.ReadDataServiceContainers(manID)
	if containerCount > len(dsContainers) {
		return false, fmt.Errorf("Expected number of containers %v, number of available containers %v", containerCount, len(dsContainers))
	}
	for _, dsContainer := range dsContainers {
		if dsContainer.State != status {
			return false, fmt.Errorf("Container expected status %s, but current status %s", status, dsContainer.State)
		}
	}

	return true, nil
}
