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

	"github.com/mark3labs/mcp-go/server"
)

var mcpServer *server.MCPServer

func RegisterTool(s *server.MCPServer) {
	// User Tool
	s.AddTools(user.Tool.Tools()...)

	// Actions Tool
	s.AddTools(actions.Tool.Tools()...)

	// Repo Tool
	s.AddTools(repo.Tool.Tools()...)

	// Notification Tool
	s.AddTools(notification.Tool.Tools()...)

	// Issue Tool
	s.AddTools(issue.Tool.Tools()...)

	// Label Tool
	s.AddTools(label.Tool.Tools()...)

	// Milestone Tool
	s.AddTools(milestone.Tool.Tools()...)

	// Pull Tool
	s.AddTools(pull.Tool.Tools()...)

	// Search Tool
	s.AddTools(search.Tool.Tools()...)

	// Version Tool
	s.AddTools(version.Tool.Tools()...)

	// Wiki Tool
	s.AddTools(wiki.Tool.Tools()...)

	// Time Tracking Tool
	s.AddTools(timetracking.Tool.Tools()...)

	s.DeleteTools("")
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
			server.WithLogger(log.New()),
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
