package manifest

type manifestMsg struct {
	ID            string `json:"_id"`
	ManifestName  string
	VersionNumber float64
	Connections   connectionsString
	Modules       []moduleMsg
	Command       string
}

type moduleMsg struct {
	ModuleName string
	Image      imageMsg
	Envs       []envMsg
	Ports      []portMsg
	Mounts     []mountMsg
	Devices    []deviceMsg
	Type       string
}

type envMsg struct {
	Key   string
	Value string
}

type portMsg struct {
	Container string
	Host      string
}

type mountMsg struct {
	Container string
	Host      string
}

type deviceMsg struct {
	Container string
	Host      string
}

type imageMsg struct {
	Name     string
	Tag      string
	Registry registryMsg
}

type registryMsg struct {
	Url      string
	UserName string
	Password string
}

type uniqueIDmsg struct {
	ManifestName  string
	VersionNumber float64
}
