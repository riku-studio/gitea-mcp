package actions

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"

	gitea_sdk "code.gitea.io/sdk/gitea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ActionsConfigReadToolName  = "actions_config_read"
	ActionsConfigWriteToolName = "actions_config_write"
)

type secretMeta struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
}

func toSecretMetas(secrets []*gitea_sdk.Secret) []secretMeta {
	metas := make([]secretMeta, 0, len(secrets))
	for _, s := range secrets {
		if s == nil {
			continue
		}
		metas = append(metas, secretMeta{
			Name:        s.Name,
			Description: s.Description,
			CreatedAt:   s.Created,
		})
	}
	return metas
}

var (
	ActionsConfigReadTool = mcp.NewTool(
		ActionsConfigReadToolName,
		mcp.WithDescription("Read Actions secrets and variables."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read Actions secrets and variables")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("list_repo_secrets", "list_org_secrets", "list_repo_variables", "get_repo_variable", "list_org_variables", "get_org_variable")),
		mcp.WithString("owner", mcp.Description("for repo methods")),
		mcp.WithString("repo", mcp.Description("for repo methods")),
		mcp.WithString("org", mcp.Description("for org methods")),
		mcp.WithString("name", mcp.Description("for get methods")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	ActionsConfigWriteTool = mcp.NewTool(
		ActionsConfigWriteToolName,
		mcp.WithDescription("Write Actions secrets and variables: upsert, create, update, delete."),
		mcp.WithToolAnnotation(annotation.Destructive("Manage Actions secrets and variables")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("upsert_repo_secret", "delete_repo_secret", "upsert_org_secret", "delete_org_secret", "create_repo_variable", "update_repo_variable", "delete_repo_variable", "create_org_variable", "update_org_variable", "delete_org_variable")),
		mcp.WithString("owner", mcp.Description("for repo methods")),
		mcp.WithString("repo", mcp.Description("for repo methods")),
		mcp.WithString("org", mcp.Description("for org methods")),
		mcp.WithString("name", mcp.Description("secret or variable name")),
		mcp.WithString("data", mcp.Description("secret value (upsert)")),
		mcp.WithString("value", mcp.Description("variable value")),
		mcp.WithString("description"),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{Tool: ActionsConfigReadTool, Handler: configReadFn})
	Tool.RegisterWrite(server.ServerTool{Tool: ActionsConfigWriteTool, Handler: configWriteFn})
}

func configReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list_repo_secrets":
		return listRepoActionSecretsFn(ctx, req)
	case "list_org_secrets":
		return listOrgActionSecretsFn(ctx, req)
	case "list_repo_variables":
		return listRepoActionVariablesFn(ctx, req)
	case "get_repo_variable":
		return getRepoActionVariableFn(ctx, req)
	case "list_org_variables":
		return listOrgActionVariablesFn(ctx, req)
	case "get_org_variable":
		return getOrgActionVariableFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func configWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "upsert_repo_secret":
		return upsertRepoActionSecretFn(ctx, req)
	case "delete_repo_secret":
		return deleteRepoActionSecretFn(ctx, req)
	case "upsert_org_secret":
		return upsertOrgActionSecretFn(ctx, req)
	case "delete_org_secret":
		return deleteOrgActionSecretFn(ctx, req)
	case "create_repo_variable":
		return createRepoActionVariableFn(ctx, req)
	case "update_repo_variable":
		return updateRepoActionVariableFn(ctx, req)
	case "delete_repo_variable":
		return deleteRepoActionVariableFn(ctx, req)
	case "create_org_variable":
		return createOrgActionVariableFn(ctx, req)
	case "update_org_variable":
		return updateOrgActionVariableFn(ctx, req)
	case "delete_org_variable":
		return deleteOrgActionVariableFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func listRepoActionSecretsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	secrets, _, err := client.ListRepoActionSecret(owner, repo, gitea_sdk.ListRepoActionSecretOption{
		ListOptions: gitea_sdk.ListOptions{Page: page, PageSize: pageSize},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list repo action secrets err: %v", err))
	}

	return to.TextResult(toSecretMetas(secrets))
}

func upsertRepoActionSecretFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	data, err := params.GetString(req.GetArguments(), "data")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.CreateRepoActionSecret(owner, repo, gitea_sdk.CreateSecretOption{
		Name:        name,
		Data:        data,
		Description: description,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("upsert repo action secret err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "secret upserted", "status": resp.StatusCode})
}

func deleteRepoActionSecretFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.DeleteRepoActionSecret(owner, repo, name)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete repo action secret err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "secret deleted", "status": resp.StatusCode})
}

func listOrgActionSecretsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	secrets, _, err := client.ListOrgActionSecret(org, gitea_sdk.ListOrgActionSecretOption{
		ListOptions: gitea_sdk.ListOptions{Page: page, PageSize: pageSize},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list org action secrets err: %v", err))
	}

	return to.TextResult(toSecretMetas(secrets))
}

func upsertOrgActionSecretFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	data, err := params.GetString(req.GetArguments(), "data")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.CreateOrgActionSecret(org, gitea_sdk.CreateSecretOption{
		Name:        name,
		Data:        data,
		Description: description,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("upsert org action secret err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "secret upserted", "status": resp.StatusCode})
}

func deleteOrgActionSecretFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}

	escapedOrg := url.PathEscape(org)
	escapedSecret := url.PathEscape(name)
	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("orgs/%s/actions/secrets/%s", escapedOrg, escapedSecret), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete org action secret err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "secret deleted"})
}

func listRepoActionVariablesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("repos/%s/%s/actions/variables", url.PathEscape(owner), url.PathEscape(repo)), query, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list repo action variables err: %v", err))
	}
	return to.TextResult(result)
}

func getRepoActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	variable, _, err := client.GetRepoActionVariable(owner, repo, name)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get repo action variable err: %v", err))
	}
	return to.TextResult(variable)
}

func createRepoActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.CreateRepoActionVariable(owner, repo, name, value)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create repo action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable created", "status": resp.StatusCode})
}

func updateRepoActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.UpdateRepoActionVariable(owner, repo, name, value)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("update repo action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable updated", "status": resp.StatusCode})
}

func deleteRepoActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.DeleteRepoActionVariable(owner, repo, name)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete repo action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable deleted", "status": resp.StatusCode})
}

func listOrgActionVariablesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	variables, _, err := client.ListOrgActionVariable(org, gitea_sdk.ListOrgActionVariableOption{
		ListOptions: gitea_sdk.ListOptions{Page: page, PageSize: pageSize},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list org action variables err: %v", err))
	}
	return to.TextResult(variables)
}

func getOrgActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	variable, _, err := client.GetOrgActionVariable(org, name)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get org action variable err: %v", err))
	}
	return to.TextResult(variable)
}

func createOrgActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.CreateOrgActionVariable(org, gitea_sdk.CreateOrgActionVariableOption{
		Name:        name,
		Value:       value,
		Description: description,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create org action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable created", "status": resp.StatusCode})
}

func updateOrgActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := req.GetArguments()["description"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	resp, err := client.UpdateOrgActionVariable(org, name, gitea_sdk.UpdateOrgActionVariableOption{
		Value:       value,
		Description: description,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("update org action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable updated", "status": resp.StatusCode})
}

func deleteOrgActionVariableFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil {
		return to.ErrorResult(err)
	}

	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("orgs/%s/actions/variables/%s", url.PathEscape(org), url.PathEscape(name)), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete org action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable deleted"})
}
