//go:build !secunet

package secunet

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/weeveiot/weeve-agent/internal/model"
)

const registryUrl = "https://registry-1.docker.io"
const authUrl = "https://auth.docker.io"
const svcUrl = "registry.docker.io"

const host = "edge.internal"
const port = 4444

var edgeUrl = fmt.Sprintf("https://%s:%d/rest_api/v1", host, port)
var client http.Client

func init() {
	certFile := "/var/hostdir/clientcert/cert.pem"
	keyFile := "/var/hostdir/clientcert/key.pem"
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// // Save TLS keys to decrypt communication with Wireshark
	// w, err := os.OpenFile("/home/paul/.ssl-key.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			// KeyLogWriter:       w,
			InsecureSkipVerify: true,
		},
	}
	client = http.Client{Transport: t}
}

func StartContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s/start", containerID)
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, nil)
	if err != nil {
		log.Error(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		log.Debug("Started container ID ", containerID)
	}

	return nil
}

func CreateAndStartContainer(containerConfig model.ContainerConfig) (string, error) {
	imageID := getImageID(containerConfig.ImageName, containerConfig.ImageTag)
	containerName := containerConfig.ContainerName + containerConfig.Labels["manifestID"] + containerConfig.Labels["version"]

	// transform port map into string
	var portBindingPairs string
	for port, bindings := range containerConfig.PortBinding {
		for _, binding := range bindings {
			portBindingPairs += " " + port.Port() + ":" + binding.HostPort
		}
	}

	postCommandUrl := fmt.Sprintf("/docker/images/%s/createcontainer", imageID)
	resp, err := client.PostForm(edgeUrl+postCommandUrl, url.Values{
		"params_name":    []string{containerName},
		"params_p":       []string{portBindingPairs},
		"params_e":       containerConfig.EnvArgs,
		"params_network": []string{containerConfig.NetworkName},
	})
	if err != nil {
		log.Error(err)
	}

	var resp_json map[string]string
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal(body, &resp_json)
	containerID := resp_json["id"]

	log.Debug("Created container " + containerConfig.ContainerName)

	return containerID, nil
}

func StopContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s/stop", containerID)
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, nil)
	if err != nil {
		log.Error(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		log.Debug("Stopped container ID ", containerID)
	}

	return nil
}

func StopAndRemoveContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s", containerID)
	req, err := http.NewRequest(http.MethodDelete, edgeUrl+commandUrl, nil)
	if err != nil {
		log.Error(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode == 200 {
		log.Debug("Killed container ID ", containerID)
	}

	return nil
}

func ReadAllContainers() ([]types.Container, error) {
	commandUrl := fmt.Sprintf("/docker/container")
	resp, err := client.Get(edgeUrl + commandUrl)
	if err != nil {
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	var containerStructs []types.Container
	if resp.StatusCode == 200 {
		type ContainerInfo map[string]string
		var resp_json map[string][]ContainerInfo
		json.Unmarshal(body, &resp_json)

		containers := resp_json["containers"]
		for _, container := range containers {
			containerStructs = append(containerStructs, types.Container{
				ID:      container["id"],
				Names:   []string{container["name"]},
				ImageID: container["image_id"],
				// Created: container["created"],	// TODO convert to int64 if needed
				Status: container["status"],
			})
		}
		return containerStructs, nil
	} else {
		err = errors.New(fmt.Sprintf("ReadAllContainers: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body))
		return nil, err
	}
}

func ReadDataServiceContainers(manifestID string, version string) ([]types.Container, error) {
	var dataServiceContainers []types.Container

	allContainers, err := ReadAllContainers()
	if err != nil {
		return nil, err
	}

	for _, container := range allContainers {
		if strings.Contains(container.Names[0], manifestID) && strings.Contains(container.Names[0], version) {
			dataServiceContainers = append(dataServiceContainers, container)
		}
	}

	return dataServiceContainers, nil
}
