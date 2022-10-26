package dataservice

import (
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	ioutility "github.com/weeveiot/weeve-agent/internal/utility/io"
)

var nodeStatus string = model.NodeAlarm

func SetNodeStatus(status string) {
	nodeStatus = status
}

func SendStatus() error {
	msg, err := GetStatusMessage()
	if err != nil {
		return err
	}
	err = com.SendHeartbeat(msg)
	if err != nil {
		return err
	}
	return nil
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

	msg := com.StatusMsg{
		Status:           nodeStatus,
		EdgeApplications: edgeApps,
		DeviceParams:     deviceParams,
	}

	return msg, nil
}

func GetDataServiceStatus() ([]com.EdgeAppMsg, error) {
	edgeApps := []com.EdgeAppMsg{}

	for _, manif := range manifest.GetKnownManifests() {
		edgeApplication := com.EdgeAppMsg{ManifestID: manif.Manifest.ID, Status: manif.Status}

		if manif.Status == model.EdgeAppUndeployed {
			edgeApps = append(edgeApps, edgeApplication)
			continue
		}

		appContainers, err := docker.ReadDataServiceContainers(manif.Manifest.ManifestUniqueID)
		if err != nil {
			return edgeApps, err
		}

		if (manif.Status == model.EdgeAppRunning || manif.Status == model.EdgeAppStopped) && len(appContainers) != len(manif.Manifest.Modules) {
			edgeApplication.Status = model.EdgeAppError
		}

		containersStat := []com.ContainerMsg{}
		for _, con := range appContainers {
			containerJSON, err := docker.InspectContainer(con.ID)
			if err != nil {
				return edgeApps, err
			}
			// The Status of each container is (assumed to be): Running, Restarting, Created, Exited
			container := com.ContainerMsg{Name: strings.Join(con.Names, ", "), Status: ioutility.FirstToUpper(con.State)}
			containersStat = append(containersStat, container)

			if (manif.Status != model.EdgeAppInitiated && manif.Status != model.EdgeAppExecuting) && edgeApplication.Status != model.EdgeAppError {
				if manif.Status == model.EdgeAppRunning && con.State != strings.ToLower(model.ModuleRunning) {
					edgeApplication.Status = model.EdgeAppError
				}
				if manif.Status == model.EdgeAppStopped && (con.State != strings.ToLower(model.ModuleExited) || containerJSON.State.ExitCode != 0) {
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
