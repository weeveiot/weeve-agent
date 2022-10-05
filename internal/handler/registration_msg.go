package handler

import "time"

type registrationMessage struct {
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Name      string `json:"name"`
}

func GetRegistrationMessage(nodeId string, nodeName string) registrationMessage {
	msg := registrationMessage{
		Id:        nodeId,
		Timestamp: time.Now().UnixMilli(),
		Status:    "Registering",
		Operation: "Registration",
		Name:      nodeName,
	}
	return msg
}
