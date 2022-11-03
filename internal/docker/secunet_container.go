//go:build secunet

package docker

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

const registryUrl = "https://registry-1.docker.io"
const authUrl = "https://auth.docker.io"
const svcUrl = "registry.docker.io"

const host = "edge.internal"
const port = 4444

var edgeUrl = fmt.Sprintf("https://%s:%d/rest_api/v1", host, port)
var client http.Client

var existingContainers = make(map[string]string)

func SetupDockerClient() {
	const certDir = "/var/hostdir/clientcert"
	const certFile = certDir + "/" + "cert.pem"
	const keyFile = certDir + "/" + "key.pem"
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Unable to setup docker client! CAUSE --> ", err)
	}

	// // Save TLS keys to decrypt communication with Wireshark
	// w, err := os.OpenFile("/home/paul/.ssl-key.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	// if err != nil {
	// 	log.Fatal("Unable to open .ssl-key.log! CAUSE --> ", err)
	// }

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			// KeyLogWriter:       w,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS13,
		},
	}
	client = http.Client{Transport: t}
}

func StartContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s/start", containerID)
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, nil)
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

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("StartContainer: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return traceutility.Wrap(err)
	}

	log.Debug("Started container ID ", containerID)
	return nil
}

func CreateAndStartContainer(containerConfig manifest.ContainerConfig) (string, error) {
	imageID := existingImagesNameToId[containerConfig.ImageName]

	// transform port map into string
	var portBindingPairs []string
	for port, bindings := range containerConfig.PortBinding {
		for _, binding := range bindings {
			portBindingPairs = append(portBindingPairs, binding.HostPort+":"+port.Port())
		}
	}

	postCommandUrl := fmt.Sprintf("/docker/images/%s/createcontainer", imageID)

	req_body := map[string]string{
		"params_name":    containerConfig.ContainerName,
		"params_p":       strings.Join(portBindingPairs, " "),
		"params_e":       strings.Join(containerConfig.EnvArgs, " "),
		"params_network": containerConfig.NetworkName,
	}

	log.Debug("Creating container with following params: ", req_body)

	req_json, err := json.Marshal(req_body)
	if err != nil {
		return "", traceutility.Wrap(err)
	}

	req, err := http.NewRequest(http.MethodPost, edgeUrl+postCommandUrl, bytes.NewBuffer(req_json))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	var resp_json map[string]string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("CreateAndStartContainer: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return "", traceutility.Wrap(err)
	}

	json.Unmarshal(body, &resp_json)
	containerID := resp_json["id"]

	log.Debugln("Created container", containerConfig.ContainerName, "with ID", containerID)

	existingContainers[containerID] = containerConfig.Labels["manifestName"] + containerConfig.Labels["versionNumber"]

	return containerID, nil
}

func StopContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s/stop", containerID)
	req, err := http.NewRequest(http.MethodPut, edgeUrl+commandUrl, nil)
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

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("StopContainer: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return traceutility.Wrap(err)
	}

	log.Debug("Stopped container ID ", containerID)
	return nil
}

func StopAndRemoveContainer(containerID string) error {
	commandUrl := fmt.Sprintf("/docker/container/%s", containerID)
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

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("StopAndRemoveContainer: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return traceutility.Wrap(err)
	}

	log.Debug("Killed container ID ", containerID)
	delete(existingContainers, containerID)
	return nil
}

func ReadAllContainers() ([]types.Container, error) {
	commandUrl := fmt.Sprintf("/docker/container")
	resp, err := client.Get(edgeUrl + commandUrl)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ReadAllContainers: HTTP request failed. Code: %d Message: %s", resp.StatusCode, body)
		return nil, traceutility.Wrap(err)
	}

	var containerStructs []types.Container
	type ContainerInfo map[string]string
	var resp_json map[string][]ContainerInfo
	json.Unmarshal(body, &resp_json)

	containers := resp_json["containers"]
	for _, container := range containers {
		containerStructs = append(containerStructs, types.Container{
			ID:      container["id"],
			Names:   []string{container["name"]},
			ImageID: container["image_id"][:12],
			State:   container["status"],
			// Created: container["created"],	// TODO convert to int64 if needed
		})
	}
	return containerStructs, nil
}

func ReadDataServiceContainers(manifestUniqueID model.ManifestUniqueID) ([]types.Container, error) {
	var dataServiceContainers []types.Container

	allContainers, err := ReadAllContainers()
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	for _, container := range allContainers {
		if existingContainers[container.ID] == manifestUniqueID.ManifestName+manifestUniqueID.VersionNumber {
			dataServiceContainers = append(dataServiceContainers, container)
		}
	}

	return dataServiceContainers, nil
}
