package controller

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// POSTpipelines creates pipeline based on input manifest
func POSTpipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")
	log.Fatal("THIS FUNCTION IS OBSELETE!")
	//Get the manifest as a []byte
	// manifestBodyBytes, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	log.Error(err)
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// man, err := model.ParseJSONManifest(manifestBodyBytes)
	// if err != nil {
	// 	log.Error(err)
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// resp := DeployManifest(man)
	// if resp == "SUCCESS" {
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Write([]byte("200 - Request processed successfully!"))
	// } else {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	w.Write([]byte("500 - Failed to create container!"))
	// }

}
