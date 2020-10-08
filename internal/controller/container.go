// container
package controller

import (
	"fmt"

	"encoding/json"
	"net/http"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"github.com/gorilla/mux"
)

func GetContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: getContainer")
	// vars := mux.Vars(r)
	// key := vars["id"]

	containers := docker.ReadAllContainers()
	json.NewEncoder(w).Encode(containers[0])
}

// Get All Containers returns all created (started & stopped) contianers
// @Summary Get all containers
// @Description Get all containers
// @Tags containers
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /containers [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GetAllContainers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAllContainers")
	// vars := mux.Vars(r)
	// key := vars["id"]

	// var host string
	// host = "ec2-3-128-113-67.us-east-2.compute.amazonaws.com"

	// containers := docker.ReadRemoteContainers(host, "true")
	// for k, v := range containers {
	// 	fmt.Println(k, v)
	// }

	containers := docker.ReadAllContainers()
	json.NewEncoder(w).Encode(containers)
}

func GetContainerLog(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAll logs of Container")
	vars := mux.Vars(r)

	key := vars["id"]

	// stream := vars["stream"]

	// fmt.Println(stream)

	// var host string
	// host = "ec2-3-128-113-67.us-east-2.compute.amazonaws.com"

	logs := docker.GetContainerLog(key)
	// for k, v := range containers {
	// 	fmt.Println(k, v)
	// }
	json.NewEncoder(w).Encode(logs)
}

// Start all stopped containers
// @Summary Start all stopped containers
// @Description Start all stopped containers
// @Tags containers
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /containers/start [post]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func StartContainers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StartContainers")
	// vars := mux.Vars(r)
	// key := vars["id"]
	res := docker.StartContainers()
	json.NewEncoder(w).Encode(res)
}

// Stop all started containers
// @Summary Stop all started containers
// @Description Stop all started containers
// @Tags containers
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /containers/stop [post]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func StopContainers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StopContainers")
	// vars := mux.Vars(r)
	// key := vars["id"]
	res := docker.StopContainers()
	json.NewEncoder(w).Encode(res)
}

// Start single stopped container by ID
// @Summary Start single stopped container by ID
// @Description Start single stopped container by ID
// @Tags containers
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param id path string true "id"
// @Router /containers/start/{id} [post]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func StartContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StartContainers")
	vars := mux.Vars(r)
	key := vars["id"]
	res := docker.StartContainer(key)
	json.NewEncoder(w).Encode(res)
}

// Stop single started container by ID
// @Summary Stop single started container by ID
// @Description Stop single started container by ID
// @Tags containers
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param id path string true "id"
// @Router /containers/stop/{id} [post]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func StopContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StopContainers")
	vars := mux.Vars(r)
	key := vars["id"]
	res := docker.StopContainer(key)
	json.NewEncoder(w).Encode(res)
}

// CreateContainer will pull image and create container based on input Image and Container name
// @Summary Pull image, Create container and Start container
// @Description Pull image, Create container and Start container
// @Tags containers
// @Accept  json
// @Produce  string
// @Success 200
// @Failure 400
// @Param image name to pull, string "imageName"
// @Param container name to create container, string  "containerName"
// @Router /containers/create/{containerName}/{imageName} [post]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func CreateContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: CreateContainer")
	// vars := mux.Vars(r)
	// imageName := vars["imageName"]
	// containerName := vars["containerName"]

	manifest := &model.Manifest{}

	err := json.NewDecoder(r.Body).Decode(manifest)
	if err != nil {
		var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	res := docker.CreateContainer(manifest.ContainerName, manifest.ImageName)
	json.NewEncoder(w).Encode(res)
}

func UpdateContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createContainer")
	vars := mux.Vars(r)
	key := vars["id"]
	// container := nil

	json.NewEncoder(w).Encode(key)
}

func DeleteContainer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createContainer")
	vars := mux.Vars(r)
	key := vars["id"]
	// container := nil
	// dao.DeleteData(key)
	json.NewEncoder(w).Encode(key)
}

// type Container struct {
// 	Id      string `json:"Id"`
// 	Name    string `json:"Name"`
// 	Tag     string `json:"Tag"`
// 	Image   string `json:"Image"`
// 	ImageID string `json:"ImageID"`
// 	Command string `json:"Command"`
// 	State string `json:"State"`
// 	Status string `json:"Status"`
// 	Ports []Port `json:"Port"`
// 	Labels Labels `json:"Labels"`
// 	HostConfig HostConfig `json:"HostConfig"`
// 	NetworkSettings NetworkSettings `json:"NetworkSettings"`
// 	Mounts []Mounts `json:"Mounts"`
// }

// type Labels struct {
// 	label []string `json:"label"`
// }

// type Mounts struct {
// 	Name string\`json:"Name"`
// 	Source string\`json:"Source"`
// 	Destination string\`json:"Destination"`
// 	Driver string\`json:"Driver"`
// 	Mode string\`json:"Mode"`
// 	RW bool\`json:"RW"`
// 	Propagation string\`json:"Propagation"`

// }

// type NetworkSettings struct {
// 	Networks Networks `json:"Networks"`
// }

// type Networks struct {
// 	Bridge BridgeNW`json:"Bridge"`
// }
// type BridgeNW struct {
// 	IPAMConfig string `json:"IPAMConfig"`
// 	Links string `json:"Links"`
// 	Aliases string `json:"Aliases"`
// 	NetworkID string `json:"NetworkID"`
// 	EndpointID string `json:"EndpointID"`
// 	Gateway string `json:"Gateway"`
// 	IPAddress string `json:"IPAddress"`
// 	IPPrefixLen string `json:"IPPrefixLen"`
// 	IPv6Gateway string `json:"IPv6Gateway"`
// 	GlobalIPv6Address string `json:"GlobalIPv6Address"`
// 	GlobalIPv6PrefixLen string `json:"GlobalIPv6PrefixLen"`
// 	MacAddress string `json:"MacAddress"`
// }

// type HostConfig struct {
// 	NetworkMode string `json:"NetworkMode"`
// }

// type Port struct {
// 	PrivatePort int `json:"PrivatePort"`
// 	PublicPort int `json:"PublicPort"`
// 	Type string `json:"Type"`
// }

// var Containers []Container
