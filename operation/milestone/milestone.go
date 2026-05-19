package milestone

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	MilestoneReadToolName  = "milestone_read"
	MilestoneWriteToolName = "milestone_write"
)

var (
	MilestoneReadTool = mcp.NewTool(
		MilestoneReadToolName,
		mcp.WithDescription("Read milestones: get one or list."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read milestones")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("get", "list")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("id", mcp.Description("for 'get'")),
		mcp.WithString("state", mcp.DefaultString("all")),
		mcp.WithString("name", mcp.Description("name filter (for 'list')")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	MilestoneWriteTool = mcp.NewTool(
		MilestoneWriteToolName,
		mcp.WithDescription("Write milestones: create, update, delete."),
		mcp.WithToolAnnotation(annotation.Destructive("Create, update, or delete milestones")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("create", "update", "edit", "delete")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("id", mcp.Description("for 'update'/'delete'")),
		mcp.WithString("title", mcp.Description("for 'create'")),
		mcp.WithString("description"),
		mcp.WithString("due_on", mcp.Description("due date")),
		mcp.WithString("state", mcp.Enum("open", "closed")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    MilestoneReadTool,
		Handler: milestoneReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    MilestoneWriteTool,
		Handler: milestoneWriteFn,
	})
}

func milestoneReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "get":
		return getMilestoneFn(ctx, req)
	case "list":
		return listMilestonesFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func milestoneWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create":
		return createMilestoneFn(ctx, req)
	case "update":
		return editMilestoneFn(ctx, req)
	case "edit":
		return editMilestoneFn(ctx, req)
	case "delete":
		return deleteMilestoneFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func getMilestoneFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	milestone, _, err := client.GetMilestone(owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/milestone/%v err: %v", owner, repo, id, err))
	}

	return to.TextResult(slimMilestone(milestone))
}

func listMilestonesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	state := params.GetOptionalString(req.GetArguments(), "state", "all")
	name := params.GetOptionalString(req.GetArguments(), "name", "")
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	opt := gitea_sdk.ListMilestoneOption{
		State: gitea_sdk.StateType(state),
		Name:  name,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	milestones, _, err := client.ListRepoMilestones(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/milestones err: %v", owner, repo, err))
	}
	return to.TextResult(slimMilestones(milestones))
}

func createMilestoneFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	title, err := params.GetString(req.GetArguments(), "title")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.CreateMilestoneOption{
		Title: title,
	}

	description, ok := req.GetArguments()["description"].(string)
	if ok {
		opt.Description = description
	}
	opt.Deadline = params.GetOptionalTime(req.GetArguments(), "due_on")

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	milestone, _, err := client.CreateMilestone(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/%v/milestone err: %v", owner, repo, err))
	}

	return to.TextResult(slimMilestone(milestone))
}

func editMilestoneFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}

	args := req.GetArguments()
	opt := gitea_sdk.EditMilestoneOption{
		Description: params.GetPresentStringPtr(args, "description"),
		Deadline:    params.GetOptionalTime(args, "due_on"),
	}
	if title, ok := args["title"].(string); ok {
		opt.Title = title
	}
	if state, ok := args["state"].(string); ok {
		s := gitea_sdk.StateType(state)
		opt.State = &s
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	milestone, _, err := client.EditMilestone(owner, repo, id, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/milestone/%v err: %v", owner, repo, id, err))
	}

	return to.TextResult(slimMilestone(milestone))
}

func deleteMilestoneFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.DeleteMilestone(owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete %v/%v/milestone/%v err: %v", owner, repo, id, err))
	}

	return to.TextResult("Milestone deleted successfully")
}
