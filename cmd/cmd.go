package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gitea.com/gitea/gitea-mcp/operation"
	flagPkg "gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/log"
)

var (
	host    string
	port    int
	token   string
	version bool
)

func init() {
	flag.StringVar(&flagPkg.Mode, "t", "stdio", "")
	flag.StringVar(&flagPkg.Mode, "transport", "stdio", "")
	flag.StringVar(&host, "H", os.Getenv("GITEA_HOST"), "")
	flag.StringVar(&host, "host", os.Getenv("GITEA_HOST"), "")
	flag.IntVar(&port, "p", 8080, "")
	flag.IntVar(&port, "port", 8080, "")
	flag.StringVar(&token, "T", "", "")
	flag.StringVar(&token, "token", "", "")
	flag.BoolVar(&flagPkg.ReadOnly, "r", false, "")
	flag.BoolVar(&flagPkg.ReadOnly, "read-only", false, "")
	flag.BoolVar(&flagPkg.Debug, "d", false, "")
	flag.BoolVar(&flagPkg.Debug, "debug", false, "")
	flag.BoolVar(&flagPkg.Insecure, "k", false, "")
	flag.BoolVar(&flagPkg.Insecure, "insecure", false, "")
	flag.BoolVar(&version, "v", false, "")
	flag.BoolVar(&version, "version", false, "")

	flag.Usage = func() {
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
		fmt.Fprintln(os.Stderr, "Usage: gitea-mcp [options]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintf(w, "  -t, -transport <type>\tTransport type: stdio or http (default: stdio)\n")
		fmt.Fprintf(w, "  -H, -host <url>\tGitea host URL (default: https://gitea.com)\n")
		fmt.Fprintf(w, "  -p, -port <number>\tHTTP server port (default: 8080)\n")
		fmt.Fprintf(w, "  -T, -token <token>\tPersonal access token\n")
		fmt.Fprintf(w, "  -r, -read-only\tExpose only read-only tools\n")
		fmt.Fprintf(w, "  -d, -debug\tEnable debug mode\n")
		fmt.Fprintf(w, "  -k, -insecure\tIgnore TLS certificate errors\n")
		fmt.Fprintf(w, "  -v, -version\tPrint version and exit\n")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Environment variables:")
		fmt.Fprintf(w, "  GITEA_ACCESS_TOKEN\tProvide access token\n")
		fmt.Fprintf(w, "  GITEA_ACCESS_TOKEN_FILE\tPath to a file containing the access token (e.g. a Docker secret)\n")
		fmt.Fprintf(w, "  GITEA_DEBUG\tSet to 'true' for debug mode\n")
		fmt.Fprintf(w, "  GITEA_HOST\tOverride Gitea host URL\n")
		fmt.Fprintf(w, "  GITEA_INSECURE\tSet to 'true' to ignore TLS errors\n")
		fmt.Fprintf(w, "  GITEA_READONLY\tSet to 'true' for read-only mode\n")
		fmt.Fprintf(w, "  MCP_MODE\tOverride transport mode\n")
		w.Flush()
	}

	flag.Parse()

	flagPkg.Host = host
	if flagPkg.Host == "" {
		flagPkg.Host = "https://gitea.com"
	}

	flagPkg.Port = port

	flagPkg.Token = token
	if flagPkg.Token == "" {
		flagPkg.Token = os.Getenv("GITEA_ACCESS_TOKEN")
	}
	if flagPkg.Token == "" {
		if tokenFile := os.Getenv("GITEA_ACCESS_TOKEN_FILE"); tokenFile != "" {
			data, err := os.ReadFile(tokenFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading GITEA_ACCESS_TOKEN_FILE: %v\n", err)
				os.Exit(1)
			}
			flagPkg.Token = strings.TrimRight(string(data), "\r\n")
		}
	}

	if os.Getenv("MCP_MODE") != "" {
		flagPkg.Mode = os.Getenv("MCP_MODE")
	}

	if os.Getenv("GITEA_READONLY") == "true" {
		flagPkg.ReadOnly = true
	}

	if os.Getenv("GITEA_DEBUG") == "true" {
		flagPkg.Debug = true
	}

	// Set insecure mode based on environment variable
	if os.Getenv("GITEA_INSECURE") == "true" {
		flagPkg.Insecure = true
	}
}

func Execute() {
	if version {
		fmt.Fprintln(os.Stdout, flagPkg.Version)
		return
	}
	defer log.Default().Sync() //nolint:errcheck // best-effort flush
	if err := operation.Run(); err != nil {
		if err == context.Canceled {
			log.Info("Server shutdown due to context cancellation")
			return
		}
		log.Fatalf("Run Gitea MCP Server Error: %v", err) //nolint:gocritic // intentional exit after defer
	}
}
