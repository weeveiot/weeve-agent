package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
)

func POSTpipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	//Get the manifest as a []byte
	manifestBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	man := model.ParseJSONManifest(manifestBodyBytes)
	// res := util.PrintManifestDetails(body)
	// util.PrettyPrintJson(body)

	//******** STEP 1 - Pull all *************//
	// Pull all images as required
	log.Debug("Iterating modules, pulling image into host if missing")

	for i, imgName := range man.ImageNamesList() {
		// Check if image exist in local
		exists := docker.ImageExists(imgName)
		if exists { // Image already exists, continue
			log.Debug(fmt.Sprintf("\tImage %v %v, already exists on host", i, imgName))
		} else { // Pull this image
			log.Debug(fmt.Sprintf("\tImage %v %v, does not exist on host", i, imgName))
			log.Debug("\t\tPulling ", imgName)
			exists = docker.PullImage(imgName)
			if exists == false {
				msg := "Unable to pull image " + imgName
				log.Error(msg)
				http.Error(w, msg, http.StatusInternalServerError)
			}
		}
	}

	//******** STEP 2 - Check containers, stop and remove *************//
	log.Debug("Checking containers, stopping and removing")

	for _, mod := range man.Manifest.Search("Modules").Children() {

		containerName := GetContainerName(man.Manifest.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		// log.Info("\tConstructed container name:", containerName)

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
	log.Debug("STEP 4 - Start all containers")
	for _, mod := range man.Manifest.Search("Modules").Children() {
		containerName := GetContainerName(man.Manifest.Search("ID").Data().(string), mod.Search("Name").Data().(string))
		imageName := mod.Search("ImageName").Data().(string)
		imageTag := mod.Search("Tag").Data().(string)

		for _, opt := range mod.Search("options").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v = %v", opt.Search("opt").Data(), opt.Search("val").Data()))
		}

		for _, arg := range mod.Search("arguments").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))
		}

		// Create and start container
		// argsString := "asdf"
		// argList := jsonParsed.Search("arguments").Data().(model.Argument)
		// TODO: Build the argument string as:
		/// InBroker=tcp://18.196.40.113:1883", "--ProcessName=container-1", "--InTopic=topic/source", "--InClient=weevenetwork/go-mqtt-gobot", "--OutBroker=tcp://18.196.40.113:1883", "--OutTopic=topic/c2", "--OutClient=weevenetwork/go-mqtt-gobot"},
		log.Info("\tPreparing command for container " + containerName + "from image" + imageName + " " + imageTag)
		var strArgs []string
		for _, arg := range mod.Search("arguments").Children() {
			strArgs = append(strArgs, "--"+arg.Search("arg").Data().(string)+"="+arg.Search("val").Data().(string))
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))

		}

		docker.CreateContainerOptsArgs(containerName, imageName, imageTag, strArgs)
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

	log.Info("Pipeline successfully instantiated from manifest ", man.Manifest.Search("Modules"))
	// Finally, return 200
	// Return payload: pipeline started / list of container IDs
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200 - Request processed successfully!"))
	return
}

