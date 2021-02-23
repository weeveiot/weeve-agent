// Package controller container
package controller

import (
	"fmt"

	"encoding/json"
	"net/http"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// GETcontainersID get all containers IDs
func GETcontainersID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: getContainer")
	// vars := mux.Vars(r)
	// key := vars["id"]

	containers := docker.ReadAllContainers()
	json.NewEncoder(w).Encode(containers[0])
}

// GETcontainers get All Containers returns all created (started & stopped) contianers
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
func GETcontainers(w http.ResponseWriter, r *http.Request) {
	log.Info("GET /containers")
	containers := docker.ReadAllContainers()
	log.Debug(len(containers), " containers found")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// w.Write([]byte("200 - Request processed successfully!"))
	log.Info("GET /containers response >> ", containers)
	json.NewEncoder(w).Encode(containers)
}

// GETcontainersIDlogs returns container logs
func GETcontainersIDlogs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAll logs of Container")
	vars := mux.Vars(r)
	key := vars["id"]
	logs := docker.GetContainerLog(key)
	json.NewEncoder(w).Encode(logs)
}

// POSTcontainersStart starts all stopped containers
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
func POSTcontainersStart(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StartContainers")
	// vars := mux.Vars(r)
	// key := vars["id"]
	res := docker.StartContainers()
	json.NewEncoder(w).Encode(res)
}

// POSTcontainersStop stops all started containers
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
func POSTcontainersStop(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StopContainers")
	// vars := mux.Vars(r)
	// key := vars["id"]
	res := docker.StopContainers()
	json.NewEncoder(w).Encode(res)
}

// POSTcontainersStartID starts single stopped container by ID
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
func POSTcontainersStartID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StartContainers")
	vars := mux.Vars(r)
	key := vars["id"]
	res := docker.StartContainer(key)
	json.NewEncoder(w).Encode(res)
}

// POSTcontainersStopID stops single started container by ID
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
func POSTcontainersStopID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: StopContainers")
	vars := mux.Vars(r)
	key := vars["id"]
	res := docker.StopContainer(key)
	json.NewEncoder(w).Encode(res)
}

// POSTcontainersDeploy will pull image and create container based on input Image and Container name
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
//TODO: Refactor!
func POSTcontainersDeploy(w http.ResponseWriter, r *http.Request) {
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

	// res := docker.CreateContainer(manifest.Name, manifest.ImageName)
	// json.NewEncoder(w).Encode(res)
}

// func UpdateContainer(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("Endpoint: createContainer")
// 	vars := mux.Vars(r)
// 	key := vars["id"]
// 	// container := nil

// 	json.NewEncoder(w).Encode(key)
// }

// DELETEcontainersID TODO: Not implemented!
func DELETEcontainersID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: Delete Container")
	w.WriteHeader(http.StatusNotImplemented)
	// vars := mux.Vars(r)
	// key := vars["id"]
	// // container := nil
	// // dao.DeleteData(key)
	// json.NewEncoder(w).Encode(key)
}
