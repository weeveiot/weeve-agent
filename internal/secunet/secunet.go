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
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ahmetb/go-linq/v3"
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
var existingImagesIdToName = make(map[string]string)
var existingImagesNameToId = make(map[string]string)
var existingNetworks = make(map[string]string)

func init() {
	certFile := "/var/hostdir/clientcert/cert.pem"
	keyFile := "/var/hostdir/clientcert/key.pem"
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	client = http.Client{Transport: t}
}

// File: container.go

func getImageID(name, tag string) string {
	fullName := name + tag
	return existingImagesNameToId[fullName]
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
	resp, err := client.Get(commandUrl)
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

// File: image.go

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
	cmd := exec.Command("./"+downloadScriptName, nameWithoutTag, imgDetails.ImageName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error(err)
		return "", err
	}

	fmt.Println(string(out))

	cmd = exec.Command("./"+archiveScriptName, nameWithoutTag)
	err = cmd.Run()
	if err != nil {
		log.Error(err)
		return "", err
	}

	fd, err := os.Open(nameWithoutTag + ".tar.gz")
	if err != nil {
		log.Error(err)
		return "", err
	}
	defer fd.Close()

	commandUrl := "/docker/images"
	req, err := http.NewRequest(http.MethodPut, commandUrl, fd)
	if err != nil {
		log.Error(err)
		return "", err
	}
	// req.Header.Set("Content-Type", "text/plain")

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

// File: network.go

// Network name constraints
const manifestNamelength = 11
const indexLength = 3
const maxNetworkIndex = 999

func makeNetworkName(name string) string {
	format := "%s_%0" + fmt.Sprint(indexLength) + "d"

	// Prune the name if necessary
	if len(name) <= 0 {
		return ""
	} else if len(name) > manifestNamelength {
		name = name[:manifestNamelength]
	}

	// Get new network count
	var newCount int
	maxCount := getLastCreatedNetworkCount()
	if maxCount < maxNetworkIndex {
		newCount = maxCount + 1
	} else {
		newCount = getLowestAvailableNetworkCount()
		if newCount < 0 { // no available network count found
			log.Warning("Number of data services limit is exceeded")
			return ""
		}
	}

	// Generate next network name
	networkName := fmt.Sprintf(format, name, newCount)

	return strings.ReplaceAll(networkName, " ", "")
}

func readAllNetworks() []types.NetworkResource {
	log.Debug("Docker_container -> readAllNetworks")

	var networks []types.NetworkResource

	for _, networkName := range existingNetworks {
		networks = append(networks, types.NetworkResource{
			Name: networkName,
		})
	}

	return networks
}

func ReadDataServiceNetworks(manifestID string, version string) ([]types.NetworkResource, error) {
	key := manifestID + version
	networkName := existingNetworks[key]

	if networkName == "" {
		return nil, nil
	} else {
		networks := []types.NetworkResource{
			{
				Name: networkName,
			},
		}
		return networks, nil
	}
}

func CreateNetwork(name string, labels map[string]string) (string, error) {
	networkName := makeNetworkName(name)
	if networkName == "" {
		log.Error("Failed to generate Network Name")
		return "", errors.New("failed to generate network name")
	}

	key := labels["manifestID"] + labels["version"]
	existingNetworks[key] = networkName
	return networkName, nil
}

func NetworkPrune(manifestID string, version string) error {
	key := manifestID + version
	delete(existingNetworks, key)
	return nil
}

func getLastCreatedNetworkCount() int {
	maxCount := 0

	counts := getExistingNetworkCounts()

	for _, e := range counts {
		if e > maxCount {
			maxCount = e
		}
	}

	return maxCount
}

func getLowestAvailableNetworkCount() int {
	counts := getExistingNetworkCounts()

	// find lowest available network count
	for minAvailCount := 0; minAvailCount < maxNetworkIndex; minAvailCount++ {
		available := true
		for _, existingCount := range counts {
			if minAvailCount == existingCount {
				available = false
				break
			}
		}
		if available {
			return minAvailCount
		}
	}

	// no available count found
	return -1
}

func getExistingNetworkCounts() []int {
	var counts []int
	networks := readAllNetworks()
	linq.From(networks).Select(func(c interface{}) interface{} {
		nm := c.(types.NetworkResource).Name
		nm = nm[len(nm)-indexLength:]
		count, _ := strconv.Atoi(nm)
		return count
	}).ToSlice(&counts)
	return counts
}
