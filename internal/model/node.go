package model

import "strings"

var Version string = "YYY.MM.DD (commit hash)"

type Params struct {
	Version      bool   `long:"version" short:"v" description:"Print version information and exit"`
	Broker       string `long:"broker" short:"b" description:"Broker to connect"`
	NodeId       string `long:"id" short:"i" description:"ID of this node"`
	NodeName     string `long:"name" short:"n" description:"Name of this node to be registered"`
	NoTLS        bool   `long:"notls" description:"For developer - disable TLS for MQTT"`
	Password     string `long:"password" description:"Password for TLS"`
	RootCertPath string `long:"rootcert" description:"Path to MQTT broker (server) certificate"`
	LogLevel     string `long:"loglevel" short:"l" description:"Set the logging level"`
	LogFileName  string `long:"logfilename" description:"Set the name of the log file"`
	LogSize      int    `long:"logsize" description:"Set the size of each log files (MB)"`
	LogAge       int    `long:"logage" description:"Set the time period to retain the log files (days)"`
	LogBackup    int    `long:"logbackup" description:"Set the max number of log files to retain"`
	LogCompress  bool   `long:"logcompress" description:"To compress the log files"`
	MqttLogs     bool   `long:"mqttlogs" description:"For developer - Display detailed MQTT logging messages"`
	Heartbeat    int    `long:"heartbeat" short:"t" description:"Heartbeat time in seconds" `
	LogSendInvl  int    `long:"logsendinvl" description:"Time interval in sec to send edge app logs" `
	Stdout       bool   `long:"out" description:"Print logs to stdout"`
	ConfigPath   string `long:"config" description:"Path to the .json config file"`
	ManifestPath string `long:"manifest" description:"Path to the .json manifest file"`
	Delete       bool   `long:"delete" short:"d" description:"Remove node from weeve manager (when uninstalling the agent)"`
}

type ManifestUniqueID struct {
	VersionNumber string
	ManifestName  string
}

func (uniqueID ManifestUniqueID) MarshalText() (text []byte, err error) {
	return []byte(uniqueID.ManifestName + "+" + uniqueID.VersionNumber), nil
}

func (uniqueID *ManifestUniqueID) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), "+")
	uniqueID.ManifestName = parts[0]
	uniqueID.VersionNumber = parts[1]
	return nil
}

const (
	NodeConnected    = "Connected"
	NodeDisconnected = "Disconnected"
	NodeDeleted      = "Deleted"
)

const (
	EdgeAppRunning    = "Running"
	EdgeAppStopped    = "Stopped"
	EdgeAppError      = "Error"
	EdgeAppInitiated  = "Initiated"
	EdgeAppExecuting  = "Executing"
	EdgeAppUndeployed = "Undeployed"
)

const (
	ModuleRunning    = "Running"
	ModuleRestarting = "Restarting"
	ModuleCreated    = "Created"
	ModuleExited     = "Exited"
)
