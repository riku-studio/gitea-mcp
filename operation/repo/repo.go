package repo

import (
	"context"
	"errors"
	"fmt"

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
	CreateRepoToolName   = "create_repo"
	ForkRepoToolName     = "fork_repo"
	ListMyReposToolName  = "list_my_repos"
	ListOrgReposToolName = "list_org_repos"
)

var (
	CreateRepoTool = mcp.NewTool(
		CreateRepoToolName,
		mcp.WithDescription("Create repository in personal account or organization"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the repository to create")),
		mcp.WithString("description", mcp.Description("Description of the repository to create")),
		mcp.WithBoolean("private", mcp.Description("Whether the repository is private")),
		mcp.WithString("issue_labels", mcp.Description("Issue Label set to use")),
		mcp.WithBoolean("auto_init", mcp.Description("Whether the repository should be auto-intialized?")),
		mcp.WithBoolean("template", mcp.Description("Whether the repository is template")),
		mcp.WithString("gitignores", mcp.Description("Gitignores to use")),
		mcp.WithString("license", mcp.Description("License to use")),
		mcp.WithString("readme", mcp.Description("Readme of the repository to create")),
		mcp.WithString("default_branch", mcp.Description("DefaultBranch of the repository (used when initializes and in template)")),
		mcp.WithString("trust_model", mcp.Description("Trust model for verifying GPG signatures"), mcp.Enum("default", "collaborator", "committer", "collaboratorcommitter")),
		mcp.WithString("object_format_name", mcp.Description("Object format: sha1 or sha256"), mcp.Enum("sha1", "sha256")),
		mcp.WithString("organization", mcp.Description("Organization name to create repository in (optional - defaults to personal account)")),
	)

	ForkRepoTool = mcp.NewTool(
		ForkRepoToolName,
		mcp.WithDescription("Fork repository"),
		mcp.WithString("user", mcp.Required(), mcp.Description("User name of the repository to fork")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name to fork")),
		mcp.WithString("organization", mcp.Description("Organization name to fork")),
		mcp.WithString("name", mcp.Description("Name of the forked repository")),
	)

	ListMyReposTool = mcp.NewTool(
		ListMyReposToolName,
		mcp.WithDescription("List my repositories"),
		mcp.WithNumber("page", mcp.Required(), mcp.Description("Page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Required(), mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	ListOrgReposTool = mcp.NewTool(
		ListOrgReposToolName,
		mcp.WithDescription("List repositories of an organization"),
		mcp.WithString("org", mcp.Required(), mcp.Description("Organization name")),
		mcp.WithNumber("page", mcp.Required(), mcp.Description("Page number"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Required(), mcp.Description("Page size number"), mcp.DefaultNumber(100), mcp.Min(1)),
	)
)

func init() {
	Tool.RegisterWrite(server.ServerTool{
		Tool:    CreateRepoTool,
		Handler: CreateRepoFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    ForkRepoTool,
		Handler: ForkRepoFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListMyReposTool,
		Handler: ListMyReposFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListOrgReposTool,
		Handler: ListOrgReposFn,
	})
}

func CreateRepoFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called CreateRepoFn")
	args := req.GetArguments()
	name, err := params.GetString(args, "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	description, _ := args["description"].(string)
	private, _ := args["private"].(bool)
	issueLabels, _ := args["issue_labels"].(string)
	autoInit, _ := args["auto_init"].(bool)
	template, _ := args["template"].(bool)
	gitignores, _ := args["gitignores"].(string)
	license, _ := args["license"].(string)
	readme, _ := args["readme"].(string)
	defaultBranch, _ := args["default_branch"].(string)
	trustModel, _ := args["trust_model"].(string)
	objectFormatName, _ := args["object_format_name"].(string)
	organization, _ := args["organization"].(string)

	opt := gitea_sdk.CreateRepoOption{
		Name:             name,
		Description:      description,
		Private:          private,
		IssueLabels:      issueLabels,
		AutoInit:         autoInit,
		Template:         template,
		Gitignores:       gitignores,
		License:          license,
		Readme:           readme,
		DefaultBranch:    defaultBranch,
		TrustModel:       gitea_sdk.TrustModel(trustModel),
		ObjectFormatName: objectFormatName,
	}

	var repo *gitea_sdk.Repository
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	if organization != "" {
		repo, _, err = client.CreateOrgRepo(organization, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("create organization repository '%s' in '%s' err: %v", name, organization, err))
		}
	} else {
		repo, _, err = client.CreateRepo(opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("create repository '%s' err: %v", name, err))
		}
	}
	return to.TextResult(slimRepo(repo))
}

func ForkRepoFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ForkRepoFn")
	args := req.GetArguments()
	user, err := params.GetString(args, "user")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	organization, ok := args["organization"].(string)
	organizationPtr := new(organization)
	if !ok || organization == "" {
		organizationPtr = nil
	}
	name, ok := args["name"].(string)
	namePtr := new(name)
	if !ok || name == "" {
		namePtr = nil
	}
	opt := gitea_sdk.CreateForkOption{
		Organization: organizationPtr,
		Name:         namePtr,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, _, err = client.CreateFork(user, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("fork repository error: %v", err))
	}
	return to.TextResult("Fork success")
}

func ListMyReposFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListMyReposFn")
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	opt := gitea_sdk.ListReposOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	repos, _, err := client.ListMyRepos(opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list my repositories error: %v", err))
	}

	return to.TextResult(slimRepos(repos))
}

func ListOrgReposFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListOrgReposFn")
	org, ok := req.GetArguments()["org"].(string)
	if !ok {
		return to.ErrorResult(errors.New("organization name is required"))
	}
	page, ok := req.GetArguments()["page"].(float64)
	if !ok {
		page = 1
	}
	pageSize, ok := req.GetArguments()["pageSize"].(float64)
	if !ok {
		pageSize = 100
	}
	opt := gitea_sdk.ListOrgReposOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     int(page),
			PageSize: int(pageSize),
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	repos, _, err := client.ListOrgRepos(org, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list organization '%s' repositories error: %v", org, err))
	}
	return to.TextResult(repos)
}
