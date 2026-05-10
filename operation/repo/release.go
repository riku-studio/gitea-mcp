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
	CreateReleaseToolName    = "create_release"
	DeleteReleaseToolName    = "delete_release"
	GetReleaseToolName       = "get_release"
	GetLatestReleaseToolName = "get_latest_release"
	ListReleasesToolName     = "list_releases"
)

var (
	CreateReleaseTool = mcp.NewTool(
		CreateReleaseToolName,
		mcp.WithDescription("Create release"),
		mcp.WithToolAnnotation(annotation.Write("Create a release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("tag_name", mcp.Required(), mcp.Description("tag name")),
		mcp.WithString("target", mcp.Required(), mcp.Description("target commitish")),
		mcp.WithString("title", mcp.Required(), mcp.Description("release title")),
		mcp.WithBoolean("is_draft", mcp.Description("Whether the release is draft"), mcp.DefaultBool(false)),
		mcp.WithBoolean("is_pre_release", mcp.Description("Whether the release is pre-release"), mcp.DefaultBool(false)),
		mcp.WithString("body", mcp.Description("release body")),
	)

	DeleteReleaseTool = mcp.NewTool(
		DeleteReleaseToolName,
		mcp.WithDescription("Delete release"),
		mcp.WithToolAnnotation(annotation.Destructive("Delete a release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("release id")),
	)

	GetReleaseTool = mcp.NewTool(
		GetReleaseToolName,
		mcp.WithDescription("Get release"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get release details")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("release id")),
	)

	GetLatestReleaseTool = mcp.NewTool(
		GetLatestReleaseToolName,
		mcp.WithDescription("Get latest release"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get latest release")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
	)

	ListReleasesTool = mcp.NewTool(
		ListReleasesToolName,
		mcp.WithDescription("List releases"),
		mcp.WithToolAnnotation(annotation.ReadOnly("List releases")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithBoolean("is_draft", mcp.Description("Whether the release is draft"), mcp.DefaultBool(false)),
		mcp.WithBoolean("is_pre_release", mcp.Description("Whether the release is pre-release"), mcp.DefaultBool(false)),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(20), mcp.Min(1)),
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
	log.Debugf("Called CreateReleasesFn")
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
	_, _, err = client.CreateRelease(owner, repo, gitea_sdk.CreateReleaseOption{
		TagName:      tagName,
		Target:       target,
		Title:        title,
		Note:         body,
		IsDraft:      isDraft,
		IsPrerelease: isPreRelease,
	})
	if err != nil {
		return nil, fmt.Errorf("create release error: %v", err)
	}

	return mcp.NewToolResultText("Release Created"), nil
}

func DeleteReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called DeleteReleaseFn")
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
	_, err = client.DeleteRelease(owner, repo, id)
	if err != nil {
		return nil, fmt.Errorf("delete release error: %v", err)
	}

	return to.TextResult("Release deleted successfully")
}

func GetReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetReleaseFn")
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
	release, _, err := client.GetRelease(owner, repo, id)
	if err != nil {
		return nil, fmt.Errorf("get release error: %v", err)
	}

	return to.TextResult(slimRelease(release))
}

func GetLatestReleaseFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetLatestReleaseFn")
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
	release, _, err := client.GetLatestRelease(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get latest release error: %v", err)
	}

	return to.TextResult(slimRelease(release))
}

func ListReleasesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListReleasesFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	var pIsDraft *bool
	isDraft, ok := args["is_draft"].(bool)
	if ok {
		pIsDraft = new(isDraft)
	}
	var pIsPreRelease *bool
	isPreRelease, ok := args["is_pre_release"].(bool)
	if ok {
		pIsPreRelease = new(isPreRelease)
	}
	page := params.GetOptionalInt(args, "page", 1)
	pageSize := params.GetOptionalInt(args, "perPage", 20)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	releases, _, err := client.ListReleases(owner, repo, gitea_sdk.ListReleasesOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     int(page),
			PageSize: int(pageSize),
		},
		IsDraft:      pIsDraft,
		IsPreRelease: pIsPreRelease,
	})
	if err != nil {
		return nil, fmt.Errorf("list releases error: %v", err)
	}

	return to.TextResult(slimReleases(releases))
}
