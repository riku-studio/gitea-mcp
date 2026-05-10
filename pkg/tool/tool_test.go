package tool

import (
	"slices"
	"testing"

	"gitea.com/gitea/gitea-mcp/pkg/flag"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func makeTool(name string) server.ServerTool {
	return server.ServerTool{Tool: mcp.NewTool(name)}
}

func names(sts []server.ServerTool) []string {
	out := make([]string, len(sts))
	for i, st := range sts {
		out[i] = st.Tool.Name
	}
	return out
}

func TestTools(t *testing.T) {
	tests := []struct {
		name     string
		readOnly bool
		allowed  map[string]struct{}
		read     []string
		write    []string
		want     []string
	}{
		{
			name:  "no filters returns write then read",
			read:  []string{"r1", "r2"},
			write: []string{"w1", "w2"},
			want:  []string{"w1", "w2", "r1", "r2"},
		},
		{
			name:     "read-only excludes write",
			readOnly: true,
			read:     []string{"r1", "r2"},
			write:    []string{"w1"},
			want:     []string{"r1", "r2"},
		},
		{
			name:    "allowlist keeps only listed",
			allowed: map[string]struct{}{"r1": {}, "w1": {}},
			read:    []string{"r1", "r2"},
			write:   []string{"w1", "w2"},
			want:    []string{"w1", "r1"},
		},
		{
			name:     "allowlist intersected with read-only drops write entries",
			readOnly: true,
			allowed:  map[string]struct{}{"r1": {}, "w1": {}},
			read:     []string{"r1", "r2"},
			write:    []string{"w1", "w2"},
			want:     []string{"r1"},
		},
		{
			name:    "allowlist with only unknown names returns empty",
			allowed: map[string]struct{}{"unknown": {}},
			read:    []string{"r1"},
			write:   []string{"w1"},
			want:    []string{},
		},
		{
			name:    "empty allowlist map passes through",
			allowed: map[string]struct{}{},
			read:    []string{"r1"},
			write:   []string{"w1"},
			want:    []string{"w1", "r1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origRO, origAllow := flag.ReadOnly, flag.AllowedTools
			t.Cleanup(func() {
				flag.ReadOnly, flag.AllowedTools = origRO, origAllow
			})
			flag.ReadOnly = tt.readOnly
			flag.AllowedTools = tt.allowed

			tr := New()
			for _, n := range tt.read {
				tr.RegisterRead(makeTool(n))
			}
			for _, n := range tt.write {
				tr.RegisterWrite(makeTool(n))
			}

			got := names(tr.Tools())
			if !slices.Equal(got, tt.want) {
				t.Errorf("Tools() = %v, want %v", got, tt.want)
			}
		})
	}
}
