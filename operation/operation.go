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

	"github.com/mark3labs/mcp-go/server"
)

var (
	mcpServer *server.MCPServer

	domainTools = []*tool.Tool{
		user.Tool, actions.Tool, repo.Tool, notification.Tool, issue.Tool,
		label.Tool, milestone.Tool, packages.Tool, pull.Tool, search.Tool,
		version.Tool, wiki.Tool, timetracking.Tool,
		repo.FileTool, repo.BranchTool, repo.TagTool, repo.CommitTool, repo.ReleaseTool,
	}
)

func RegisterTool(s *server.MCPServer) {
	for _, t := range domainTools {
		s.AddTools(t.Tools()...)
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

func getContextWithToken(ctx context.Context, r *http.Request) context.Context {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ctx
	}

	token, ok := parseAuthToken(authHeader)
	if !ok {
		return ctx
	}

	return context.WithValue(ctx, mcpContext.TokenContextKey, token)
}

func Run() error {
	mcpServer = newMCPServer(flag.Version)
	RegisterTool(mcpServer)
	switch flag.Mode {
	case "stdio":
		if err := server.ServeStdio(
			mcpServer,
		); err != nil {
			return err
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithStreamableHTTPLogger(log.Slog()),
			server.WithHeartbeatInterval(30*time.Second),
			server.WithHTTPContextFunc(getContextWithToken),
		)
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

		if err := httpServer.Start(fmt.Sprintf(":%d", flag.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		<-shutdownDone // Wait for shutdown to finish
	default:
		return fmt.Errorf("invalid transport type: %s. Must be 'stdio' or 'http'", flag.Mode)
	}
	return nil
}

func newMCPServer(version string) *server.MCPServer {
	return server.NewMCPServer(
		"Gitea MCP Server",
		version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)
}
