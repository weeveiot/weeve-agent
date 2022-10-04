package dataservice

import (
	"strings"

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

func GetStatusMessage() (com.StatusMsg, error) {
	edgeApps, err := GetDataServiceStatus()
	if err != nil {
		return com.StatusMsg{}, err
	}

	deviceParams, err := getDeviceParams()
	if err != nil {
		return com.StatusMsg{}, err
	}

	// TODO: Do proper check for node status
	nodeStatus := model.NodeAlarm
	if config.GetRegistered() {
		nodeStatus = model.NodeConnected
	}

	if disconnect {
		nodeStatus = model.NodeDisconnected
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

		appContainers, err := docker.ReadDataServiceContainers(manif.ManifestUniqueID)
		if err != nil {
			return edgeApps, err
		}

		if !manif.InTransition && (manif.Status == model.EdgeAppRunning || manif.Status == model.EdgeAppPaused) && len(appContainers) != manif.ContainerCount {
			edgeApplication.Status = model.EdgeAppError
		}

		for _, con := range appContainers {
			// The Status of each container is (assumed to be): Running, Paused, Restarting, Created, Exited
			container := com.ContainerMsg{Name: strings.Join(con.Names, ", "), Status: ioutility.FirstToUpper(con.State)}
			containersStat = append(containersStat, container)

			if !manif.InTransition && edgeApplication.Status != model.EdgeAppError {
				if manif.Status == model.EdgeAppRunning && ioutility.FirstToUpper(con.State) != model.ModuleRunning {
					edgeApplication.Status = model.EdgeAppError
				}
				if manif.Status == model.EdgeAppPaused && ioutility.FirstToUpper(con.State) != model.ModulePaused {
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
		RamFree:      float64(verMem.Available) / float64(verMem.Total),
	}
	return params, nil
}
