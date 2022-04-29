package handler

import (
	"time"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	jsonutility "github.com/weeveiot/weeve-agent/internal/utility/json"
)

type statusMessage struct {
	Id                 string           `json:"ID"`
	Timestamp          int64            `json:"timestamp"`
	Status             string           `json:"status"`
	ActiveServiceCount int              `json:"activeServiceCount"`
	ServiceCount       int              `json:"serviceCount"`
	ServicesStatus     []manifestStatus `json:"servicesStatus"`
	DeviceParams       deviceParams     `json:"deviceParams"`
}

type manifestStatus struct {
	ManifestId      string `json:"manifestId"`
	ManifestVersion string `json:"manifestVersion"`
	Status          string `json:"status"`
}

type deviceParams struct {
	Sensors string `json:"sensors"`
	Uptime  string `json:"uptime"`
	CpuTemp string `json:"cputemp"`
}

type registrationMessage struct {
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Name      string `json:"name"`
}

func ProcessMessage(operation string, payload []byte, retry bool) {
	// flag for exception handling
	exception := true
	defer func() {
		if exception && retry {
			// on exception sleep 5s and try again
			time.Sleep(5 * time.Second)
			ProcessMessage(operation, payload, false)
		}
	}()

	log.Info("Processing the message >> ", operation)

	jsonParsed, err := gabs.ParseJSON(payload)
	if err != nil {
		log.Error("Error on parsing message : ", err)
	} else {
		log.Debug("Parsed JSON >> ", jsonParsed)

		if operation == dataservice.CMDCheckVersion {

		} else if operation == dataservice.CMDDeploy {

			var thisManifest = manifest.Manifest{}
			thisManifest.Manifest = *jsonParsed
			err := dataservice.DeployDataService(thisManifest, operation)
			if err != nil {
				log.Info("Deployment failed! CAUSE --> ", err, "!")
			} else {
				log.Info("Deployment done!")

			}

		} else if operation == dataservice.CMDReDeploy {

			var thisManifest = manifest.Manifest{}
			thisManifest.Manifest = *jsonParsed
			err := dataservice.DeployDataService(thisManifest, operation)
			if err != nil {
				log.Info("Redeployment failed! CAUSE --> ", err, "!")
			} else {
				log.Info("Redeployment done!")

			}

		} else if operation == dataservice.CMDStopService {

			err := manifest.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := dataservice.StopDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Service stopped!")
				}
			}

		} else if operation == dataservice.CMDStartService {

			err := manifest.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := dataservice.StartDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Service started!")
				}
			}

		} else if operation == dataservice.CMDUndeploy {

			err := manifest.ValidateStartStopJSON(jsonParsed)
			if err == nil {
				serviceId := jsonParsed.Search("id").Data().(string)
				serviceVersion := jsonParsed.Search("version").Data().(string)

				err := dataservice.UndeployDataService(serviceId, serviceVersion)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("Undeployment done!")
				}
			}
		}
	}

	exception = false
}

func GetStatusMessage(nodeId string) statusMessage {
	manifests, err := jsonutility.Read(dataservice.ManifestFile, nil, false)

	if err != nil {
		return statusMessage{}
	}

	var mani []manifestStatus
	var deviceParams = deviceParams{Sensors: "10", Uptime: "10", CpuTemp: "20"}

	actv_cnt := 0
	serv_cnt := 0
	for _, rec := range manifests {
		mani = append(mani, manifestStatus{ManifestId: rec["id"].(string), ManifestVersion: rec["version"].(string), Status: rec["status"].(string)})
		serv_cnt = serv_cnt + 1
		if rec["status"].(string) == "SUCCESS" {
			actv_cnt = actv_cnt + 1
		}
	}

	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000
	return statusMessage{Id: nodeId, Timestamp: millis, Status: "Available", ActiveServiceCount: actv_cnt, ServiceCount: serv_cnt, ServicesStatus: mani, DeviceParams: deviceParams}
}

func GetRegistrationMessage(nodeId string, nodeName string) registrationMessage {
	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000

	return registrationMessage{Id: nodeId, Timestamp: millis, Status: "Registering", Operation: "Registration", Name: nodeName}
}
