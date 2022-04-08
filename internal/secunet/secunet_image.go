//go:build !secunet

package secunet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

var existingImagesIdToName = make(map[string]string)
var existingImagesNameToId = make(map[string]string)

func getImageID(name, tag string) string {
	fullName := name + tag
	return existingImagesNameToId[fullName]
}

func getAuthToken(imageName string) (string, error) {
	commandUrl := fmt.Sprintf("%s/token?service=%s&scope=repository:library/%s:pull", authUrl, svcUrl, imageName)
	resp, err := http.Get(commandUrl)
	if err != nil {
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		var resp_json map[string]string
		json.Unmarshal(body, &resp_json)

		return resp_json["token"], nil
	} else {
		err = errors.New(fmt.Sprintf("PullImage: Could not get the authentication token. HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return "", err
	}
}

// WIP!!!
func getManifest(token, imageName, digest string) ([]byte, error) {
	if digest == "" {
		digest = "latest"
	}
	commandUrl := fmt.Sprintf("%s/v2/library/%s/manifests/%s", registryUrl, imageName, digest)
	req, err := http.NewRequest(http.MethodGet, commandUrl, nil)
	if err != nil {
		log.Error(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		return body, nil
	} else {
		err = errors.New(fmt.Sprintf("getManifest: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return nil, err
	}
}

// WIP!!!
func PullImage(imgDetails model.RegistryDetails) (string, error) {
	// TODO: transform to HTTPS

	// token, err := getAuthToken(imgDetails.ImageName)
	// if err != nil {
	// 	log.Error(err)
	// }

	// getManifest(token, imgDetails.ImageName, "")

	// INTERIM SOLUTION
	downloadScriptName := "download-frozen-image-v2.sh"
	archiveScriptName := "archive.sh"
	nameWithoutTag := strings.Split(imgDetails.ImageName, ":")[0] // extract the image name w/o tag
	fileName := nameWithoutTag + ".tar.gz"
	cmd := exec.Command("./"+downloadScriptName, nameWithoutTag, imgDetails.ImageName)
	out, err := cmd.CombinedOutput()

	fmt.Println(string(out))
	if err != nil {
		log.Error(err)
		return "", err
	}

	cmd = exec.Command("./"+archiveScriptName, nameWithoutTag)
	err = cmd.Run()
	if err != nil {
		log.Error(err)
		return "", err
	}

	// New multipart writer.
	bufferedFile := &bytes.Buffer{}
	writer := multipart.NewWriter(bufferedFile)
	fw, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		log.Error(err)
		return "", err
	}
	fd, err := os.Open(fileName)
	if err != nil {
		log.Error(err)
		return "", err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Error(err)
		return "", err
	}
	writer.Close()

	commandUrl := "/docker/images"
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, bytes.NewReader(bufferedFile.Bytes()))
	if err != nil {
		log.Error(err)
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// username := "Admin"
	// password := "Secure Visibility"
	// req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		var resp_json map[string]string
		json.Unmarshal(body, &resp_json)
		imageID := resp_json["id"]

		// add image to local database
		existingImagesIdToName[imageID] = nameWithoutTag
		existingImagesNameToId[nameWithoutTag] = imageID

		return imageID, nil
	} else {
		err = errors.New(fmt.Sprintf("PullImage: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return "", err
	}
}

func ImageExists(id string) (bool, error) {
	if existingImagesIdToName[id] == "" {
		return false, nil
	} else {
		return true, nil
	}
}

func ImageRemove(imageID string) error {
	commandUrl := fmt.Sprintf("/docker/images/%s", imageID)
	req, err := http.NewRequest(http.MethodDelete, edgeUrl+commandUrl, nil)
	if err != nil {
		log.Error(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		// remove image from local database
		fullName := existingImagesIdToName[imageID]
		delete(existingImagesNameToId, fullName)
		delete(existingImagesIdToName, imageID)

		log.Debug("Removed image ID ", imageID)
	} else {
		err = errors.New(fmt.Sprintf("ImageRemove: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
	}

	return nil
}
