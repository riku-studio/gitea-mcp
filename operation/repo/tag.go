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
	CreateTagToolName = "create_tag"
	DeleteTagToolName = "delete_tag"
	GetTagToolName    = "get_tag"
	ListTagsToolName  = "list_tags"
)

var (
	CreateTagTool = mcp.NewTool(
		CreateTagToolName,
		mcp.WithDescription("Create tag"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("tag_name", mcp.Required(), mcp.Description("tag name")),
		mcp.WithString("target", mcp.Description("target commitish"), mcp.DefaultString("")),
		mcp.WithString("message", mcp.Description("tag message"), mcp.DefaultString("")),
	)

	DeleteTagTool = mcp.NewTool(
		DeleteTagToolName,
		mcp.WithDescription("Delete tag"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("tag_name", mcp.Required(), mcp.Description("tag name")),
	)

	GetTagTool = mcp.NewTool(
		GetTagToolName,
		mcp.WithDescription("Get tag"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("tag_name", mcp.Required(), mcp.Description("tag name")),
	)

	ListTagsTool = mcp.NewTool(
		ListTagsToolName,
		mcp.WithDescription("List tags"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(20), mcp.Min(1)),
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
	log.Debugf("Called CreateTagFn")
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
	_, _, err = client.CreateTag(owner, repo, gitea_sdk.CreateTagOption{
		TagName: tagName,
		Target:  target,
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("create tag error: %v", err)
	}

	return mcp.NewToolResultText("Tag Created"), nil
}

func DeleteTagFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called DeleteTagFn")
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
	_, err = client.DeleteTag(owner, repo, tagName)
	if err != nil {
		return nil, fmt.Errorf("delete tag error: %v", err)
	}

	return to.TextResult("Tag deleted")
}

func GetTagFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called GetTagFn")
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
	tag, _, err := client.GetTag(owner, repo, tagName)
	if err != nil {
		return nil, fmt.Errorf("get tag error: %v", err)
	}

	return to.TextResult(slimTag(tag))
}

func ListTagsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListTagsFn")
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
	pageSize := params.GetOptionalInt(args, "perPage", 20)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	tags, _, err := client.ListRepoTags(owner, repo, gitea_sdk.ListRepoTagsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     int(page),
			PageSize: int(pageSize),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list tags error: %v", err)
	}

	return to.TextResult(slimTags(tags))
}
