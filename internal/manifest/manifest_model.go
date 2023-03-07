package manifest

type manifestMsg struct {
	ID            string `json:"_id" validate:"required,notblank,alphanum"`
	ManifestName  string `validate:"required,notblank"`
	UpdatedAt     string `validate:"required,notblank"`
	VersionNumber float64
	Connections   connectionsString `validate:"required"`
	Modules       []moduleMsg       `validate:"required,notblank"`
	Command       string            `validate:"required,notblank"`
	DebugMode     bool
}

type moduleMsg struct {
	ModuleName string   `validate:"required,notblank"`
	Image      imageMsg `validate:"required"`
	Envs       []envMsg
	Ports      []portMsg
	Mounts     []mountMsg
	Devices    []deviceMsg
	Type       string `validate:"required,notblank"`
}

type envMsg struct {
	Key    string `validate:"required,notblank"`
	Value  string `validate:"required"`
	Secret bool   `validate:"required"`
}

type portMsg struct {
	Container string `validate:"required,notblank"`
	Host      string `validate:"required,notblank"`
}

type mountMsg struct {
	Container string `validate:"required,notblank"`
	Host      string `validate:"required,notblank"`
}

type deviceMsg struct {
	Container string `validate:"required,notblank"`
	Host      string `validate:"required,notblank"`
}

type imageMsg struct {
	Name     string `validate:"required,notblank"`
	Tag      string
	Registry registryMsg
}

type registryMsg struct {
	Url      string `validate:"required,notblank"`
	UserName string
	Password string
}

type uniqueIDmsg struct {
	ID            string `json:"_id" validate:"required,notblank,alphanum"`
	ManifestName  string
	UpdatedAt     string
	VersionNumber float64
}

type commandMsg struct {
	Command string `validate:"required,notblank"`
}
