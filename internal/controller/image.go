// image
package controller

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
)

func PullImage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: PullImage")
	vars := mux.Vars(r)
	imageName := vars["imageName"]
	docker.PullImage(imageName)
}
