package cmd

import (
	"bytes"
	"flag"
	"maps"
	"slices"
	"testing"

	flagPkg "gitea.com/gitea/gitea-mcp/pkg/flag"
)

func TestInitFlagSetBind(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "default is empty, meaning all interfaces", args: []string{}},
		{name: "-b sets the address", args: []string{"-b", "127.0.0.1"}, want: "127.0.0.1"},
		{name: "-bind sets an IPv6 literal", args: []string{"-bind", "::1"}, want: "::1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(func() { flagPkg.Bind = "" })
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			initFlagSet(fs, test.args, func(string) string { return "" }, func(string) ([]byte, error) { return nil, nil }, &bytes.Buffer{})
			if flagPkg.Bind != test.want {
				t.Errorf("Bind = %q, want %q", flagPkg.Bind, test.want)
			}
		})
	}
}

func TestInitFlagSetOAuth(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		env      map[string]string
		want     bool
		wantURL  string
		wantExit bool
	}{
		{name: "off by default"},
		{
			name:    "flags enable it and drop the static token",
			args:    []string{"-t", "http", "-oauth", "-oauth-public-url", "https://mcp.example.com", "-T", "statictoken"},
			want:    true,
			wantURL: "https://mcp.example.com",
		},
		{
			name:    "env enables it and trims the trailing slash",
			args:    []string{"-t", "http"},
			env:     map[string]string{"GITEA_MCP_OAUTH": "true", "GITEA_MCP_PUBLIC_URL": "https://mcp.example.com/"},
			want:    true,
			wantURL: "https://mcp.example.com",
		},
		{name: "public URL is required", args: []string{"-t", "http", "-oauth"}, wantExit: true},
		{name: "public URL must not have a path", args: []string{"-t", "http", "-oauth", "-oauth-public-url", "https://mcp.example.com/base"}, wantExit: true},
		{name: "stdio transport is rejected", args: []string{"-oauth", "-oauth-public-url", "https://mcp.example.com"}, wantExit: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			origExit := osExit
			t.Cleanup(func() {
				osExit = origExit
				flagPkg.OAuth, flagPkg.OAuthPublicURL, flagPkg.Token = false, "", ""
			})
			var exits int
			osExit = func(int) { exits++ }

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			initFlagSet(fs, test.args, func(key string) string { return test.env[key] }, func(string) ([]byte, error) { return nil, nil }, &bytes.Buffer{})

			if (exits > 0) != test.wantExit {
				t.Fatalf("exits = %d, want exit %v", exits, test.wantExit)
			}
			if test.wantExit {
				return // state after a rejected flag set is not reached in the real binary
			}
			if flagPkg.OAuth != test.want {
				t.Errorf("OAuth = %v, want %v", flagPkg.OAuth, test.want)
			}
			if flagPkg.OAuthPublicURL != test.wantURL {
				t.Errorf("OAuthPublicURL = %q, want %q", flagPkg.OAuthPublicURL, test.wantURL)
			}
			if test.want && flagPkg.Token != "" {
				t.Errorf("Token = %q, want it dropped in OAuth mode", flagPkg.Token)
			}
		})
	}
}

func TestInitFlagSetScopes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want []string
	}{
		{
			name: "no scope flag or env leaves AllowedScopes unset",
			args: []string{},
			want: nil,
		},
		{
			name: "-S sets a single scope",
			args: []string{"-S", "repository"},
			want: []string{"repository"},
		},
		{
			name: "-scope sets a comma-separated list",
			args: []string{"-scope", "repository,file"},
			want: []string{"file", "repository"},
		},
		{
			name: "GITEA_SCOPES env sets the default",
			args: []string{},
			env:  map[string]string{"GITEA_SCOPES": "issue,pull_request"},
			want: []string{"issue", "pull_request"},
		},
		{
			name: "-S flag takes precedence over GITEA_SCOPES env",
			args: []string{"-S", "file"},
			env:  map[string]string{"GITEA_SCOPES": "issue"},
			want: []string{"file"},
		},
		{
			name: "normalizes case, whitespace, and hyphens/spaces to underscores",
			args: []string{"-S", " Pull Request , pull-request , PULL_REQUEST "},
			want: []string{"pull_request"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origScopes := flagPkg.AllowedScopes
			t.Cleanup(func() {
				flagPkg.AllowedScopes = origScopes
			})
			flagPkg.AllowedScopes = nil

			getenv := func(key string) string { return tt.env[key] }
			readFile := func(string) ([]byte, error) { return nil, nil }
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			var stderr bytes.Buffer

			initFlagSet(fs, tt.args, getenv, readFile, &stderr)

			got := slices.Sorted(maps.Keys(flagPkg.AllowedScopes))
			want := slices.Clone(tt.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("AllowedScopes = %v, want %v", got, want)
			}
		})
	}
}

func TestInitFlagSetHealthcheck(t *testing.T) {
	t.Cleanup(func() { healthcheck = false })
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	initFlagSet(fs, []string{"-healthcheck"}, func(string) string { return "" }, func(string) ([]byte, error) { return nil, nil }, &bytes.Buffer{})
	if !healthcheck {
		t.Error("healthcheck = false, want true")
	}
}
