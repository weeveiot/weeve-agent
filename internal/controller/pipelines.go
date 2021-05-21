package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util/jsonlines"
)

// POSTpipelines creates pipeline based on input manifest
func POSTpipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	//Get the manifest as a []byte
	manifestBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// man := gabs.New()
	man, err := model.ParseJSONManifest(manifestBodyBytes)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if process is failed and needs to return
	failed := false

	jsonlines.Insert(constants.ManifestLogFile, man.Manifest.String())

	jsonlines.Delete(constants.ManifestFile, "id", man.Manifest.Search("id").Data().(string))

	// man.Manifest.Set("SUCCESS", "status")
	// jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

	// res := util.PrintManifestDetails(body)
	// util.PrettyPrintJson(manifestBodyBytes)
	// man.PrintManifest()
	// man.SpewManifest()
	// return

	//******** STEP 1 - Pull all *************//
	// Pull all images as required
	log.Info("Iterating modules, pulling image into host if missing")

	for i, imgDetails := range man.ImageNamesWithRegList() {
		// Check if image exist in local
		exists := docker.ImageExists(imgDetails.ImageName)
		if exists { // Image already exists, continue
			log.Info(fmt.Sprintf("\tImage %v %v, already exists on host", i, imgDetails.ImageName))
		} else { // Pull this image
			log.Info(fmt.Sprintf("\tImage %v %v, does not exist on host", i, imgDetails.ImageName))
			log.Info("\t\tPulling ", imgDetails.ImageName)
			exists = docker.PullImage(imgDetails)
			if exists == false {
				failed = true
				msg := "404 - Unable to pull image " + imgDetails.ImageName
				log.Error(msg)
				http.Error(w, msg, http.StatusNotFound)
				break
			}
		}
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return
	}

	//******** STEP 2 - Check containers, stop and remove *************//
	log.Info("Checking containers, stopping and removing")

	for _, containerName := range man.ContainerNamesList() {

		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Stop + remove container if exists, start fresh
		if containerExists {
			log.Info("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				failed = true
				log.Error(err)
				http.Error(w, string(err.Error()), http.StatusInternalServerError)
			}
			log.Info("\tContainer ", containerName, " removed")
		}
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return
	}

	//******** STEP 3 - Create the network *************//
	// var networkName = "my-net5"
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	log.Info("Pruning networks")
	filter := filters.NewArgs()

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	log.Info("Pruned:", pruneReport)
	// var report types.NetworksPruneReport
	log.Info("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	// var networkCreateOptions = &NetworkCreate

	// _ = ctx
	// _ = cli
	// fmt.Println(networkCreateOptions)
	networkName := man.GetNetworkName()
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		log.Error("Error trying to create network " + networkName)
		panic(err)

	}
	log.Info("Created network named ", networkName)

	_ = resp
	// log.Info(resp.ID, resp.Warning)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info("Start all containers")

	for _, startCommand := range man.GetContainerStart() {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand.ContainerName, startCommand.EntryPointArgs)
		log.Info("\tSuccessfully created with args: ", startCommand.EntryPointArgs)
		if err != nil {
			failed = true
			log.Info("Started")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Failed to create container!"))
		}

		// Attach to network
		var netConfig network.EndpointSettings
		err = cli.NetworkConnect(ctx, startCommand.NetworkName, containerCreateResponse.ID, &netConfig)
		if err != nil {
			panic(err)
		}
		log.Info("\tConnected to network", startCommand.NetworkName)
	}

	if failed {
		man.Manifest.Set("FAILED", "status")
		jsonlines.Insert(constants.ManifestFile, man.Manifest.String())
		return
	}
	man.Manifest.Set("SUCCESS", "status")
	jsonlines.Insert(constants.ManifestFile, man.Manifest.String())

	// Finally, return 200
	// Return payload: pipeline started / list of container IDs
	log.Info("Started")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200 - Request processed successfully!"))
	return
}
