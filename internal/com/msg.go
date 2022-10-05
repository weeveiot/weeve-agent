package com

import "github.com/weeveiot/weeve-agent/internal/docker"

type EdgeAppLogMsg struct {
	ManifestID    string                `json:"manifestID"`
	ContainerLogs []docker.ContainerLog `json:"containerLog"`
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

type StatusMsg struct {
	Status           string          `json:"status"`
	EdgeApplications []EdgeAppMsg    `json:"edgeApplications"`
	DeviceParams     DeviceParamsMsg `json:"deviceParams"`
}

type DeviceParamsMsg struct {
	SystemUpTime uint64  `json:"systemUpTime"`
	SystemLoad   float64 `json:"systemLoad"`
	StorageFree  float64 `json:"storageFree"`
	RamFree      float64 `json:"ramFree"`
}

type nodePublicKeyMsg struct {
	NodePublicKey string
}
