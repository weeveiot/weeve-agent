package com

import (
	"time"

	"github.com/weeveiot/weeve-agent/internal/model"
)

type EdgeAppLogMsg struct {
	ManifestID  string    `json:"manifestID"`
	ContainerID string    `json:"containerID"`
	ModuleName  string    `json:"moduleName"`
	Time        time.Time `json:"time"`
	Level       string    `json:"level"`
	Filename    string    `json:"filename"`
	Message     string    `json:"message"`
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
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type StatusMsg struct {
	Status           string          `json:"status"`
	EdgeApplications []EdgeAppMsg    `json:"edgeApplications"`
	DeviceParams     DeviceParamsMsg `json:"deviceParams"`
	AgentVersion     string          `json:"agentVersion"`
	OrgKeyHash       string          `json:"orgKeyHash"`
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
