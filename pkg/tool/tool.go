package tool

import (
	"slices"
	"strings"

	"gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/log"

	"github.com/mark3labs/mcp-go/server"
)

type Tool struct {
	write []server.ServerTool
	read  []server.ServerTool
}

func New() *Tool {
	return &Tool{
		write: make([]server.ServerTool, 0, 100),
		read:  make([]server.ServerTool, 0, 100),
	}
}

func (t *Tool) RegisterWrite(s server.ServerTool) {
	t.write = append(t.write, s)
}

func (t *Tool) RegisterRead(s server.ServerTool) {
	t.read = append(t.read, s)
}

// ReadTools returns the read-only tools registered on this domain, ignoring
// the read-only and allowlist flags that Tools applies.
func (t *Tool) ReadTools() []server.ServerTool {
	return t.read
}

// WriteTools returns the write tools registered on this domain, ignoring the
// read-only and allowlist flags that Tools applies.
func (t *Tool) WriteTools() []server.ServerTool {
	return t.write
}

func (t *Tool) Tools() []server.ServerTool {
	all := make([]server.ServerTool, 0, len(t.write)+len(t.read))
	if !flag.ReadOnly {
		all = append(all, t.write...)
	}
	all = append(all, t.read...)
	if len(flag.AllowedTools) == 0 {
		return all
	}
	filtered := make([]server.ServerTool, 0, len(all))
	for _, st := range all {
		if _, ok := flag.AllowedTools[st.Tool.Name]; ok {
			filtered = append(filtered, st)
		}
	}
	return filtered
}

// WarnUnmatchedAllowedTools logs any names in flag.AllowedTools that don't
// match a tool registered on any of the given domains. No-op if the allowlist
// is empty.
func WarnUnmatchedAllowedTools(domains ...*Tool) {
	if len(flag.AllowedTools) == 0 {
		return
	}
	known := map[string]struct{}{}
	for _, d := range domains {
		for _, st := range d.read {
			known[st.Tool.Name] = struct{}{}
		}
		for _, st := range d.write {
			known[st.Tool.Name] = struct{}{}
		}
	}
	var unmatched []string
	for name := range flag.AllowedTools {
		if _, ok := known[name]; !ok {
			unmatched = append(unmatched, name)
		}
	}
	if len(unmatched) == 0 {
		return
	}
	slices.Sort(unmatched)
	log.Warnf("Unknown tools in --tools allowlist (ignored): %s", strings.Join(unmatched, ", "))
}
