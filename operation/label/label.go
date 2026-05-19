package label

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/slim"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	LabelReadToolName  = "label_read"
	LabelWriteToolName = "label_write"
)

var (
	LabelReadTool = mcp.NewTool(
		LabelReadToolName,
		mcp.WithDescription("Read repo or org labels."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read labels")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("list_repo_labels", "get_repo_label", "list_org_labels")),
		mcp.WithString("owner", mcp.Description("for repo methods")),
		mcp.WithString("repo", mcp.Description("for repo methods")),
		mcp.WithString("org", mcp.Description("for org methods")),
		mcp.WithNumber("id", mcp.Description("label ID (for 'get_repo_label')")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	LabelWriteTool = mcp.NewTool(
		LabelWriteToolName,
		mcp.WithDescription("Write labels (repo or org): create, edit, delete."),
		mcp.WithToolAnnotation(annotation.Destructive("Create, update, or delete labels")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("create_repo_label", "edit_repo_label", "delete_repo_label", "create_org_label", "edit_org_label", "delete_org_label")),
		mcp.WithString("owner", mcp.Description("for repo methods")),
		mcp.WithString("repo", mcp.Description("for repo methods")),
		mcp.WithString("org", mcp.Description("for org methods")),
		mcp.WithNumber("id", mcp.Description("for edit/delete")),
		mcp.WithString("name", mcp.Description("required for create")),
		mcp.WithString("color", mcp.Description("hex (#RRGGBB); required for create")),
		mcp.WithString("description"),
		mcp.WithBoolean("exclusive", mcp.Description("exclusive (org only)")),
		mcp.WithBoolean("is_archived", mcp.Description("archived (repo only)")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    LabelReadTool,
		Handler: labelReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    LabelWriteTool,
		Handler: labelWriteFn,
	})
}

func labelReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list_repo_labels":
		return listRepoLabelsFn(ctx, req)
	case "get_repo_label":
		return getRepoLabelFn(ctx, req)
	case "list_org_labels":
		return listOrgLabelsFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func labelWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create_repo_label":
		return createRepoLabelFn(ctx, req)
	case "edit_repo_label":
		return editRepoLabelFn(ctx, req)
	case "delete_repo_label":
		return deleteRepoLabelFn(ctx, req)
	case "create_org_label":
		return createOrgLabelFn(ctx, req)
	case "edit_org_label":
		return editOrgLabelFn(ctx, req)
	case "delete_org_label":
		return deleteOrgLabelFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func listRepoLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	opt := gitea_sdk.ListLabelsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	labels, _, err := client.ListRepoLabels(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list %v/%v/labels err: %v", owner, repo, err))
	}
	return to.TextResult(slim.Labels(labels))
}

func getRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	label, _, err := client.GetRepoLabel(owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/label/%v err: %v", owner, repo, id, err))
	}
	return to.TextResult(slim.Label(label))
}

func createRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	color, err := params.GetString(req.GetArguments(), "color")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string) // Optional

	isArchived, _ := req.GetArguments()["is_archived"].(bool)

	opt := gitea_sdk.CreateLabelOption{
		Name:        name,
		Color:       color,
		Description: description,
		IsArchived:  isArchived,
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.CreateLabel(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/%v/label err: %v", owner, repo, err))
	}
	return to.TextResult(slim.Label(label))
}

func editRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	opt := gitea_sdk.EditLabelOption{
		Name:        params.GetOptionalStringPtr(args, "name"),
		Color:       params.GetOptionalStringPtr(args, "color"),
		Description: params.GetPresentStringPtr(args, "description"),
		IsArchived:  params.GetOptionalBoolPtr(args, "is_archived"),
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.EditLabel(owner, repo, id, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/label/%v err: %v", owner, repo, id, err))
	}
	return to.TextResult(slim.Label(label))
}

func deleteRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	_, err = client.DeleteLabel(owner, repo, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete %v/%v/label/%v err: %v", owner, repo, id, err))
	}
	return to.TextResult("Label deleted successfully")
}

func listOrgLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	opt := gitea_sdk.ListOrgLabelsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	labels, _, err := client.ListOrgLabels(org, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list %v/labels err: %v", org, err))
	}
	return to.TextResult(slim.Labels(labels))
}

func createOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	color, err := params.GetString(req.GetArguments(), "color")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string)
	exclusive, _ := req.GetArguments()["exclusive"].(bool)

	opt := gitea_sdk.CreateOrgLabelOption{
		Name:        name,
		Color:       color,
		Description: description,
		Exclusive:   exclusive,
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.CreateOrgLabel(org, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/labels err: %v", org, err))
	}
	return to.TextResult(slim.Label(label))
}

func editOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}

	args := req.GetArguments()
	opt := gitea_sdk.EditOrgLabelOption{
		Name:        params.GetOptionalStringPtr(args, "name"),
		Color:       params.GetOptionalStringPtr(args, "color"),
		Description: params.GetPresentStringPtr(args, "description"),
		Exclusive:   params.GetOptionalBoolPtr(args, "exclusive"),
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.EditOrgLabel(org, id, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/labels/%v err: %v", org, id, err))
	}
	return to.TextResult(slim.Label(label))
}

func deleteOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
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
	_, err = client.DeleteOrgLabel(org, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete %v/labels/%v err: %v", org, id, err))
	}
	return to.TextResult("Label deleted successfully")
}
