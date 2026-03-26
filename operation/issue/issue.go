package issue

import (
	"context"
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
	ListRepoIssuesToolName = "list_issues"
	IssueReadToolName      = "issue_read"
	IssueWriteToolName     = "issue_write"
)

var (
	ListRepoIssuesTool = mcp.NewTool(
		ListRepoIssuesToolName,
		mcp.WithDescription("List repository issues"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("state", mcp.Description("issue state"), mcp.DefaultString("all")),
		mcp.WithArray("labels", mcp.Description("filter by label names"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("since", mcp.Description("filter issues updated after this ISO 8601 timestamp")),
		mcp.WithString("before", mcp.Description("filter issues updated before this ISO 8601 timestamp")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	IssueReadTool = mcp.NewTool(
		IssueReadToolName,
		mcp.WithDescription("Get information about a specific issue. Use method 'get' for issue details, 'get_comments' for issue comments, 'get_labels' for issue labels."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("get", "get_comments", "get_labels")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("index", mcp.Required(), mcp.Description("repository issue index")),
	)

	IssueWriteTool = mcp.NewTool(
		IssueWriteToolName,
		mcp.WithDescription("Create or update issues and comments, manage labels. Use method 'create' to create an issue, 'update' to edit, 'add_comment'/'edit_comment' for comments, 'add_labels'/'remove_label'/'replace_labels'/'clear_labels' for label management."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("create", "update", "add_comment", "edit_comment", "add_labels", "remove_label", "replace_labels", "clear_labels")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("index", mcp.Description("issue index (required for all methods except 'create')")),
		mcp.WithString("title", mcp.Description("issue title (required for 'create')")),
		mcp.WithString("body", mcp.Description("issue/comment body (required for 'create', 'add_comment', 'edit_comment')")),
		mcp.WithArray("assignees", mcp.Description("usernames to assign (for 'create', 'update')"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithNumber("milestone", mcp.Description("milestone number (for 'create', 'update')")),
		mcp.WithString("state", mcp.Description("issue state, one of open, closed, all (for 'update')")),
		mcp.WithNumber("commentID", mcp.Description("id of issue comment (required for 'edit_comment')")),
		mcp.WithArray("labels", mcp.Description("array of label IDs (for 'create', 'add_labels', 'replace_labels')"), mcp.Items(map[string]any{"type": "number"})),
		mcp.WithNumber("label_id", mcp.Description("label ID to remove (required for 'remove_label')")),
		mcp.WithString("ref", mcp.Description("branch name to associate with the issue (for 'create', 'update')")),
		mcp.WithString("deadline", mcp.Description("due date in ISO 8601 format (for 'create', 'update')")),
		mcp.WithBoolean("remove_deadline", mcp.Description("unset due date (for 'update')")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListRepoIssuesTool,
		Handler: listRepoIssuesFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    IssueReadTool,
		Handler: issueReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    IssueWriteTool,
		Handler: issueWriteFn,
	})
}

func issueReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "get":
		return getIssueByIndexFn(ctx, req)
	case "get_comments":
		return getIssueCommentsByIndexFn(ctx, req)
	case "get_labels":
		return getIssueLabelsFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func issueWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create":
		return createIssueFn(ctx, req)
	case "update":
		return editIssueFn(ctx, req)
	case "add_comment":
		return createIssueCommentFn(ctx, req)
	case "edit_comment":
		return editIssueCommentFn(ctx, req)
	case "add_labels":
		return addIssueLabelsFn(ctx, req)
	case "remove_label":
		return removeIssueLabelFn(ctx, req)
	case "replace_labels":
		return replaceIssueLabelsFn(ctx, req)
	case "clear_labels":
		return clearIssueLabelsFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func getIssueByIndexFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getIssueByIndexFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issue, _, err := client.GetIssue(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/issue/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimIssue(issue))
}

func listRepoIssuesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListIssuesFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	state, ok := req.GetArguments()["state"].(string)
	if !ok {
		state = "all"
	}
	labels := params.GetStringSlice(req.GetArguments(), "labels")
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	opt := gitea_sdk.ListIssueOption{
		State:  gitea_sdk.StateType(state),
		Labels: labels,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if t := params.GetOptionalTime(req.GetArguments(), "since"); t != nil {
		opt.Since = *t
	}
	if t := params.GetOptionalTime(req.GetArguments(), "before"); t != nil {
		opt.Before = *t
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issues, _, err := client.ListRepoIssues(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/issues err: %v", owner, repo, err))
	}
	return to.TextResult(slimIssues(issues))
}

func createIssueFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createIssueFn")
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
	body, err := params.GetString(req.GetArguments(), "body")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	opt := gitea_sdk.CreateIssueOption{
		Title: title,
		Body:  body,
	}
	opt.Assignees = params.GetStringSlice(req.GetArguments(), "assignees")
	if val, exists := req.GetArguments()["milestone"]; exists {
		if milestone, ok := params.ToInt64(val); ok {
			opt.Milestone = milestone
		}
	}
	if labelIDs, err := params.GetInt64Slice(req.GetArguments(), "labels"); err == nil {
		opt.Labels = labelIDs
	}
	if ref, ok := req.GetArguments()["ref"].(string); ok {
		opt.Ref = ref
	}
	opt.Deadline = params.GetOptionalTime(req.GetArguments(), "deadline")
	issue, _, err := client.CreateIssue(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/%v/issue err: %v", owner, repo, err))
	}

	return to.TextResult(slimIssue(issue))
}

func createIssueCommentFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createIssueCommentFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	body, err := params.GetString(req.GetArguments(), "body")
	if err != nil {
		return to.ErrorResult(err)
	}
	opt := gitea_sdk.CreateIssueCommentOption{
		Body: body,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issueComment, _, err := client.CreateIssueComment(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/%v/issue/%v/comment err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimComment(issueComment))
}

func editIssueFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called editIssueFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.EditIssueOption{}

	title, ok := req.GetArguments()["title"].(string)
	if ok {
		opt.Title = title
	}
	body, ok := req.GetArguments()["body"].(string)
	if ok {
		opt.Body = new(body)
	}
	opt.Assignees = params.GetStringSlice(req.GetArguments(), "assignees")
	if val, exists := req.GetArguments()["milestone"]; exists {
		if milestone, ok := params.ToInt64(val); ok {
			opt.Milestone = new(milestone)
		}
	}
	state, ok := req.GetArguments()["state"].(string)
	if ok {
		opt.State = new(gitea_sdk.StateType(state))
	}
	if ref, ok := req.GetArguments()["ref"].(string); ok {
		opt.Ref = &ref
	}
	opt.Deadline = params.GetOptionalTime(req.GetArguments(), "deadline")
	if removeDeadline, ok := req.GetArguments()["remove_deadline"].(bool); ok {
		opt.RemoveDeadline = &removeDeadline
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issue, _, err := client.EditIssue(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/issue/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimIssue(issue))
}

func editIssueCommentFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called editIssueCommentFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	commentID, err := params.GetIndex(req.GetArguments(), "commentID")
	if err != nil {
		return to.ErrorResult(err)
	}
	body, err := params.GetString(req.GetArguments(), "body")
	if err != nil {
		return to.ErrorResult(err)
	}
	opt := gitea_sdk.EditIssueCommentOption{
		Body: body,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issueComment, _, err := client.EditIssueComment(owner, repo, commentID, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/issues/comments/%v err: %v", owner, repo, commentID, err))
	}

	return to.TextResult(slimComment(issueComment))
}

func getIssueCommentsByIndexFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getIssueCommentsByIndexFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	opt := gitea_sdk.ListIssueCommentOptions{}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issue, _, err := client.ListIssueComments(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/issues/%v/comments err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimComments(issue))
}

func getIssueLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getIssueLabelsFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	labels, _, err := client.GetIssueLabels(owner, repo, index, gitea_sdk.ListLabelsOptions{})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/issues/%v/labels err: %v", owner, repo, index, err))
	}
	return to.TextResult(slimLabels(labels))
}

// Issue label operations (moved from label package)

func addIssueLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called addIssueLabelsFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	labels, err := params.GetInt64Slice(req.GetArguments(), "labels")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issueLabels, _, err := client.AddIssueLabels(owner, repo, index, gitea_sdk.IssueLabelsOption{Labels: labels})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("add labels to %v/%v/issue/%v err: %v", owner, repo, index, err))
	}
	return to.TextResult(slimLabels(issueLabels))
}

func replaceIssueLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called replaceIssueLabelsFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	labels, err := params.GetInt64Slice(req.GetArguments(), "labels")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	issueLabels, _, err := client.ReplaceIssueLabels(owner, repo, index, gitea_sdk.IssueLabelsOption{Labels: labels})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("replace labels on %v/%v/issue/%v err: %v", owner, repo, index, err))
	}
	return to.TextResult(slimLabels(issueLabels))
}

func clearIssueLabelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called clearIssueLabelsFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.ClearIssueLabels(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("clear labels on %v/%v/issue/%v err: %v", owner, repo, index, err))
	}
	return to.TextResult("Labels cleared successfully")
}

func removeIssueLabelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called removeIssueLabelFn")
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	labelID, err := params.GetIndex(req.GetArguments(), "label_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.DeleteIssueLabel(owner, repo, index, labelID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("remove label %v from %v/%v/issue/%v err: %v", labelID, owner, repo, index, err))
	}
	return to.TextResult("Label removed successfully")
}
