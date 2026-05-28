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
	CreateTagToolName = "create_tag"
	DeleteTagToolName = "delete_tag"
	GetTagToolName    = "get_tag"
	ListTagsToolName  = "list_tags"
)

var (
	CreateTagTool = mcp.NewTool(
		CreateTagToolName,
		mcp.WithToolAnnotation(annotation.Write("Create a tag")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("tag_name", mcp.Required()),
		mcp.WithString("target", mcp.Description("commitish")),
		mcp.WithString("message", mcp.Description("tag message")),
	)

	DeleteTagTool = mcp.NewTool(
		DeleteTagToolName,
		mcp.WithToolAnnotation(annotation.Destructive("Delete a tag")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("tag_name", mcp.Required()),
	)

	GetTagTool = mcp.NewTool(
		GetTagToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("Get tag details")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("tag_name", mcp.Required()),
	)

	ListTagsTool = mcp.NewTool(
		ListTagsToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("List tags")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(20), mcp.Min(1)),
	)
)

func init() {
	Tool.RegisterWrite(server.ServerTool{
		Tool:    CreateTagTool,
		Handler: CreateTagFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    DeleteTagTool,
		Handler: DeleteTagFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    GetTagTool,
		Handler: GetTagFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListTagsTool,
		Handler: ListTagsFn,
	})
}

func CreateTagFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	target, _ := args["target"].(string)
	message, _ := args["message"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, _, err = client.Repositories.CreateTag(ctx, owner, repo, gitea_sdk.CreateTagOption{
		TagName: tagName,
		Target:  target,
		Message: message,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create tag error: %v", err))
	}

	return to.TextResult("Tag Created")
}

func DeleteTagFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.Repositories.DeleteTag(ctx, owner, repo, tagName)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete tag error: %v", err))
	}

	return to.TextResult("Tag deleted")
}

func GetTagFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	tag, _, err := client.Repositories.GetTag(ctx, owner, repo, tagName)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get tag error: %v", err))
	}

	return to.TextResult(slimTag(tag))
}

func ListTagsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page := params.GetOptionalInt(args, "page", 1)
	pageSize := params.GetOptionalInt(args, "per_page", 20)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	tags, _, err := client.Repositories.ListRepoTags(ctx, owner, repo, gitea_sdk.ListRepoTagsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     int(page),
			PageSize: int(pageSize),
		},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list tags error: %v", err))
	}

	return to.TextResult(slimTags(tags))
}
