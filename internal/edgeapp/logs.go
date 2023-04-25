package edgeapp

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/weeveiot/weeve-agent/internal/com"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/manifest"
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
	msg, err := GetEdgeAppLogs(manif, until)
	if err != nil {
		return traceutility.Wrap(err)
	}
	err = com.SendEdgeAppLogs(msg)
	if err != nil {
		return traceutility.Wrap(err)
	}

	return manifest.SetLastLogRead(manif.Manifest.UniqueID, until)
}

func GetEdgeAppLogs(manif manifest.ManifestRecord, until string) ([]com.EdgeAppLogMsg, error) {
	var edgeAppLogs []com.EdgeAppLogMsg

	appContainers, err := docker.ReadEdgeAppContainers(manif.Manifest.UniqueID)
	if err != nil {
		return nil, traceutility.Wrap(err)
	}

	for _, container := range appContainers {
		logs, err := docker.ReadContainerLogs(container.ID, manif.LastLogReadTime, until)
		if err != nil {
			return nil, traceutility.Wrap(err)
		}
		logMsgs := constructLogEntry(manif.Manifest.ID, container.ID, strings.Split(container.Image, "/")[1], logs, manif.LastLogReadTime, until)

		if len(logMsgs) > 0 {
			edgeAppLogs = append(edgeAppLogs, logMsgs...)
		}
	}

	return edgeAppLogs, nil
}

func constructLogEntry(manifestID string, containerID string, moduleName string, logLines []string, since string, until string) []com.EdgeAppLogMsg {
	defaultTime := meanTime(since, until)
	var logMsgs []com.EdgeAppLogMsg

	for _, line := range logLines {
		// * default log message
		logMsg := com.EdgeAppLogMsg{
			ManifestID:  manifestID,
			ContainerID: containerID,
			ModuleName:  moduleName,
			Time:        defaultTime,
			Level:       "DEBUG",
			Filename:    "unknown",
			Message:     line,
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

func parseJSONLogLine(line string) (com.EdgeAppLogMsg, bool) {
	docLog := dockerLogLine{}
	logMsg := com.EdgeAppLogMsg{}
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
