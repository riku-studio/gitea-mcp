package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"gitea.com/gitea/gitea-mcp/operation"
	flagPkg "gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/log"
)

var (
	host                            string
	bind                            string
	port                            int
	token                           string
	tools                           string
	scopes                          string
	version                         bool
	healthcheck                     bool
	maxInlineAttachmentBytes        int
	maxInlineAttachmentBytesFlagSet bool
	osExit                          = os.Exit
	healthcheckClient               = http.DefaultClient
)

func initFlagSet(fs *flag.FlagSet, args []string, getenv func(string) string, readFile func(string) ([]byte, error), stderr io.Writer) {
	fs.StringVar(&flagPkg.Mode, "t", "stdio", "")
	fs.StringVar(&flagPkg.Mode, "transport", "stdio", "")
	fs.StringVar(&host, "H", getenv("GITEA_HOST"), "")
	fs.StringVar(&host, "host", getenv("GITEA_HOST"), "")
	fs.StringVar(&bind, "b", "", "")
	fs.StringVar(&bind, "bind", "", "")
	fs.IntVar(&port, "p", 8080, "")
	fs.IntVar(&port, "port", 8080, "")
	fs.StringVar(&token, "T", "", "")
	fs.StringVar(&token, "token", "", "")
	fs.BoolVar(&flagPkg.ReadOnly, "r", false, "")
	fs.BoolVar(&flagPkg.ReadOnly, "read-only", false, "")
	defaultTools := getenv("GITEA_TOOLS")
	fs.StringVar(&tools, "O", defaultTools, "")
	fs.StringVar(&tools, "tools", defaultTools, "")
	defaultScopes := getenv("GITEA_SCOPES")
	fs.StringVar(&scopes, "S", defaultScopes, "")
	fs.StringVar(&scopes, "scope", defaultScopes, "")
	fs.BoolVar(&flagPkg.Debug, "d", false, "")
	fs.BoolVar(&flagPkg.Debug, "debug", false, "")
	fs.BoolVar(&flagPkg.Insecure, "k", false, "")
	fs.BoolVar(&flagPkg.Insecure, "insecure", false, "")
	fs.BoolVar(&version, "v", false, "")
	fs.BoolVar(&version, "version", false, "")
	fs.BoolVar(&healthcheck, "healthcheck", false, "")
	maxInlineAttachmentBytes = 5 * 1024 * 1024
	fs.Func("max-inline-attachment-bytes", "", func(val string) error {
		parsed, err := strconv.Atoi(val)
		if err != nil || parsed < 0 {
			return fmt.Errorf("invalid value %q", val)
		}
		maxInlineAttachmentBytes = parsed
		maxInlineAttachmentBytesFlagSet = true
		return nil
	})

	fs.Usage = func() {
		w := tabwriter.NewWriter(stderr, 0, 0, 3, ' ', 0)
		fmt.Fprintln(stderr, "Usage: gitea-mcp [options]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Options:")
		fmt.Fprintf(w, "  -t, -transport <type>\tTransport type: stdio or http (default: stdio)\n")
		fmt.Fprintf(w, "  -H, -host <url>\tGitea host URL (default: https://gitea.com)\n")
		fmt.Fprintf(w, "  -b, -bind <address>\tHTTP listen address, e.g. 127.0.0.1 (default: all interfaces)\n")
		fmt.Fprintf(w, "  -p, -port <number>\tHTTP server port (default: 8080)\n")
		fmt.Fprintf(w, "  -T, -token <token>\tPersonal access token\n")
		fmt.Fprintf(w, "  -r, -read-only\tExpose only read-only tools\n")
		fmt.Fprintf(w, "  -O, -tools <names>\tComma-separated list of tool names to expose\n")
		fmt.Fprintf(w, "  -S, -scope <names>\tComma-separated list of tool scopes to expose\n")
		fmt.Fprintf(w, "  -d, -debug\tEnable debug mode\n")
		fmt.Fprintf(w, "  -k, -insecure\tIgnore TLS certificate errors\n")
		fmt.Fprintf(w, "  -max-inline-attachment-bytes <bytes>\tInline image attachments up to this size (default: 5242880)\n")
		fmt.Fprintf(w, "  -v, -version\tPrint version and exit\n")
		fmt.Fprintf(w, "  -healthcheck\tCheck a running HTTP server's /healthz endpoint and exit\n")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Environment variables:")
		fmt.Fprintf(w, "  GITEA_ACCESS_TOKEN\tProvide access token\n")
		fmt.Fprintf(w, "  GITEA_ACCESS_TOKEN_FILE\tPath to a file containing the access token (e.g. a Docker secret)\n")
		fmt.Fprintf(w, "  GITEA_DEBUG\tSet to 'true' for debug mode\n")
		fmt.Fprintf(w, "  GITEA_HOST\tOverride Gitea host URL\n")
		fmt.Fprintf(w, "  GITEA_INSECURE\tSet to 'true' to ignore TLS errors\n")
		fmt.Fprintf(w, "  GITEA_MAX_INLINE_ATTACHMENT_BYTES\tOverride inline image attachment size limit in bytes\n")
		fmt.Fprintf(w, "  GITEA_READONLY\tSet to 'true' for read-only mode\n")
		fmt.Fprintf(w, "  GITEA_SCOPES\tComma-separated list of tool scopes to expose\n")
		fmt.Fprintf(w, "  GITEA_TOOLS\tComma-separated list of tool names to expose\n")
		fmt.Fprintf(w, "  MCP_MODE\tOverride transport mode\n")
		_ = w.Flush()
	}

	_ = fs.Parse(args)

	flagPkg.Host = host
	if flagPkg.Host == "" {
		flagPkg.Host = "https://gitea.com"
	}

	flagPkg.Bind = bind
	flagPkg.Port = port
	flagPkg.MaxInlineAttachmentBytes = maxInlineAttachmentBytes

	flagPkg.Token = token
	if flagPkg.Token == "" {
		flagPkg.Token = getenv("GITEA_ACCESS_TOKEN")
	}
	if flagPkg.Token == "" {
		if tokenFile := getenv("GITEA_ACCESS_TOKEN_FILE"); tokenFile != "" {
			data, err := readFile(tokenFile)
			if err != nil {
				fmt.Fprintf(stderr, "error reading GITEA_ACCESS_TOKEN_FILE: %v\n", err)
				osExit(1)
			}
			flagPkg.Token = strings.TrimRight(string(data), "\r\n")
		}
	}

	if getenv("MCP_MODE") != "" {
		flagPkg.Mode = getenv("MCP_MODE")
	}
	if getenv("GITEA_READONLY") == "true" {
		flagPkg.ReadOnly = true
	}

	allowed := map[string]struct{}{}
	for t := range strings.SplitSeq(tools, ",") {
		if t = strings.TrimSpace(t); t != "" {
			allowed[t] = struct{}{}
		}
	}
	if len(allowed) > 0 {
		flagPkg.AllowedTools = allowed
	}

	allowedScopes := map[string]struct{}{}
	for s := range strings.SplitSeq(scopes, ",") {
		if s = normalizeScope(s); s != "" {
			allowedScopes[s] = struct{}{}
		}
	}
	if len(allowedScopes) > 0 {
		flagPkg.AllowedScopes = allowedScopes
	}
	if getenv("GITEA_DEBUG") == "true" {
		flagPkg.Debug = true
	}
	if getenv("GITEA_INSECURE") == "true" {
		flagPkg.Insecure = true
	}
	if !maxInlineAttachmentBytesFlagSet {
		if val := getenv("GITEA_MAX_INLINE_ATTACHMENT_BYTES"); val != "" {
			parsed, err := strconv.Atoi(val)
			if err != nil || parsed < 0 {
				fmt.Fprintf(stderr, "invalid GITEA_MAX_INLINE_ATTACHMENT_BYTES: %q\n", val)
				osExit(1)
			}
			flagPkg.MaxInlineAttachmentBytes = parsed
		}
	}
}

// normalizeScope trims whitespace, lowercases, and converts internal spaces
// and hyphens to underscores, so "Pull Request", "pull-request", and
// "PULL_REQUEST" all normalize to "pull_request".
func normalizeScope(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func Execute() {
	initFlagSet(flag.CommandLine, os.Args[1:], os.Getenv, os.ReadFile, os.Stderr)
	if version {
		fmt.Fprintln(os.Stdout, flagPkg.Version)
		return
	}
	if healthcheck {
		if runHealthcheck(healthcheckClient, flagPkg.Port, os.Stdout, os.Stderr) {
			osExit(0)
		} else {
			osExit(1)
		}
		return
	}
	if err := operation.Run(); err != nil {
		if err == context.Canceled {
			log.Info("Server shutdown due to context cancellation")
			_ = log.Default().Sync() // best-effort flush
			return
		}
		_ = log.Default().Sync() // best-effort flush
		log.Fatalf("Run Gitea MCP Server Error: %v", err)
	}
	_ = log.Default().Sync() // best-effort flush
}
