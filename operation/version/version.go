package version

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	GetGiteaMCPServerVersion = "get_gitea_mcp_server_version"
)

var GetGiteaMCPServerVersionTool = mcp.NewTool(
	GetGiteaMCPServerVersion,
	mcp.WithToolAnnotation(annotation.ReadOnly("Get server version")),
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetGiteaMCPServerVersionTool,
		Handler: GetGiteaMCPServerVersionFn,
	})
}

func GetGiteaMCPServerVersionFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetGiteaMCPServerVersionFn")
	version := flag.Version
	if version == "" {
		version = "dev"
	}
	return to.TextResult(fmt.Sprintf("Gitea MCP Server version: %v", version))
}
