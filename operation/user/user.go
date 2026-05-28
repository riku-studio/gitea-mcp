package user

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/slim"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	GetMyUserInfoToolName = "get_me"
	GetUserOrgsToolName   = "get_user_orgs"
)

var Tool = tool.New()

var (
	GetMyUserInfoTool = mcp.NewTool(
		GetMyUserInfoToolName,
		mcp.WithDescription("Get current user"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get current user information")),
	)

	GetUserOrgsTool = mcp.NewTool(
		GetUserOrgsToolName,
		mcp.WithDescription("List current user's organizations"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get user organizations")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{Tool: GetMyUserInfoTool, Handler: GetUserInfoFn})
	Tool.RegisterRead(server.ServerTool{Tool: GetUserOrgsTool, Handler: GetUserOrgsFn})
}

func GetUserInfoFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	user, _, err := client.Users.GetMyUserInfo(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get user info err: %v", err))
	}
	return to.TextResult(slim.UserDetail(user))
}

func GetUserOrgsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

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
	orgs, _, err := client.Organizations.ListMyOrgs(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get user orgs err: %v", err))
	}
	return to.TextResult(slimOrgs(orgs))
}
