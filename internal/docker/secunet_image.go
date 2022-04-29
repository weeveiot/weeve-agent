//go:build secunet

package docker

import (
	"bytes"
	"encoding/json"
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

var existingImagesNameToId = make(map[string]string)

func getAuthToken(imageName string) (string, error) {
	commandUrl := fmt.Sprintf("%s/token?service=%s&scope=repository:library/%s:pull", authUrl, svcUrl, imageName)
	resp, err := http.Get(commandUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("PullImage: Could not get the authentication token. HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return "", err
	}

	var resp_json map[string]string
	json.Unmarshal(body, &resp_json)

	return resp_json["token"], nil
}

// WIP!!!
func getManifest(token, imageName, digest string) ([]byte, error) {
	if digest == "" {
		digest = "latest"
	}
	commandUrl := fmt.Sprintf("%s/v2/library/%s/manifests/%s", registryUrl, imageName, digest)
	req, err := http.NewRequest(http.MethodGet, commandUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("getManifest: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return nil, err
	}

	return body, nil
}

func getNameAndTag(fullImageName string) (string, string) {
	var nameWithoutTag, tag string
	imageNameParts := strings.Split(fullImageName, ":")
	nameWithoutTag = imageNameParts[0]
	if len(imageNameParts) == 2 {
		tag = imageNameParts[1]
	}

	return nameWithoutTag, tag
}

// WIP!!!
func PullImage(imgDetails model.RegistryDetails) error {
	// TODO: transform to HTTPS

	// token, err := getAuthToken(imgDetails.ImageName)
	// if err != nil {
	// 	return err
	// }

	// getManifest(token, imgDetails.ImageName, "")

	// INTERIM SOLUTION
	const downloadScriptName = "download-frozen-image-v2.sh"
	const archiveScriptName = "archive.sh"
	nameWithoutTag, _ := getNameAndTag(imgDetails.ImageName)
	fileName := nameWithoutTag + ".tar.gz"
	cmd := exec.Command("./"+downloadScriptName, nameWithoutTag, imgDetails.ImageName)
	out, err := cmd.CombinedOutput()

	fmt.Println(string(out))
	if err != nil {
		return err
	}

	cmd = exec.Command("./"+archiveScriptName, nameWithoutTag)
	err = cmd.Run()
	if err != nil {
		return err
	}

	// New multipart writer.
	bufferedFile := &bytes.Buffer{}
	writer := multipart.NewWriter(bufferedFile)
	fw, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	fd, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		return err
	}
	writer.Close()

	commandUrl := "/docker/images"
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, bytes.NewReader(bufferedFile.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("PullImage: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return err
	}

	var resp_json map[string]string
	json.Unmarshal(body, &resp_json)
	imageID := resp_json["id"]

	// add image to local database
	existingImagesNameToId[nameWithoutTag] = imageID

	return nil
}

func ImageExists(imageName string) (bool, error) {
	log.Debug("Checking if the image ", imageName, " exists")
	nameWithoutTag, tag := getNameAndTag(imageName)

	// first make a lookup in the local database
	if existingImagesNameToId[nameWithoutTag] != "" {
		return true, nil
	} else {
		// if it fails look through all images on the device
		commandUrl := "/docker/images"
		resp, err := client.Get(edgeUrl + commandUrl)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("ReadAllContainers: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
			return false, err
		}

		type ImageInfo map[string]interface{}
		var resp_json map[string][]ImageInfo
		json.Unmarshal(body, &resp_json)

		images := resp_json["images"]
		for _, image := range images {
			currentImageName, currentImageTag := getNameAndTag(image["tags"].([]interface{})[0].(string))
			log.Debug("Looking at image ", currentImageName+":"+currentImageTag)
			if currentImageName == nameWithoutTag {
				if tag != "" {
					if currentImageTag == tag {
						// make sure the image is in the local database
						existingImagesNameToId[currentImageName] = image["id"].(string)
						return true, nil
					}
				} else {
					// make sure the image is in the local database
					existingImagesNameToId[currentImageName] = image["id"].(string)
					return true, nil
				}
			}
		}
		return false, nil
	}
}

func ImageRemove(imageID string) error {
	commandUrl := fmt.Sprintf("/docker/images/%s", imageID)
	req, err := http.NewRequest(http.MethodDelete, edgeUrl+commandUrl, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		err = fmt.Errorf("ImageRemove: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return err
	}

	// remove image from local database
	for name, id := range existingImagesNameToId {
		if id == imageID {
			delete(existingImagesNameToId, name)
			break
		}
	}

	log.Debug("Removed image ID ", imageID)
	return nil
}
