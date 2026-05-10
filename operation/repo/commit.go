package repo

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ListRepoCommitsToolName = "list_commits"
	GetCommitToolName       = "get_commit"
)

var (
	ListRepoCommitsTool = mcp.NewTool(
		ListRepoCommitsToolName,
		mcp.WithDescription("List repository commits"),
		mcp.WithToolAnnotation(annotation.ReadOnly("List repository commits")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("sha", mcp.Description("SHA or branch to start listing commits from")),
		mcp.WithString("path", mcp.Description("path indicates that only commits that include the path's file/dir should be returned.")),
		mcp.WithNumber("page", mcp.Required(), mcp.Description("page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Required(), mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	GetCommitTool = mcp.NewTool(
		GetCommitToolName,
		mcp.WithDescription("Get details of a specific commit"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get commit details")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("sha", mcp.Required(), mcp.Description("commit SHA")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListRepoCommitsTool,
		Handler: ListRepoCommitsFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetCommitTool,
		Handler: GetCommitFn,
	})
}

func ListRepoCommitsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListRepoCommitsFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, err := params.GetIndex(args, "page")
	if err != nil {
		return to.ErrorResult(err)
	}
	pageSize, err := params.GetIndex(args, "perPage")
	if err != nil {
		return to.ErrorResult(err)
	}
	sha, _ := args["sha"].(string)
	path, _ := args["path"].(string)
	opt := gitea_sdk.ListCommitOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     int(page),
			PageSize: int(pageSize),
		},
		SHA:  sha,
		Path: path,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	commits, _, err := client.ListRepoCommits(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list repo commits err: %v", err))
	}
	return to.TextResult(slimCommits(commits))
}

func GetCommitFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetCommitFn")
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
	commit, _, err := client.GetSingleCommit(owner, repo, sha)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get commit %v err: %v", sha, err))
	}
	return to.TextResult(slimCommit(commit))
}
