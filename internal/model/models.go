package model

import (
	jwt "github.com/dgrijalva/jwt-go"
)

type ErrorResponse struct {
	Err string
}

type Exception struct {
	Message string `json:"message"`
}

type User struct {
	ID       uint
	Name     string
	Email    string `json:"Email:varchar(100):unique_index"`
	Password string `json:"Password"`
}

type Token struct {
	UserID uint
	Name   string
	Email  string
	*jwt.StandardClaims
}

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
