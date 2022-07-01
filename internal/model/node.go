package model

type Params struct {
	Stdout       bool   `long:"out" description:"Print logs to stdout"`
	Broker       string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	Heartbeat    int    `long:"heartbeat" short:"t" description:"Heartbeat time in seconds" required:"false" default:"30"`
	MqttLogs     bool   `long:"mqttlogs" short:"m" description:"For developer - Display detailed MQTT logging messages" required:"false"`
	NoTLS        bool   `long:"notls" description:"For developer - disable TLS for MQTT" required:"false"`
	LogLevel     string `long:"loglevel" short:"l" default:"info" description:"Set the logging level" required:"false"`
	LogFileName  string `long:"logfilename" default:"Weeve_Agent.log" description:"Set the name of the log file" required:"false"`
	LogSize      int    `long:"logsize" default:"1" description:"Set the size of each log files (MB)" required:"false"`
	LogAge       int    `long:"logage" default:"1" description:"Set the time period to retain the log files (days)" required:"false"`
	LogBackup    int    `long:"logbackup" default:"5" description:"Set the max number of log files to retain" required:"false"`
	LogCompress  bool   `long:"logcompress" description:"To compress the log files" required:"false"`
	NodeId       string `long:"nodeId" short:"i" description:"ID of this node" required:"false" default:""`
	NodeName     string `long:"name" short:"n" description:"Name of this node to be registered" required:"false"`
	RootCertPath string `long:"rootcert" short:"r" description:"Path to MQTT broker (server) certificate" required:"false"`
	ConfigPath   string `long:"config" description:"Path to the .json config file" required:"false"`
	ManifestPath string `long:"manifest" description:"Path to the .json manifest file" required:"false"`
}

type ManifestStatus struct {
	ManifestID       string           `json:"manifestID"`
	ManifestUniqueID ManifestUniqueID `json:"manifestUniqueID"`
	Status           string           `json:"status"`
	ContainerCount   int              `json:"containerCount"`
	InTransition     bool             `json:"inTransition"`
}

type ManifestUniqueID struct {
	VersionNumber string
	ManifestName  string
}

const (
	NodeConnected = "Connected"
	NodeAlarm     = "Alarm"
)

const (
	EdgeAppRunning = "Running"
	EdgeAppPaused  = "Paused"
	EdgeAppError   = "Error"
	EdgeAppIninted = "Initiated"
	EdgeAppDeleted = "Deleted"
)

const (
	ModuleRunning    = "Running"
	ModulePaused     = "Paused"
	ModuleRestarting = "Restarting"
	ModuleCreated    = "Created"
	ModuleExited     = "Exited"
)
