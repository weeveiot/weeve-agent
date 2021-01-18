package controller

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
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

	logrus.Debug("Called post pipeline")
}

func TestImageNotFound(t *testing.T) {
	logrus.Debug("Running test Pipeline POST")
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
