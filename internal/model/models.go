package model

type StatusMessage struct {
	Id                 string           `json:"ID"`
	Timestamp          int64            `json:"timestamp"`
	Status             string           `json:"status"`
	ActiveServiceCount int              `json:"activeServiceCount"`
	ServiceCount       int              `json:"serviceCount"`
	ServicesStatus     []ManifestStatus `json:"servicesStatus"`
	DeviceParams       DeviceParams     `json:"deviceParams"`
}

type ManifestStatus struct {
	ManifestId      string `json:"manifestId"`
	ManifestVersion string `json:"manifestVersion"`
	Status          string `json:"status"`
}

type RegistrationMessage struct {
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Name      string `json:"name"`
}
type DeviceParams struct {
	Sensors string `json:"sensors"`
	Uptime  string `json:"uptime"`
	CpuTemp string `json:"cputemp"`
}
