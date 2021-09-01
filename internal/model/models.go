package model

import jwt "github.com/dgrijalva/jwt-go"

type ErrorResponse struct {
	Err string
}

type Exception struct {
	Message string `json:"message"`
}

type User struct {
	ID       uint
	Name     string
	Email    string `json:"Email:varchar(100);unique_index"`
	Password string `json:"Password"`
}

type Token struct {
	UserID uint
	Name   string
	Email  string
	*jwt.StandardClaims
}

/*
type ManifestReq struct {
	ID      string     `json:"ID"`
	Name    string     `json:"Name"`
	Modules []Manifest `json:"Modules"`
}
*/

/*
type Manifest struct {
	Index     int      `json:"Index"`
	Name      string   `json:"Name"`
	Tag       string   `json:"Tag"`
	ImageID   string   `json:"ImageID"`
	ImageName string   `json:"ImageName"`
	Options   Option   `json:"Options"`
	Arguments Argument `json:"Arguments"`
}
*/

type Argument []struct {
	Arg string `json:"opt"`
	Val bool   `json:"val"`
}

type Option []struct {
	Arg string `json:"arg"`
	Val bool   `json:"val"`
}

type StatusMessage struct {
	Id                 string           `json:"ID"`
	Timestamp          int64            `json:"timestamp"`
	Connectivity       string           `json:"connectivity"`
	ActiveServiceCount int              `json:"activeServiceCount"`
	ServiceCount       int              `json:"serviceCount"`
	DeployStatus       []ManifestStatus `json:"deployStatus"`
	DeviceParams       DeviceParams     `json:"deviceParams"`
}

type ManifestStatus struct {
	ManifestId      string `json:"manifestId"`
	ManifestVersion string `json:"manifestVersion"`
	Status          string `json:"status"`
}

type DeviceParams struct {
	Sensors string `json:"sensors"`
	Uptime  string `json:"uptime"`
	CpuTemp string `json:"cputemp"`
}
