package edgeapp

import (
	"encoding/json"
	"time"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
	traceutility "github.com/weeveiot/weeve-agent/internal/utility/trace"
)

type logTimestamp time.Time

type dockerLogLine struct {
	Time     logTimestamp `json:"timestamp"`
	Level    string       `json:"level"`
	Filename string       `json:"filename"`
	Message  string       `json:"message"`
}

func (t *logTimestamp) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsedTime, err := time.Parse(time.DateTime, s)
	if err != nil {
		return err
	}
	*t = logTimestamp(parsedTime)
	return nil
}

func SendEdgeAppLogs(manif manifest.ManifestRecord, until string) error {
	msg, err := GetEdgeAppLogsMsg(manif, until)
	if err != nil {
		return traceutility.Wrap(err)
	}
	err = com.SendEdgeAppLogs(msg)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return manifest.SetLastLogRead(manif.Manifest.ManifestUniqueID, until)
}

func GetEdgeAppLogsMsg(manif manifest.ManifestRecord, until string) (com.EdgeAppLogMsg, error) {
	var msg com.EdgeAppLogMsg
	containerLogs, err := GetEdgeAppLogs(manif.Manifest.ManifestUniqueID, manif.LastLogReadTime, until)
	if err != nil {
		return msg, traceutility.Wrap(err)
	}

	msg = com.EdgeAppLogMsg{
		ManifestID:    manif.Manifest.ID,
		ContainerLogs: containerLogs,
	}

	return msg, nil
}

func GetEdgeAppLogs(uniqueID model.ManifestUniqueID, since string, until string) ([]com.ContainerLogMsg, error) {
	var containerLogs []com.ContainerLogMsg

	appContainers, err := docker.ReadEdgeAppContainers(uniqueID)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	for _, container := range appContainers {
		dockerLogs := com.ContainerLogMsg{ContainerID: container.ID}
		logs, err := docker.ReadContainerLogs(container.ID, since, until)
		if err != nil {
			return nil, traceutility.Wrap(err)
		}
		dockerLogs.Log = constructLogEntry(logs, since, until)

		if len(dockerLogs.Log) > 0 {
			containerLogs = append(containerLogs, dockerLogs)
		}
	}

	return containerLogs, nil
}

func constructLogEntry(logLines []string, since string, until string) []com.ContainerLogLineMsg {
	defaultTime := meanTime(since, until)
	var logMsgs []com.ContainerLogLineMsg

	for _, line := range logLines {
		// * default log message
		logMsg := com.ContainerLogLineMsg{
			Time:     defaultTime,
			Level:    "DEBUG",
			Filename: "unknown",
			Message:  line,
		}

		// try to extract log message from json
		if parsedLog, ok := parseJSONLogLine(line); ok {
			if !parsedLog.Time.IsZero() {
				logMsg.Time = parsedLog.Time
			}
			if parsedLog.Level != "" {
				logMsg.Level = parsedLog.Level
			}
			if parsedLog.Filename != "" {
				logMsg.Filename = parsedLog.Filename
			}
			if parsedLog.Message != "" {
				logMsg.Message = parsedLog.Message
			}
		}

		logMsgs = append(logMsgs, logMsg)
	}

	return logMsgs
}

func parseJSONLogLine(line string) (com.ContainerLogLineMsg, bool) {
	docLog := dockerLogLine{}
	logMsg := com.ContainerLogLineMsg{}
	logLine := []byte(line)

	err := json.Unmarshal(logLine, &docLog)
	if err != nil {
		return logMsg, false
	}

	logMsg.Time = time.Time(docLog.Time)
	logMsg.Level = docLog.Level
	logMsg.Filename = docLog.Filename
	logMsg.Message = docLog.Message

	return logMsg, true
}

func meanTime(first, second string) time.Time {
	firstTime, _ := time.Parse(time.RFC3339Nano, first)
	secondTime, _ := time.Parse(time.RFC3339Nano, second)
	return firstTime.Add(secondTime.Sub(firstTime) / 2)
}
