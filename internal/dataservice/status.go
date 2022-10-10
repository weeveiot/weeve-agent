package dataservice

import (
	"strings"

	linq "github.com/ahmetb/go-linq/v3"
	"github.com/docker/docker/api/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/config"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

var disconnect bool

func SetDisconnected(disconnectParam bool) {
	disconnect = disconnectParam
}

func SendStatus(nodeStatus string) error {
	msg, err := GetStatusMessage(nodeStatus)
	if err != nil {
		return err
	}
	err = com.SendHeartbeat(msg)
	if err != nil {
		return err
	}
	return nil
}

func GetStatusMessage(nodeStatus string) (com.StatusMsg, error) {
	edgeApps, err := GetDataServiceStatus()
	if err != nil {
		return com.StatusMsg{}, err
	}

	deviceParams, err := getDeviceParams()
	if err != nil {
		return com.StatusMsg{}, err
	}

	if nodeStatus == "" {
		// TODO: Do proper check for node status
		nodeStatus = model.NodeAlarm
		if config.GetRegistered() {
			nodeStatus = model.NodeConnected
		}

		if disconnect {
			nodeStatus = model.NodeDisconnected
		}
	}

	msg := com.StatusMsg{
		Status:           nodeStatus,
		EdgeApplications: edgeApps,
		DeviceParams:     deviceParams,
	}

	return msg, nil
}

func GetDataServiceStatus() ([]com.EdgeAppMsg, error) {
	edgeApps := []com.EdgeAppMsg{}
	knownManifests := manifest.GetKnownManifests()

	for _, manif := range knownManifests {
		edgeApplication := com.EdgeAppMsg{ManifestID: manif.ManifestID, Status: manif.Status}
		containersStat := []com.ContainerMsg{}
		edgeAppStatusSet := false

		appContainers, err := docker.ReadDataServiceContainers(manif.ManifestUniqueID)
		if err != nil {
			return edgeApps, err
		}

		if !manif.InTransition {
			if len(appContainers) != manif.ContainerCount {
				edgeApplication.Status = model.EdgeAppError
				edgeAppStatusSet = true
			}

			if !edgeAppStatusSet && edgeApplication.Status == model.EdgeAppRunning {
				runningCount := linq.From(appContainers).Where(func(c interface{}) bool {
					return c.(types.Container).State == strings.ToLower(model.ModuleRunning)
				}).Count()

				if runningCount != len(appContainers) {
					edgeApplication.Status = model.EdgeAppError
				}

				edgeAppStatusSet = true
			}
		}

		for _, con := range appContainers {
			containerJSON, err := docker.InspectContainer(con.ID)
			if err != nil {
				return edgeApps, err
			}
			// The Status of each container is (assumed to be): Running, Restarting, Created, Exited
			container := com.ContainerMsg{Name: strings.Join(con.Names, ", "), Status: ioutility.FirstToUpper(con.State)}
			containersStat = append(containersStat, container)

			if !edgeAppStatusSet && !manif.InTransition && edgeApplication.Status == model.EdgeAppStopped {
				if con.State != strings.ToLower(model.ModuleExited) || containerJSON.State.ExitCode != 0 {
					edgeApplication.Status = model.EdgeAppError
				}
			}
		}

		edgeApplication.Containers = containersStat
		edgeApps = append(edgeApps, edgeApplication)
	}

	return edgeApps, nil
}

func CompareDataServiceStatus(edgeApps []com.EdgeAppMsg) ([]com.EdgeAppMsg, bool, error) {
	statusChange := false

	latestEdgeApps, err := GetDataServiceStatus()
	if err != nil {
		return nil, false, err
	}
	if len(edgeApps) == len(latestEdgeApps) {
		for index, edgeApp := range edgeApps {
			if edgeApp.Status != latestEdgeApps[index].Status {
				statusChange = true
			}
		}
	} else {
		statusChange = true
	}
	return latestEdgeApps, statusChange, nil
}

func getDeviceParams() (com.DeviceParamsMsg, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return com.DeviceParamsMsg{}, err
	}

	cpu, err := cpu.Percent(0, false)
	if err != nil {
		return com.DeviceParamsMsg{}, err
	}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return com.DeviceParamsMsg{}, err
	}

	verMem, err := mem.VirtualMemory()
	if err != nil {
		return com.DeviceParamsMsg{}, err
	}

	params := com.DeviceParamsMsg{
		SystemUpTime: uptime,
		SystemLoad:   cpu[0],
		StorageFree:  100.0 - diskStat.UsedPercent,
		RamFree:      float64(verMem.Available) / float64(verMem.Total) * 100.0,
	}
	return params, nil
}
