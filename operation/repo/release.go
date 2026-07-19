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
	CreateReleaseToolName    = "create_release"
	DeleteReleaseToolName    = "delete_release"
	GetReleaseToolName       = "get_release"
	GetLatestReleaseToolName = "get_latest_release"
	ListReleasesToolName     = "list_releases"
)

var (
	CreateReleaseTool = mcp.NewTool(
		CreateReleaseToolName,
		mcp.WithDescription("Create a new release in a repository from a tag, optionally marking it as a draft or pre-release."),
		mcp.WithToolAnnotation(annotation.Write("Create a release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("tag_name", mcp.Required()),
		mcp.WithString("target", mcp.Required(), mcp.Description("commitish")),
		mcp.WithString("title", mcp.Required()),
		mcp.WithBoolean("is_draft"),
		mcp.WithBoolean("is_pre_release"),
		mcp.WithString("body"),
	)

	DeleteReleaseTool = mcp.NewTool(
		DeleteReleaseToolName,
		mcp.WithDescription("Delete a release from a repository by its numeric ID. This action is destructive and cannot be undone."),
		mcp.WithToolAnnotation(annotation.Destructive("Delete a release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("id", mcp.Required()),
	)

	GetReleaseTool = mcp.NewTool(
		GetReleaseToolName,
		mcp.WithDescription("Get a release by ID"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get release details")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("id", mcp.Required()),
	)

	GetLatestReleaseTool = mcp.NewTool(
		GetLatestReleaseToolName,
		mcp.WithDescription("Get the most recent published (non-draft) release in a repository."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get latest release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
	)

	ListReleasesTool = mcp.NewTool(
		ListReleasesToolName,
		mcp.WithDescription("List releases in a repository, optionally filtered to drafts or pre-releases."),
		mcp.WithToolAnnotation(annotation.ReadOnly("List releases")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithBoolean("is_draft"),
		mcp.WithBoolean("is_pre_release"),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(20), mcp.Min(1)),
	)
)

func init() {
	Tool.RegisterWrite(server.ServerTool{
		Tool:    CreateReleaseTool,
		Handler: CreateReleaseFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    DeleteReleaseTool,
		Handler: DeleteReleaseFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetReleaseTool,
		Handler: GetReleaseFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetLatestReleaseTool,
		Handler: GetLatestReleaseFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListReleasesTool,
		Handler: ListReleasesFn,
	})
}

func CreateReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	tagName, err := params.GetString(args, "tag_name")
	if err != nil {
		return to.ErrorResult(err)
	}
	target, err := params.GetString(args, "target")
	if err != nil {
		return to.ErrorResult(err)
	}
	title, err := params.GetString(args, "title")
	if err != nil {
		return to.ErrorResult(err)
	}
	isDraft, _ := args["is_draft"].(bool)
	isPreRelease, _ := args["is_pre_release"].(bool)
	body, _ := args["body"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, _, err = client.Releases.CreateRelease(ctx, owner, repo, gitea_sdk.CreateReleaseOption{
		TagName:      tagName,
		Target:       target,
		Title:        title,
		Note:         body,
		IsDraft:      isDraft,
		IsPrerelease: isPreRelease,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create release error: %v", err))
	}

	return to.TextResult("Release Created")
}

func DeleteReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(args, "id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.Releases.DeleteRelease(ctx, owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete release error: %v", err))
	}

	return to.TextResult("Release deleted successfully")
}

func GetReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(args, "id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	release, _, err := client.Releases.GetRelease(ctx, owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get release error: %v", err))
	}

	return to.TextResult(slimRelease(release))
}

func GetLatestReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	release, _, err := client.Releases.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get latest release error: %v", err))
	}

	return to.TextResult(slimRelease(release))
}

func ListReleasesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(args, 20)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	releases, _, err := client.Releases.ListReleases(ctx, owner, repo, gitea_sdk.ListReleasesOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
		IsDraft:      params.GetOptionalBoolPtr(args, "is_draft"),
		IsPreRelease: params.GetOptionalBoolPtr(args, "is_pre_release"),
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list releases error: %v", err))
	}

	return to.TextResult(slimReleases(releases))
}
