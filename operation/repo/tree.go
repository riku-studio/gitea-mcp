package repo

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"

	gitea_sdk "gitea.dev/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	GetRepoTreeToolName = "get_repository_tree"
)

var GetRepoTreeTool = mcp.NewTool(
	GetRepoTreeToolName,
	mcp.WithDescription("Get the file tree of a repository at a given ref (SHA, branch, or tag), optionally recursively."),
	mcp.WithToolAnnotation(annotation.ReadOnly("Get repository file tree")),
	mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
	mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
	mcp.WithString("tree_sha", mcp.Required(), mcp.Description("SHA, branch, or tag")),
	mcp.WithBoolean("recursive"),
	mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
	mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetRepoTreeTool,
		Handler: GetRepoTreeFn,
	})
}

func GetRepoTreeFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	tree, _, err := client.Git.GetTrees(ctx, owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get repository tree err: %v", err))
	}
	return to.TextResult(slimTree(tree))
}
