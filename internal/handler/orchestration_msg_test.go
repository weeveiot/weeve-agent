package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"errors"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/edgeapp"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func init() {
	docker.SetupDockerClient()
}

type msgType struct {
	ManifestName string `json:"manifestName"`
	UpdatedAt    string `json:"updatedAt"`
	Command      string `json:"command"`
}

var ctx = context.Background()
var dockerCli *client.Client

func TestProcessMessagePass(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	opt := model.Params{
		Broker:    "mqtt://test.mosquitto.org:1883",
		NoTLS:     true,
		Heartbeat: 60,
		NodeId:    "1234567890",
		NodeName:  "Test Node",
	}
	config.Set(opt)
	com.ConnectNode(map[string]mqtt.MessageHandler{})

	assert := assert.New(t)
	// Prepare test data
	manifestPath := "../../testdata/test_manifest.json"
	msg, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(msg)
	if err != nil {
		t.Fatal(err)
	}

	man, err := parseManifest(jsonParsed)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath2 := "../../testdata/test_manifest2.json"
	msg2, err := os.ReadFile(manifestPath2)
	if err != nil {
		t.Fatal(err)
	}

	jsonParsed2, err := gabs.ParseJSON(msg2)
	if err != nil {
		t.Fatal(err)
	}

	man2, err := parseManifest(jsonParsed2)
	if err != nil {
		t.Fatal(err)
	}

	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("TESTING EDGE APPLICATION DEPLOYMENT...")
	err = deployEdgeApplication(msg, man)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("TESTING STOP EDGE APPLICATION...")
	err = stopEdgeApplication(man)
	if err != nil {
		t.Error(err)
		err = undeployEdgeApplication(man, edgeapp.CMDRemove)
		if err != nil {
			t.Fatal(err)
		}
		t.FailNow()
	}

	fmt.Println("TESTING RESUME EDGE APPLICATION...")
	err = resumeEdgeApplication(man)
	if err != nil {
		t.Error(err)
		err = undeployEdgeApplication(man, edgeapp.CMDRemove)
		if err != nil {
			t.Fatal(err)
		}
		t.FailNow()
	}

	fmt.Println("TESTING EDGE APPLICATION REDEPLOYMENT...")
	err = deployEdgeApplication(msg2, man2)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("TESTING UNDEPLOY EDGE APPLICATION...")
	err = undeployEdgeApplication(man2, edgeapp.CMDUndeploy)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("DEPLOYING EDGE APPLICATION FOR TESTING REMOVE EDGE APPLICATION...")
	err = deployEdgeApplication(msg2, man2)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("TESTING REMOVE EDGE APPLICATION...")
	err = undeployEdgeApplication(man2, edgeapp.CMDRemove)
	assert.Nil(err)
}

func deployEdgeApplication(jsonBytes []byte, man manifest.Manifest) error {
	// Process deploy edge application
	err := handler.ProcessOrchestrationMessage(jsonBytes)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	// Verify deployment
	net, err := getNetwork(man.ManifestUniqueID)
	if err != nil {
		return err
	}

	if len(net) > 0 {
		_, err := checkContainersExistsWithStatus(man.ManifestUniqueID, len(man.Modules), "running")
		if err != nil {
			return err
		}
	} else {
		return errors.New("Network not created")
	}

	return nil
}

func stopEdgeApplication(man manifest.Manifest) error {
	msg := msgType{
		ManifestName: man.ManifestUniqueID.ManifestName,
		UpdatedAt:    man.ManifestUniqueID.UpdatedAt,
		Command:      edgeapp.CMDStopService,
	}
	jsonB, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessOrchestrationMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = checkContainersExistsWithStatus(man.ManifestUniqueID, len(man.Modules), "exited")
	if err != nil {
		return err
	}

	return nil
}

func resumeEdgeApplication(man manifest.Manifest) error {
	msg := msgType{
		ManifestName: man.ManifestUniqueID.ManifestName,
		UpdatedAt:    man.ManifestUniqueID.UpdatedAt,
		Command:      edgeapp.CMDResumeService,
	}
	jsonB, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = handler.ProcessOrchestrationMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = checkContainersExistsWithStatus(man.ManifestUniqueID, len(man.Modules), "running")
	if err != nil {
		return err
	}

	return nil
}

func undeployEdgeApplication(man manifest.Manifest, operation string) error {
	msg := msgType{
		ManifestName: man.ManifestUniqueID.ManifestName,
		UpdatedAt:    man.ManifestUniqueID.UpdatedAt,
	}
	if operation == edgeapp.CMDUndeploy || operation == edgeapp.CMDRemove {
		msg.Command = operation
	} else {
		return errors.New("Invalid operation: " + operation)
	}
	jsonB, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = handler.ProcessOrchestrationMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	net, err := getNetwork(man.ManifestUniqueID)
	if err != nil {
		return err
	}

	if len(net) <= 0 {
		dsContainers, _ := getEdgeApplicationContainers(man.ManifestUniqueID)
		if len(dsContainers) > 0 {
			return errors.New("Edge application undeployment failed, containers not deleted")
		}
	} else {
		return errors.New("Edge application undeployment failed, network not deleted")
	}

	if operation == edgeapp.CMDUndeploy {
		exist, err := checkImages(man, true)
		if err != nil {
			return err
		}

		if !exist {
			return errors.New("Edge application undeploy should not delete images")
		}
	} else {
		deleted, err := checkImages(man, false)
		if err != nil {
			return err
		}

		if !deleted {
			return errors.New("Edge application removal should delete images")
		}
	}

	return nil
}

func parseManifest(jsonParsed *gabs.Container) (manifest.Manifest, error) {
	manifestName := jsonParsed.Search("manifestName").Data().(string)
	updatedAt := jsonParsed.Search("updatedAt").Data().(string)

	var containerConfigs []manifest.ContainerConfig

	modules := jsonParsed.Search("modules").Children()
	for _, module := range modules {
		var containerConfig manifest.ContainerConfig

		imageName := module.Search("image").Search("name").Data().(string)
		imageTag := module.Search("image").Search("tag").Data().(string)

		if imageTag == "" {
			containerConfig.ImageNameFull = imageName
		} else {
			containerConfig.ImageNameFull = imageName + ":" + imageTag
		}

		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := manifest.Manifest{
		ManifestUniqueID: model.ManifestUniqueID{ManifestName: manifestName, UpdatedAt: updatedAt},
		Modules:          containerConfigs,
	}

	return manifest, nil
}

func getNetwork(manID model.ManifestUniqueID) ([]types.NetworkResource, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manID.ManifestName)
	filter.Add("label", "updatedAt="+manID.UpdatedAt)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := dockerCli.NetworkList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return networks, nil
}

func getEdgeApplicationContainers(manifestUniqueID model.ManifestUniqueID) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manifestUniqueID.ManifestName)
	filter.Add("label", "updatedAt="+manifestUniqueID.UpdatedAt)
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := dockerCli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func checkContainersExistsWithStatus(manID model.ManifestUniqueID, containerCount int, status string) (bool, error) {
	dsContainers, _ := getEdgeApplicationContainers(manID)
	if containerCount != len(dsContainers) {
		return false, fmt.Errorf("Expected number of containers %v, number of available containers %v", containerCount, len(dsContainers))
	}
	for _, dsContainer := range dsContainers {
		if dsContainer.State != strings.ToLower(status) {
			return false, fmt.Errorf("Container expected status %s, but current status %s", status, dsContainer.State)
		}
	}

	return true, nil
}

func checkImages(man manifest.Manifest, exist bool) (bool, error) {

	for _, module := range man.Modules {
		_, _, err := dockerCli.ImageInspectWithRaw(ctx, module.ImageNameFull)
		if err != nil {
			if client.IsErrNotFound(err) {
				if exist {
					return false, nil
				}
			} else {
				return false, err
			}
		} else if !exist {
			return false, err
		}
	}

	return true, nil
}
