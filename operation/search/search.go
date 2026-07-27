package search

import (
	"context"
	"fmt"
	"strings"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/slim"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New("search")

const (
	SearchUsersToolName    = "search_users"
	SearchOrgTeamsToolName = "search_org_teams"
	SearchReposToolName    = "search_repos"
	SearchIssuesToolName   = "search_issues"
)

var (
	SearchUsersTool = mcp.NewTool(
		SearchUsersToolName,
		mcp.WithDescription("Search for Gitea users by username or full name."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Search users")),
		mcp.WithString("query", mcp.Required()),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	SearOrgTeamsTool = mcp.NewTool(
		SearchOrgTeamsToolName,
		mcp.WithDescription("Search for teams within an organization by name, optionally including each team's description in the results."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Search organization teams")),
		mcp.WithString("org", mcp.Required()),
		mcp.WithString("query", mcp.Required()),
		mcp.WithBoolean("includeDescription"),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	SearchReposTool = mcp.NewTool(
		SearchReposToolName,
		mcp.WithDescription("Search for repositories by keyword, with filters for topic/description matching, owner, visibility, archived status, and sort order."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Search repositories")),
		mcp.WithString("query", mcp.Required()),
		mcp.WithBoolean("keywordIsTopic"),
		mcp.WithBoolean("keywordInDescription"),
		mcp.WithNumber("ownerID"),
		mcp.WithBoolean("isPrivate"),
		mcp.WithBoolean("isArchived"),
		mcp.WithString("sort"),
		mcp.WithString("order"),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	SearchIssuesTool = mcp.NewTool(
		SearchIssuesToolName,
		mcp.WithDescription("Search issues and PRs across repositories"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Search issues")),
		mcp.WithString("query", mcp.Required()),
		mcp.WithString("state", mcp.Enum("open", "closed", "all")),
		mcp.WithString("type", mcp.Enum("issues", "pulls")),
		mcp.WithString("labels", mcp.Description("comma-separated")),
		mcp.WithString("owner", mcp.Description("filter by owner")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    SearchUsersTool,
		Handler: UsersFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    SearOrgTeamsTool,
		Handler: OrgTeamsFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    SearchReposTool,
		Handler: ReposFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    SearchIssuesTool,
		Handler: IssuesFn,
	})
}

func UsersFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, err := params.GetString(req.GetArguments(), "query")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	opt := gitea_sdk.SearchUsersOption{
		KeyWord: keyword,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	users, _, err := client.Users.SearchUsers(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("search users err: %v", err))
	}
	return to.TextResult(slimUserDetails(users))
}

func OrgTeamsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org, err := params.GetString(req.GetArguments(), "org")
	if err != nil {
		return to.ErrorResult(err)
	}
	query, err := params.GetString(req.GetArguments(), "query")
	if err != nil {
		return to.ErrorResult(err)
	}
	includeDescription, _ := req.GetArguments()["includeDescription"].(bool)
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	opt := gitea_sdk.SearchTeamsOptions{
		Query:              query,
		IncludeDescription: includeDescription,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	teams, _, err := client.Organizations.SearchOrgTeams(ctx, org, &opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("search organization teams error: %v", err))
	}
	return to.TextResult(slimTeams(teams))
}

func ReposFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, err := params.GetString(req.GetArguments(), "query")
	if err != nil {
		return to.ErrorResult(err)
	}
	args := req.GetArguments()
	keywordIsTopic, _ := args["keywordIsTopic"].(bool)
	keywordInDescription, _ := args["keywordInDescription"].(bool)
	sort, _ := args["sort"].(string)
	order, _ := args["order"].(string)
	page, pageSize := params.GetPagination(args, 30)
	opt := gitea_sdk.SearchRepoOptions{
		Keyword:              keyword,
		KeywordIsTopic:       keywordIsTopic,
		KeywordInDescription: keywordInDescription,
		OwnerID:              params.GetOptionalInt(args, "ownerID", 0),
		IsPrivate:            params.GetOptionalBoolPtr(args, "isPrivate"),
		IsArchived:           params.GetOptionalBoolPtr(args, "isArchived"),
		Sort:                 sort,
		Order:                order,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	repos, _, err := client.Repositories.SearchRepos(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("search repos error: %v", err))
	}
	return to.TextResult(slim.Repos(repos))
}

func IssuesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query, err := params.GetString(args, "query")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(args, 30)

	opt := gitea_sdk.ListIssueOption{
		KeyWord: query,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if state, ok := args["state"].(string); ok {
		opt.State = gitea_sdk.StateType(state)
	}
	if issueType, ok := args["type"].(string); ok {
		opt.Type = gitea_sdk.IssueType(issueType)
	}
	if labels, ok := args["labels"].(string); ok && labels != "" {
		opt.Labels = strings.Split(labels, ",")
	}
	if owner, ok := args["owner"].(string); ok {
		opt.Owner = owner
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issues, _, err := client.Issues.ListIssues(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("search issues err: %v", err))
	}
	return to.TextResult(slimIssues(issues))
}
