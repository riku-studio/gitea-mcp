package label

import (
	"context"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/params"
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
		mcp.WithDescription("Read label information. Use method 'list_repo_labels' to list repository labels, 'get_repo_label' to get a specific repo label, 'list_org_labels' to list organization labels."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read labels")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("list_repo_labels", "get_repo_label", "list_org_labels")),
		mcp.WithString("owner", mcp.Description("repository owner (required for repo methods)")),
		mcp.WithString("repo", mcp.Description("repository name (required for repo methods)")),
		mcp.WithString("org", mcp.Description("organization name (required for 'list_org')")),
		mcp.WithNumber("id", mcp.Description("label ID (required for 'get_repo')")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	LabelWriteTool = mcp.NewTool(
		LabelWriteToolName,
		mcp.WithDescription("Create, edit, or delete labels for repositories or organizations."),
		mcp.WithToolAnnotation(annotation.Destructive("Create, update, or delete labels")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("create_repo_label", "edit_repo_label", "delete_repo_label", "create_org_label", "edit_org_label", "delete_org_label")),
		mcp.WithString("owner", mcp.Description("repository owner (required for repo methods)")),
		mcp.WithString("repo", mcp.Description("repository name (required for repo methods)")),
		mcp.WithString("org", mcp.Description("organization name (required for org methods)")),
		mcp.WithNumber("id", mcp.Description("label ID (required for edit/delete methods)")),
		mcp.WithString("name", mcp.Description("label name (required for create, optional for edit)")),
		mcp.WithString("color", mcp.Description("label color hex code e.g. #RRGGBB (required for create, optional for edit)")),
		mcp.WithString("description", mcp.Description("label description")),
		mcp.WithBoolean("exclusive", mcp.Description("whether the label is exclusive (org labels only)")),
		mcp.WithBoolean("is_archived", mcp.Description("whether the label is archived (for create/edit repo label methods)")),
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
	log.Debugf("Called listRepoLabelsFn")
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
	return to.TextResult(slimLabels(labels))
}

func getRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getRepoLabelFn")
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
	return to.TextResult(slimLabel(label))
}

func createRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createRepoLabelFn")
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
	return to.TextResult(slimLabel(label))
}

func editRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called editRepoLabelFn")
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

	opt := gitea_sdk.EditLabelOption{}
	if name, ok := req.GetArguments()["name"].(string); ok {
		opt.Name = new(name)
	}
	if color, ok := req.GetArguments()["color"].(string); ok {
		opt.Color = new(color)
	}
	if description, ok := req.GetArguments()["description"].(string); ok {
		opt.Description = new(description)
	}
	if isArchived, ok := req.GetArguments()["is_archived"].(bool); ok {
		opt.IsArchived = &isArchived
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.EditLabel(owner, repo, id, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/label/%v err: %v", owner, repo, id, err))
	}
	return to.TextResult(slimLabel(label))
}

func deleteRepoLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deleteRepoLabelFn")
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
	log.Debugf("Called listOrgLabelsFn")
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
	return to.TextResult(slimLabels(labels))
}

func createOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createOrgLabelFn")
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
	return to.TextResult(slimLabel(label))
}

func editOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called editOrgLabelFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.EditOrgLabelOption{}
	if name, ok := req.GetArguments()["name"].(string); ok {
		opt.Name = new(name)
	}
	if color, ok := req.GetArguments()["color"].(string); ok {
		opt.Color = new(color)
	}
	if description, ok := req.GetArguments()["description"].(string); ok {
		opt.Description = new(description)
	}
	if exclusive, ok := req.GetArguments()["exclusive"].(bool); ok {
		opt.Exclusive = new(exclusive)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	label, _, err := client.EditOrgLabel(org, id, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/labels/%v err: %v", org, id, err))
	}
	return to.TextResult(slimLabel(label))
}

func deleteOrgLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deleteOrgLabelFn")
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
