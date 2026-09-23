package flag

var (
	Host    string
	Bind    string
	Port    int
	Token   string
	Version string
	Mode    string

	MaxInlineAttachmentBytes int

	OAuth          bool
	OAuthPublicURL string // origin, no trailing slash

	Insecure      bool
	ReadOnly      bool
	Debug         bool
	AllowedTools  map[string]struct{}
	AllowedScopes map[string]struct{}
)
