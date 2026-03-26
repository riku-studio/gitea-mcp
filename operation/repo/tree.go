package repo

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	GetRepoTreeToolName = "get_repository_tree"
)

var GetRepoTreeTool = mcp.NewTool(
	GetRepoTreeToolName,
	mcp.WithDescription("Get the file tree of a repository"),
	mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
	mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
	mcp.WithString("tree_sha", mcp.Required(), mcp.Description("SHA, branch name, or tag name")),
	mcp.WithBoolean("recursive", mcp.Description("whether to get the tree recursively")),
	mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
	mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetRepoTreeTool,
		Handler: GetRepoTreeFn,
	})
}

func GetRepoTreeFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetRepoTreeFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	treeSHA, err := params.GetString(args, "tree_sha")
	if err != nil {
		return to.ErrorResult(err)
	}
	recursive, _ := args["recursive"].(bool)
	page, pageSize := params.GetPagination(args, 30)

	opt := gitea_sdk.ListTreeOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
		Ref:       treeSHA,
		Recursive: recursive,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	tree, _, err := client.GetTrees(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get repository tree err: %v", err))
	}
	return to.TextResult(slimTree(tree))
}
