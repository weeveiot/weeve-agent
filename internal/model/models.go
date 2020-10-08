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

type Manifest struct {
	Index     int              `json:"Index"`
	Tag       string           `json:"Tag"`
	ImageID   string           `json:"ImageID"`
	ImageName string           `json:"ImageID"`
	State     string           `json:"State"`
	Ingress   DatasourceConfig `json:"Ingress"`
	Egress    DatasourceConfig `json:"Egress"`
}

type DatasourceConfig struct {
	Type     string `json:"Type"`
	Protocol string `json:"Protocol"`
	Port     string `json:"Port"`
	Param1   string `json:"Param1"`
	Param2   string `json:"Param2"`
	Param3   string `json:"Param3"`
	Param4   string `json:"Param4"`
	Param5   string `json:"Param5"`
	Param6   string `json:"Param6"`
}
