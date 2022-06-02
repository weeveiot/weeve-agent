package handler_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
)

const localManifest = "../../testdata/manifest/test_manifest.json"

func init() {
	docker.SetupDockerClient()
}

func TestReadDeployManifestLocalPass(t *testing.T) {
	err := handler.ReadDeployManifestLocal(localManifest)
	if err != nil {
		t.Error("Deployment of the local manifest expected to succeed, but deployment failed! CAUSE --> ", err)
	}

	// Cleanup
	json, err := ioutil.ReadFile(localManifest)
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

	err = dataservice.UndeployDataService(thisManifest.ID, thisManifest.VersionName)
	if err != nil {
		t.Error("UndeployDataService returned error: ", err)
	}
}

func TestReadDeployManifestLocalWrongPath(t *testing.T) {
	docker.SetupDockerClient()
	err := handler.ReadDeployManifestLocal("fest/test_manifest.json")
	e, ok := err.(*os.PathError)
	if e != nil && !ok {
		t.Error("Expected PathError! But exception received -> ", err)
	}
}
