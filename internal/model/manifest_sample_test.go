package model

import (
	"os"
	"testing"

	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)

var SampleManifestBytesMVP []byte

func TestLoad(m *testing.M) {

	manifestBytesMVP = LoadJsonBytes("pipeline_unit/workingMVP.json")
	code := m.Run()

	os.Exit(code)
}
