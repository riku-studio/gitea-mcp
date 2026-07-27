package operation

import (
	"testing"

	"gitea.com/gitea/gitea-mcp/pkg/flag"
)

// TestAllToolsHaveDescriptions ensures every registered tool sets a non-empty
// Tool.Description. mcp-go only serializes the "description" field of a tool
// when it is non-empty, so an omitted description makes strict MCP clients
// (e.g. mcp-probe) reject the tools/list response with "missing field
// `description`".
func TestAllToolsHaveDescriptions(t *testing.T) {
	origRO, origAllow := flag.ReadOnly, flag.AllowedTools
	t.Cleanup(func() {
		flag.ReadOnly, flag.AllowedTools = origRO, origAllow
	})
	flag.ReadOnly = false
	flag.AllowedTools = nil

	var missing []string
	for _, d := range domainTools {
		for _, st := range d.Tools() {
			if st.Tool.Description == "" {
				missing = append(missing, st.Tool.Name)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("tools missing a description: %v", missing)
	}
}

// TestDomainToolsScopesAreUniqueAndNonEmpty ensures every entry registered in
// domainTools has a canonical, non-empty scope name and that no two domains
// share the same scope (each domain.Tools() call is filtered by exactly one
// scope name via flag.AllowedScopes).
func TestDomainToolsScopesAreUniqueAndNonEmpty(t *testing.T) {
	seen := map[string]struct{}{}
	for _, d := range domainTools {
		scope := d.Scope()
		if scope == "" {
			t.Errorf("domainTools contains a domain with an empty scope")
			continue
		}
		if _, ok := seen[scope]; ok {
			t.Errorf("domainTools contains a duplicate scope %q", scope)
			continue
		}
		seen[scope] = struct{}{}
	}
}

func TestParseAuthToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{
			name:      "valid Bearer token",
			header:    "Bearer validtoken",
			wantToken: "validtoken",
			wantOK:    true,
		},
		{
			name:      "lowercase bearer",
			header:    "bearer lowercase",
			wantToken: "lowercase",
			wantOK:    true,
		},
		{
			name:      "uppercase BEARER",
			header:    "BEARER uppercase",
			wantToken: "uppercase",
			wantOK:    true,
		},
		{
			name:      "token with spaces trimmed",
			header:    "Bearer   spacedToken ",
			wantToken: "spacedToken",
			wantOK:    true,
		},
		{
			name:      "bearer with no token",
			header:    "Bearer ",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "bearer with only spaces",
			header:    "Bearer     ",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "missing space after Bearer",
			header:    "Bearertoken",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "Gitea token format",
			header:    "token giteaapitoken",
			wantToken: "giteaapitoken",
			wantOK:    true,
		},
		{
			name:      "Gitea Token format capitalized",
			header:    "Token giteaapitoken",
			wantToken: "giteaapitoken",
			wantOK:    true,
		},
		{
			name:      "token with no value",
			header:    "token ",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "different auth type",
			header:    "Basic dXNlcjpwYXNz",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "empty header",
			header:    "",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "bearer token with internal spaces",
			header:    "Bearer token with spaces",
			wantToken: "token with spaces",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotOK := parseAuthToken(tt.header)
			if gotToken != tt.wantToken {
				t.Errorf("parseAuthToken() token = %q, want %q", gotToken, tt.wantToken)
			}
			if gotOK != tt.wantOK {
				t.Errorf("parseAuthToken() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}
