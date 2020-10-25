package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"gitlab.com/weeve/edge-server/edge-manager-service/internal/dao"
)

func TestGetAllImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAllImages")
	images := dao.ReadAllData("images.json")
	json.NewEncoder(w).Encode(images)
}

func TestGetImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: getImage")
	vars := mux.Vars(r)
	fmt.Println("vars", vars)
	key := vars["id"]
	image := dao.ReadSingleData(key, "images.json")
	json.NewEncoder(w).Encode(image)
}

func TestCreateImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	image := dao.SaveData(vars)
	json.NewEncoder(w).Encode(image)
}

func TestUpdateImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	image := dao.EditData(key, vars)
	json.NewEncoder(w).Encode(image)
}

func TestDeleteImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	image := dao.DeleteData(key)
	json.NewEncoder(w).Encode(image)
}
