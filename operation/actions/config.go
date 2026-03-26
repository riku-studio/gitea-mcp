package actions

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
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
		mcp.WithDescription("Read Actions secrets and variables configuration."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("list_repo_secrets", "list_org_secrets", "list_repo_variables", "get_repo_variable", "list_org_variables", "get_org_variable")),
		mcp.WithString("owner", mcp.Description("repository owner (required for repo methods)")),
		mcp.WithString("repo", mcp.Description("repository name (required for repo methods)")),
		mcp.WithString("org", mcp.Description("organization name (required for org methods)")),
		mcp.WithString("name", mcp.Description("variable name (required for get methods)")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	ActionsConfigWriteTool = mcp.NewTool(
		ActionsConfigWriteToolName,
		mcp.WithDescription("Manage Actions secrets and variables: create, update, or delete."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("upsert_repo_secret", "delete_repo_secret", "upsert_org_secret", "delete_org_secret", "create_repo_variable", "update_repo_variable", "delete_repo_variable", "create_org_variable", "update_org_variable", "delete_org_variable")),
		mcp.WithString("owner", mcp.Description("repository owner (required for repo methods)")),
		mcp.WithString("repo", mcp.Description("repository name (required for repo methods)")),
		mcp.WithString("org", mcp.Description("organization name (required for org methods)")),
		mcp.WithString("name", mcp.Description("secret or variable name (required for most methods)")),
		mcp.WithString("data", mcp.Description("secret value (required for upsert secret methods)")),
		mcp.WithString("value", mcp.Description("variable value (required for create/update variable methods)")),
		mcp.WithString("description", mcp.Description("description for secret or variable")),
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

// Secret functions

func listRepoActionSecretsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listRepoActionSecretsFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
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
	log.Debugf("Called upsertRepoActionSecretFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	data, err := params.GetString(req.GetArguments(), "data")
	if err != nil || data == "" {
		return to.ErrorResult(errors.New("data is required"))
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
	log.Debugf("Called deleteRepoActionSecretFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
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
	log.Debugf("Called listOrgActionSecretsFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
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
	log.Debugf("Called upsertOrgActionSecretFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	data, err := params.GetString(req.GetArguments(), "data")
	if err != nil || data == "" {
		return to.ErrorResult(errors.New("data is required"))
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
	log.Debugf("Called deleteOrgActionSecretFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}

	escapedOrg := url.PathEscape(org)
	escapedSecret := url.PathEscape(name)
	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("orgs/%s/actions/secrets/%s", escapedOrg, escapedSecret), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete org action secret err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "secret deleted"})
}

// Variable functions

func listRepoActionVariablesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listRepoActionVariablesFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
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
	log.Debugf("Called getRepoActionVariableFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
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
	log.Debugf("Called createRepoActionVariableFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil || value == "" {
		return to.ErrorResult(errors.New("value is required"))
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
	log.Debugf("Called updateRepoActionVariableFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil || value == "" {
		return to.ErrorResult(errors.New("value is required"))
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
	log.Debugf("Called deleteRepoActionVariableFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil || owner == "" {
		return to.ErrorResult(errors.New("owner is required"))
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil || repo == "" {
		return to.ErrorResult(errors.New("repo is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
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
	log.Debugf("Called listOrgActionVariablesFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
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
	log.Debugf("Called getOrgActionVariableFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
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
	log.Debugf("Called createOrgActionVariableFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil || value == "" {
		return to.ErrorResult(errors.New("value is required"))
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
	log.Debugf("Called updateOrgActionVariableFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}
	value, err := params.GetString(req.GetArguments(), "value")
	if err != nil || value == "" {
		return to.ErrorResult(errors.New("value is required"))
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
	log.Debugf("Called deleteOrgActionVariableFn")
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil || org == "" {
		return to.ErrorResult(errors.New("org is required"))
	}
	name, err := params.GetString(req.GetArguments(), "name")
	if err != nil || name == "" {
		return to.ErrorResult(errors.New("name is required"))
	}

	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("orgs/%s/actions/variables/%s", url.PathEscape(org), url.PathEscape(name)), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete org action variable err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "variable deleted"})
}
