package user

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// GetMyUserInfoToolName is the unique tool name used for MCP registration and lookup of the get_me command.
	GetMyUserInfoToolName = "get_me"
	// GetUserOrgsToolName is the unique tool name used for MCP registration and lookup of the get_user_orgs command.
	GetUserOrgsToolName = "get_user_orgs"

	// defaultPage is the default starting page number used for paginated organization listings.
	defaultPage = 1
	// defaultPageSize is the default number of organizations per page for paginated queries.
	defaultPageSize = 30
)

// Tool is the MCP tool manager instance for registering all MCP tools in this package.
var Tool = tool.New()

var (
	// GetMyUserInfoTool is the MCP tool for retrieving the current user's info.
	// It is registered with a specific name and a description string.
	GetMyUserInfoTool = mcp.NewTool(
		GetMyUserInfoToolName,
		mcp.WithDescription("Get current user"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get current user information")),
	)

	// GetUserOrgsTool is the MCP tool for listing organizations for the authenticated user.
	// It supports pagination via "page" and "per_page" arguments with default values specified above.
	GetUserOrgsTool = mcp.NewTool(
		GetUserOrgsToolName,
		mcp.WithDescription("List current user's organizations"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get user organizations")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(defaultPage)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(defaultPageSize)),
	)
)

// init registers all MCP tools in Tool at package initialization.
// This function ensures the handler functions are registered before server usage.
func init() {
	registerTools()
}

// registerTools registers all local MCP tool definitions and their handler functions.
// To add new functionality, append your tool/handler pair to the tools slice below.
func registerTools() {
	tools := []server.ServerTool{
		{Tool: GetMyUserInfoTool, Handler: GetUserInfoFn},
		{Tool: GetUserOrgsTool, Handler: GetUserOrgsFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

// GetUserInfoFn is the handler for "get_me" MCP tool requests.
// Logs invocation, fetches current user info from gitea, wraps result for MCP.
func GetUserInfoFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called GetUserInfoFn")
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	user, _, err := client.GetMyUserInfo()
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get user info err: %v", err))
	}
	return to.TextResult(slimUserDetail(user))
}

// GetUserOrgsFn is the handler for "get_user_orgs" MCP tool requests.
// Logs invocation, pulls validated pagination arguments from request,
// performs Gitea organization listing, and wraps the result for MCP.
func GetUserOrgsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called GetUserOrgsFn")
	page, pageSize := params.GetPagination(req.GetArguments(), defaultPageSize)

	opt := gitea_sdk.ListOrgsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	orgs, _, err := client.ListMyOrgs(opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get user orgs err: %v", err))
	}
	return to.TextResult(slimOrgs(orgs))
}
