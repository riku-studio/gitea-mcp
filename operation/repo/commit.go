package repo

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CommitTool holds the commit-related tools (scope "commit").
var CommitTool = tool.New("commit")

const (
	ListRepoCommitsToolName = "list_commits"
	GetCommitToolName       = "get_commit"
)

var (
	ListRepoCommitsTool = mcp.NewTool(
		ListRepoCommitsToolName,
		mcp.WithDescription("List commits in a repository, optionally starting from a specific branch or SHA and filtered to commits touching a given file path."),
		mcp.WithToolAnnotation(annotation.ReadOnly("List repository commits")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("sha", mcp.Description("starting SHA or branch")),
		mcp.WithString("path", mcp.Description("only commits touching this path")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	GetCommitTool = mcp.NewTool(
		GetCommitToolName,
		mcp.WithDescription("Get details for a single commit in a repository by its SHA."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get commit details")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("sha", mcp.Required()),
	)
)

func init() {
	CommitTool.RegisterRead(server.ServerTool{
		Tool:    ListRepoCommitsTool,
		Handler: ListRepoCommitsFn,
	})
	CommitTool.RegisterRead(server.ServerTool{
		Tool:    GetCommitTool,
		Handler: GetCommitFn,
	})
}

func ListRepoCommitsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(args, 30)
	sha, _ := args["sha"].(string)
	path, _ := args["path"].(string)
	opt := gitea_sdk.ListCommitOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
		SHA:  sha,
		Path: path,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	commits, _, err := client.Repositories.ListRepoCommits(ctx, owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list repo commits err: %v", err))
	}
	return to.TextResult(slimCommits(commits))
}

func GetCommitFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	sha, err := params.GetString(args, "sha")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	commit, _, err := client.Repositories.GetSingleCommit(ctx, owner, repo, sha)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get commit %v err: %v", sha, err))
	}
	return to.TextResult(slimCommit(commit))
}
