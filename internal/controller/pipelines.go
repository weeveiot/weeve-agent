package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Jeffail/gabs/v2"

	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

// POSTpipelines function to,
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func POSTpipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	// Decode the JSON manifest into Golang struct
	// manifest := model.ManifestReq{}
	manifestBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	// res := util.PrintManifestDetails(body)
	// fmt.Println(res)
	// util.PrettyPrintJson(body)

	/*
		err = util.DecodeJSONBody(w, r, &manifest)
		if err != nil {
			var mr *util.MalformedRequest
			if errors.As(err, &mr) {
				log.Error(err.Error())
				http.Error(w, mr.Msg, mr.Status)
			} else {
				log.Error(err.Error())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
	*/

	//******** STEP 1 - Pull all *************//
	// Pull all images as required
	log.Debug("STEP 1 - Iterate modules, Docker Pull image into host if missing")
	// var imageNamesList []string
	// for i := range manifest.Modules {
	// 	imageNamesList = append(imageNamesList, manifest.Modules[i].ImageName)
	// }
	imageNamesList := util.ParseImagesList(manifestBodyBytes)
	imagesPulled := docker.PullImagesNew(imageNamesList)

	//******** STEP 2 - Check if pulled *************//
	// Check if all images pulled, else return
	log.Debug("STEP 2 - Check if all images pulled, else return")
	if imagesPulled == false {
		msg := "Unable to pull all images"
		log.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	//******** STEP 3 - Check containers, stop and remove *************//
	// Create and start containers
	log.Debug("STEP 3 - Check containers, stop and remove")

	jsonParsed, err := gabs.ParseJSON(manifestBodyBytes)
	if err != nil {
		panic(err)
	}

	for _, mod := range jsonParsed.Search("Modules").Children() {
		log.Debug(fmt.Sprintf("\t***** index: %v, name: %v", mod.Search("Index").Data(), mod.Search("Name").Data()))
		log.Debug(fmt.Sprintf("\timage %v:%v", mod.Search("ImageName").Data(), mod.Search("Tag").Data()))
		log.Debug("\toptions:")
		for _, opt := range mod.Search("options").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v = %v", opt.Search("opt").Data(), opt.Search("val").Data()))
		}
		log.Debug("\targuments:")
		for _, arg := range mod.Search("arguments").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))
		}
		containerName := GetContainerName(jsonParsed.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		log.Info("\tConstructed container name:", containerName)

		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Stop + remove container if exists, start fresh
		if containerExists {
			log.Debug("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				// msg := ""
				log.Error(err)
				http.Error(w, string(err.Error()), http.StatusInternalServerError)
			}
			log.Debug("\tContainer ", containerName, " removed")
		}
	}

	/*
		for i := range manifest.Modules {
			mod := manifest.Modules[i]
			log.Debug("\tModule: ", mod.ImageName)

			// Build container name
			containerName := GetContainerName(manifest.ID, mod.Name)
			log.Info("\tContainer name:", containerName)

			// Check if container already exists
			containerExists := docker.ContainerExists(containerName)
			log.Info("\tContainer exists:", containerExists)

			// Create container if not exists
			if containerExists {
				log.Debug("\tStopAndRemoveContainer - ", containerName)
				// Stop and delete container
				err := docker.StopAndRemoveContainer(containerName)
				if err != nil {
					// msg := ""
					log.Error(err)
					http.Error(w, string(err.Error()), http.StatusInternalServerError)
				}
				log.Debug("\tContainer ", containerName, " removed")
			}

		}
	*/

	//******** STEP 4 - Start all containers *************//
	// Start all containers iteratively

	for _, mod := range jsonParsed.Search("Modules").Children() {
		containerName := GetContainerName(jsonParsed.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		imageName := mod.Search("ImageName").Data().(string)
		imageTag := mod.Search("Tag").Data().(string)
		log.Info("\tPreparing command for container " + containerName + "from image" + imageName + " " + imageTag)

		for _, opt := range mod.Search("options").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v = %v", opt.Search("opt").Data(), opt.Search("val").Data()))
		}

		for _, arg := range mod.Search("arguments").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))
		}

		// Create and start container
		argsString := "asdf"
		docker.CreateContainerOptsArgs(containerName, imageName, argsString)
		// log.Info("\tCreateContainer - successfully started:", containerName)

	}
	/*
		log.Debug("Iterate modules, start each container")
		for i := range manifest.Modules {
			mod := manifest.Modules[i]
			// log.Debug("\tContainer: ", mod.ImageName)

			// Build container name
			containerName := GetContainerName(manifest.ID, mod.Name)
			log.Info("\tCreateContainer - Container name:", containerName)

			// Create and start container
			docker.CreateContainer(containerName, mod.ImageName)
			log.Info("\tCreateContainer - successfully started:", containerName)
		}
	*/

	/*
		log.Info("Pipeline successfully instantiated from manifest ", manifest.ID)

		// Finally, return 200
		// Return payload: pipeline started / list of container IDs
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("200 - Request processed successfully!"))
	*/
	return
}

// GetContainerName build container name
func GetContainerName(pipelineID string, containerName string) string {
	return pipelineID + "_" + containerName
}
