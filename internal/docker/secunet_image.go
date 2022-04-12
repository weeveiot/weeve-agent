//go:build secunet

package docker

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

var existingImagesNameToId = make(map[string]string)

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
	// 	log.Error(err)
	// }

	// getManifest(token, imgDetails.ImageName, "")

	// INTERIM SOLUTION
	downloadScriptName := "download-frozen-image-v2.sh"
	archiveScriptName := "archive.sh"
	nameWithoutTag, _ := getNameAndTag(imgDetails.ImageName)
	fileName := nameWithoutTag + ".tar.gz"
	cmd := exec.Command("./"+downloadScriptName, nameWithoutTag, imgDetails.ImageName)
	out, err := cmd.CombinedOutput()

	fmt.Println(string(out))
	if err != nil {
		log.Error(err)
		return err
	}

	cmd = exec.Command("./"+archiveScriptName, nameWithoutTag)
	err = cmd.Run()
	if err != nil {
		log.Error(err)
		return err
	}

	// New multipart writer.
	bufferedFile := &bytes.Buffer{}
	writer := multipart.NewWriter(bufferedFile)
	fw, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		log.Error(err)
		return err
	}
	fd, err := os.Open(fileName)
	if err != nil {
		log.Error(err)
		return err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Error(err)
		return err
	}
	writer.Close()

	commandUrl := "/docker/images"
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, bytes.NewReader(bufferedFile.Bytes()))
	if err != nil {
		log.Error(err)
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// username := "Admin"
	// password := "Secure Visibility"
	// req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return err
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
		existingImagesNameToId[nameWithoutTag] = imageID

		return nil
	} else {
		err = errors.New(fmt.Sprintf("PullImage: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return err
	}
}

func ImageExists(imageName string) (bool, error) {
	nameWithoutTag, tag := getNameAndTag(imageName)

	// first make a lookup in the local database
	if existingImagesNameToId[nameWithoutTag] != "" {
		return true, nil
	} else {
		// if it fails look through all images on the device
		commandUrl := fmt.Sprintf("/docker/images")
		resp, err := client.Get(edgeUrl + commandUrl)
		if err != nil {
			log.Error(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Error(err)
		}

		if resp.StatusCode == 200 {
			type ImageInfo map[string]string
			var resp_json map[string][]ImageInfo
			json.Unmarshal(body, &resp_json)

			images := resp_json["images"]
			for _, image := range images {
				if image["repository"] == nameWithoutTag {
					if tag != "" {
						if image["tag"] == tag {
							return true, nil
						}
					} else {
						return true, nil
					}
				}
			}
			return false, nil
		} else {
			err = errors.New(fmt.Sprintf("ReadAllContainers: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
			return false, err
		}
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
		for name, id := range existingImagesNameToId {
			if id == imageID {
				delete(existingImagesNameToId, name)
			}
		}

		log.Debug("Removed image ID ", imageID)
		return nil
	} else {
		err = errors.New(fmt.Sprintf("ImageRemove: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return err
	}
}
