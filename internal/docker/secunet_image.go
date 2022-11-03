//go:build secunet

package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

var existingImagesNameToId = make(map[string]string)

func getAuthToken(imageName string) (string, error) {
	commandUrl := fmt.Sprintf("%s/token?service=%s&scope=repository:library/%s:pull", authUrl, svcUrl, imageName)
	resp, err := http.Get(commandUrl)
	if err != nil {
		return "", traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("PullImage: Could not get the authentication token. HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return "", traceutility.Wrap(err)
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
		return nil, traceutility.Wrap(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("getManifest: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return nil, traceutility.Wrap(err)
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
func PullImage(authConfig types.AuthConfig, imageName string) error {
	// TODO: transform to HTTPS

	// token, err := getAuthToken(imageName)
	// if err != nil {
	// 	return traceutility.Wrap(err)
	// }

	// getManifest(token, imageName, "")

	// INTERIM SOLUTION
	const downloadScriptName = "download-frozen-image-v2.sh"
	const archiveScriptName = "archive.sh"
	nameWithoutTag, _ := getNameAndTag(imageName)
	fileName := nameWithoutTag + ".tar.gz"
	cmd := exec.Command("./"+downloadScriptName, nameWithoutTag, imageName)
	out, err := cmd.CombinedOutput()

	fmt.Println(string(out))
	if err != nil {
		return traceutility.Wrap(err)
	}

	cmd = exec.Command("./"+archiveScriptName, nameWithoutTag)
	err = cmd.Run()
	if err != nil {
		return traceutility.Wrap(err)
	}

	// New multipart writer.
	bufferedFile := &bytes.Buffer{}
	writer := multipart.NewWriter(bufferedFile)
	fw, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return traceutility.Wrap(err)
	}
	fd, err := os.Open(fileName)
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		return traceutility.Wrap(err)
	}
	writer.Close()

	commandUrl := "/docker/images"
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, bytes.NewReader(bufferedFile.Bytes()))
	if err != nil {
		return traceutility.Wrap(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("PullImage: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return traceutility.Wrap(err)
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
			return false, traceutility.Wrap(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, traceutility.Wrap(err)
		}

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("ReadAllContainers: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
			return false, traceutility.Wrap(err)
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
		return traceutility.Wrap(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		err = fmt.Errorf("ImageRemove: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return traceutility.Wrap(err)
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
