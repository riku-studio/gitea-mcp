package flag

var (
	Host    string
	Port    int
	Token   string
	Version string
	Mode    string

	Insecure     bool
	ReadOnly     bool
	Debug        bool
	AllowedTools map[string]struct{}
)
