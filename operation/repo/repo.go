package repo

import (
	"context"
	"errors"
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
	CreateRepoToolName   = "create_repo"
	ForkRepoToolName     = "fork_repo"
	ListMyReposToolName  = "list_my_repos"
	ListOrgReposToolName = "list_org_repos"
)

var (
	CreateRepoTool = mcp.NewTool(
		CreateRepoToolName,
		mcp.WithToolAnnotation(annotation.Write("Create a new repository")),
		mcp.WithString("name", mcp.Required()),
		mcp.WithString("description"),
		mcp.WithBoolean("private"),
		mcp.WithString("issue_labels"),
		mcp.WithBoolean("auto_init"),
		mcp.WithBoolean("template"),
		mcp.WithString("gitignores"),
		mcp.WithString("license"),
		mcp.WithString("readme"),
		mcp.WithString("default_branch"),
		mcp.WithString("trust_model", mcp.Enum("default", "collaborator", "committer", "collaboratorcommitter")),
		mcp.WithString("object_format_name", mcp.Enum("sha1", "sha256")),
		mcp.WithString("organization", mcp.Description("defaults to personal account")),
	)

	ForkRepoTool = mcp.NewTool(
		ForkRepoToolName,
		mcp.WithToolAnnotation(annotation.Write("Fork a repository")),
		mcp.WithString("user", mcp.Required(), mcp.Description("owner of source repo")),
		mcp.WithString("repo", mcp.Required()),
		mcp.WithString("organization", mcp.Description("target org")),
		mcp.WithString("name", mcp.Description("fork name")),
	)

	ListMyReposTool = mcp.NewTool(
		ListMyReposToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("List my repositories")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	ListOrgReposTool = mcp.NewTool(
		ListOrgReposToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("List organization repositories")),
		mcp.WithString("org", mcp.Required()),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(100), mcp.Min(1)),
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
	page, pageSize := params.GetPagination(req.GetArguments(), 100)
	opt := gitea_sdk.ListOrgReposOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
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
