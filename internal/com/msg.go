package com

import (
	"time"

	"github.com/weeveiot/weeve-agent/internal/model"
)

type ContainerLogLineMsg struct {
	Level    string    `json:"level"`
	Time     time.Time `json:"time"`
	Filename string    `json:"filename"`
	Message  string    `json:"message"`
}

type ContainerLogMsg struct {
	ContainerID string                `json:"containerID"`
	Log         []ContainerLogLineMsg `json:"log"`
}

type EdgeAppLogMsg struct {
	ManifestID    string            `json:"manifestID"`
	ContainerLogs []ContainerLogMsg `json:"containerLog"`
}

type ContainerMsg struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type EdgeAppMsg struct {
	ManifestID string         `json:"manifestID"`
	Status     string         `json:"status"`
	Containers []ContainerMsg `json:"containers"`
}

type agentLogMsg struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
}

type StatusMsg struct {
	Status           string          `json:"status"`
	EdgeApplications []EdgeAppMsg    `json:"edgeApplications"`
	DeviceParams     DeviceParamsMsg `json:"deviceParams"`
	AgentVersion     string          `json:"agentVersion"`
}

type DeviceParamsMsg struct {
	SystemUpTime uint64  `json:"systemUpTime"`
	SystemLoad   float64 `json:"systemLoad"`
	StorageFree  float64 `json:"storageFree"`
	RamFree      float64 `json:"ramFree"`
}

type nodePublicKeyMsg struct {
	NodePublicKey string `json:"nodePublicKey"`
}

var disconnectedMsg = StatusMsg{
	Status:           model.NodeDisconnected,
	EdgeApplications: nil,
	DeviceParams: DeviceParamsMsg{
		SystemUpTime: 0,
		SystemLoad:   0,
		StorageFree:  100,
		RamFree:      100,
	},
	AgentVersion: model.Version,
}
