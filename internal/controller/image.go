package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/aws"
	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/constants"
	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"

	"github.com/gorilla/mux"
)

// ShowImages godoc
// @Summary Get all images
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /images [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GetAllImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAllImages")
	images := docker.ReadAllImages()
	json.NewEncoder(w).Encode(images)
}

// ShowImages godoc
// @Summary Get all images
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param id path int true "Image ID"
// @Router /images/{id} [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GetImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: getImage")
	vars := mux.Vars(r)
	fmt.Println("vars", vars)
	key := vars["id"]
	images := docker.ReadImage(key)
	json.NewEncoder(w).Encode(images)
}

func CreateImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	// image := dao.SaveData(vars)
	json.NewEncoder(w).Encode(vars)
}

func UpdateImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	// image := dao.EditData(key, vars)
	json.NewEncoder(w).Encode(key)
}

func DeleteImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	// image := dao.DeleteData(key)
	json.NewEncoder(w).Encode(key)
}


/* OUT OF SCOPE - WE ONLY USE DOCKERHUB!
// GetAllEcrImages returns all images from ECR respository
// @Summary Get all images from Registry
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param parentPath path string true "parentPath"
// @Param imageName path string true "imageName"
// @Router /ecrimages/{parentPath}/{imageName} [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GetAllEcrImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAllImages")
	vars := mux.Vars(r)
	fmt.Println("vars", vars)
	repoName := vars["parentPath"]
	imageName := vars["imageName"]

	if repoName == "" {
		panic("Repository Name is required")
	}

	if imageName == "" {
		panic("Image Name is required")
	}

	repoName = repoName + "/" + imageName

	//TODO: AWS WAS PUT OUT OF SCOPE!
	// images := aws.ReadAllEcrImages(repoName, constants.RoleArn)
	json.NewEncoder(w).Encode(images)
}

*/

// type Image struct {
// 	Id   string `json:"Id"`
// 	Name string `json:"Name"`
// 	tag  string `json:"tag"`
// }

// var Images []Image
