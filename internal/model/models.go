package model

type Params struct {
	Verbose      bool   `long:"verbose" short:"v" description:"Show verbose debug information"`
	Broker       string `long:"broker" short:"b" description:"Broker to connect" required:"true"`
	PubClientId  string `long:"pubClientId" short:"c" description:"Publisher ClientId" required:"true"`
	SubClientId  string `long:"subClientId" short:"s" description:"Subscriber ClientId" required:"true"`
	TopicName    string `long:"publish" short:"t" description:"Topic Name" required:"true"`
	Heartbeat    int    `long:"heartbeat" short:"h" description:"Heartbeat time in seconds" required:"false" default:"30"`
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
	CertPath     string `long:"cert" short:"f" description:"Path to certificate to authenticate to Broker" required:"false"`
	KeyPath      string `long:"key" short:"k" description:"Path to private key to authenticate to Broker" required:"false"`
	ConfigPath   string `long:"config" description:"Path to the .json config file" required:"false"`
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
