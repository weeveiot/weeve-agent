package com

import (
	log "github.com/sirupsen/logrus"

	"github.com/weeveiot/weeve-agent/internal/config"
)

type mqttHook struct {
	levels []log.Level
	topic  string
}

func addMqttHookToLogs(level log.Level) {
	log.Debug("Adding MQTT hook to logs...")

	hook := mqttHook{
		levels: log.AllLevels[:level+1],
		topic:  topicAgentLogs + "/" + config.Params.NodeId,
	}

	log.AddHook(hook)
	log.Debug("MQTT hook to send agent's logs to MAPI is set up.")
}

// Fire sends logs over MQTT to MAPI
func (hook mqttHook) Fire(entry *log.Entry) error {
	msg := agentLogMsg{
		Time:    entry.Time.UTC(),
		Level:   entry.Level.String(),
		Message: entry.Message,
	}

	return publishMessage(hook.topic, msg, false)
}

// Levels returns the list of logging levels that will trigger Fire
func (hook mqttHook) Levels() []log.Level {
	return hook.levels
}
