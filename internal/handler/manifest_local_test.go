package handler_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

const localManifest = "/testdata/manifest/test_manifest.json"

func TestReadDeployManifestLocalPass(t *testing.T) {
	var (
		_, b, _, _ = runtime.Caller(0)

		// Root folder of this project
		Root = filepath.Join(filepath.Dir(b), "../..")
	)
	filePath := Root + localManifest

	docker.SetupDockerClient()
	err := handler.ReadDeployManifestLocal(filePath)
	if err != nil {
		t.Error("Deployment of the local manifest expected to succeed, but deployment failed! CAUSE --> ", err)
	}

	// Cleanup
	json := ioutility.LoadJsonBytes(filePath)

	// Parse to gabs Container type
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		log.Info("Error on parsing message: ", err)
	}

	thisManifest, err := manifest.GetManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	err = dataservice.UndeployDataService(thisManifest.ID, thisManifest.Version)
	if err != nil {
		t.Errorf("UndeployDataService returned %v status", err)
	}
}

func TestReadDeployManifestLocalWrongPath(t *testing.T) {
	docker.SetupDockerClient()
	err := handler.ReadDeployManifestLocal(localManifest)
	if !strings.Contains(err.Error(), "The system cannot find the path specified.") {
		t.Error("Expected The system cannot find the path specified exception! But exception received -> ", err)
	}
}
