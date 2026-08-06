package operation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitea.com/gitea/gitea-mcp/operation/actions"
	"gitea.com/gitea/gitea-mcp/operation/issue"
	"gitea.com/gitea/gitea-mcp/operation/label"
	"gitea.com/gitea/gitea-mcp/operation/milestone"
	"gitea.com/gitea/gitea-mcp/operation/notification"
	"gitea.com/gitea/gitea-mcp/operation/packages"
	"gitea.com/gitea/gitea-mcp/operation/pull"
	"gitea.com/gitea/gitea-mcp/operation/repo"
	"gitea.com/gitea/gitea-mcp/operation/search"
	"gitea.com/gitea/gitea-mcp/operation/timetracking"
	"gitea.com/gitea/gitea-mcp/operation/user"
	"gitea.com/gitea/gitea-mcp/operation/version"
	"gitea.com/gitea/gitea-mcp/operation/wiki"
	mcpContext "gitea.com/gitea/gitea-mcp/pkg/context"
	"gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxRequestBodyBytes raises the SDK's 4 MiB default, which is too tight for the
// base64 file content create_or_update_file accepts.
const maxRequestBodyBytes = 32 << 20

// sessionTimeout expires idle sessions, which the SDK otherwise keeps for the
// process lifetime: a client that goes away without DELETE /mcp leaks its
// session, and initialize takes no token. Clients re-initialize on the 404.
const sessionTimeout = 30 * time.Minute

// httpReadHeaderTimeout bounds slow header reads without limiting SSE writes.
const httpReadHeaderTimeout = 10 * time.Second

var (
	mcpServer *mcp.Server

	domainTools = []*tool.Tool{
		user.Tool, actions.Tool, repo.Tool, notification.Tool, issue.Tool,
		label.Tool, milestone.Tool, packages.Tool, pull.Tool, search.Tool,
		version.Tool, wiki.Tool, timetracking.Tool,
		repo.FileTool, repo.BranchTool, repo.TagTool, repo.CommitTool, repo.ReleaseTool,
	}
)

func RegisterTool(s *mcp.Server) {
	for _, t := range domainTools {
		for _, registeredTool := range t.Tools() {
			s.AddTool(registeredTool.Tool, registeredTool.MCPHandler())
		}
	}
	tool.WarnUnmatchedAllowedTools(domainTools...)
	tool.WarnUnmatchedAllowedScopes(domainTools...)
}

// parseAuthToken extracts the token from an Authorization header.
// Supports "Bearer <token>" (case-insensitive per RFC 7235) and
// Gitea-style "token <token>" formats.
// Returns the token and true if valid, empty string and false otherwise.
func parseAuthToken(authHeader string) (string, bool) {
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		if token != "" {
			return token, true
		}
	}
	if len(authHeader) > 6 && strings.EqualFold(authHeader[:6], "token ") {
		token := strings.TrimSpace(authHeader[6:])
		if token != "" {
			return token, true
		}
	}
	return "", false
}

func getContextWithToken(ctx context.Context, authHeader string) context.Context {
	if authHeader == "" {
		return ctx
	}

	token, ok := parseAuthToken(authHeader)
	if !ok {
		return ctx
	}

	return context.WithValue(ctx, mcpContext.TokenContextKey, token)
}

func authTokenMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if extra := req.GetExtra(); extra != nil {
			ctx = getContextWithToken(ctx, extra.Header.Get("Authorization"))
		}
		return next(ctx, method, req)
	}
}

func newHTTPServer(addr string, s *mcp.Server) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{
			Logger:              log.Slog(),
			MaxRequestBodyBytes: maxRequestBodyBytes,
			Stateless:           false, // SessionTimeout requires stateful sessions.
			SessionTimeout:      sessionTimeout,
		},
	))
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}
}

func Run() error {
	mcpServer = newMCPServer(flag.Version)
	RegisterTool(mcpServer)
	switch flag.Mode {
	case "stdio":
		if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			return err
		}
	case "http":
		httpServer := newHTTPServer(fmt.Sprintf(":%d", flag.Port), mcpServer)
		log.Infof("Gitea MCP HTTP server listening on :%d", flag.Port)

		// Graceful shutdown setup
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		shutdownDone := make(chan struct{})

		go func() {
			<-sigCh
			log.Infof("Shutdown signal received, gracefully stopping HTTP server...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Errorf("HTTP server shutdown error: %v", err)
			}
			close(shutdownDone)
		}()

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		<-shutdownDone // Wait for shutdown to finish
	default:
		return fmt.Errorf("invalid transport type: %s. Must be 'stdio' or 'http'", flag.Mode)
	}
	return nil
}

func newMCPServer(version string) *mcp.Server {
	// SDK keepalives send MCP ping requests and disconnect clients without a
	// server-to-client channel, so KeepAlive stays disabled.
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "Gitea MCP Server",
			Version: version,
		},
		&mcp.ServerOptions{Logger: log.Slog()},
	)
	s.AddReceivingMiddleware(authTokenMiddleware)
	return s
}
