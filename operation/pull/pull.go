package pull

import (
	"context"
	"fmt"
	"strings"

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
	ListRepoPullRequestsToolName   = "list_pull_requests"
	PullRequestReadToolName        = "pull_request_read"
	PullRequestWriteToolName       = "pull_request_write"
	PullRequestReviewWriteToolName = "pull_request_review_write"
)

var (
	ListRepoPullRequestsTool = mcp.NewTool(
		ListRepoPullRequestsToolName,
		mcp.WithDescription("List repository pull requests"),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("state", mcp.Description("state"), mcp.Enum("open", "closed", "all"), mcp.DefaultString("all")),
		mcp.WithString("sort", mcp.Description("sort"), mcp.Enum("oldest", "recentupdate", "leastupdate", "mostcomment", "leastcomment", "priority"), mcp.DefaultString("recentupdate")),
		mcp.WithNumber("milestone", mcp.Description("milestone")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	PullRequestReadTool = mcp.NewTool(
		PullRequestReadToolName,
		mcp.WithDescription("Get pull request information. Use method 'get' for PR details, 'get_diff' for diff, 'get_reviews'/'get_review'/'get_review_comments' for review data."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("get", "get_diff", "get_reviews", "get_review", "get_review_comments")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("index", mcp.Required(), mcp.Description("pull request index")),
		mcp.WithNumber("review_id", mcp.Description("review ID (required for 'get_review', 'get_review_comments')")),
		mcp.WithBoolean("binary", mcp.Description("whether to include binary file changes (for 'get_diff')")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	PullRequestWriteTool = mcp.NewTool(
		PullRequestWriteToolName,
		mcp.WithDescription("Create, update, or merge pull requests, manage reviewers."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("create", "update", "merge", "add_reviewers", "remove_reviewers")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("index", mcp.Description("pull request index (required for all methods except 'create')")),
		mcp.WithString("title", mcp.Description("PR title (required for 'create', optional for 'update', 'merge')")),
		mcp.WithString("body", mcp.Description("PR body (required for 'create', optional for 'update')")),
		mcp.WithString("head", mcp.Description("PR head branch (required for 'create')")),
		mcp.WithString("base", mcp.Description("PR base branch (required for 'create', optional for 'update')")),
		mcp.WithString("assignee", mcp.Description("username to assign (for 'update')")),
		mcp.WithArray("assignees", mcp.Description("usernames to assign (for 'update')"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithNumber("milestone", mcp.Description("milestone number (for 'update')")),
		mcp.WithString("state", mcp.Description("PR state (for 'update')"), mcp.Enum("open", "closed")),
		mcp.WithBoolean("allow_maintainer_edit", mcp.Description("allow maintainer to edit (for 'update')")),
		mcp.WithArray("labels", mcp.Description("array of label IDs (for 'create', 'update')"), mcp.Items(map[string]any{"type": "number"})),
		mcp.WithString("deadline", mcp.Description("due date in ISO 8601 format (for 'create', 'update')")),
		mcp.WithBoolean("remove_deadline", mcp.Description("unset due date (for 'update')")),
		mcp.WithString("merge_style", mcp.Description("merge style (for 'merge')"), mcp.Enum("merge", "rebase", "rebase-merge", "squash", "fast-forward-only"), mcp.DefaultString("merge")),
		mcp.WithString("message", mcp.Description("merge commit message (for 'merge') or dismissal reason")),
		mcp.WithBoolean("delete_branch", mcp.Description("delete branch after merge (for 'merge')")),
		mcp.WithBoolean("force_merge", mcp.Description("force merge even if checks are not passing (for 'merge')")),
		mcp.WithBoolean("merge_when_checks_succeed", mcp.Description("auto-merge when checks succeed (for 'merge')")),
		mcp.WithString("head_commit_id", mcp.Description("expected head commit SHA for merge conflict detection (for 'merge')")),
		mcp.WithArray("reviewers", mcp.Description("reviewer usernames (for 'add_reviewers', 'remove_reviewers')"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("team_reviewers", mcp.Description("team reviewer names (for 'add_reviewers', 'remove_reviewers')"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithBoolean("draft", mcp.Description("mark PR as draft (for 'create', 'update'). Gitea uses a 'WIP: ' title prefix for drafts.")),
	)

	PullRequestReviewWriteTool = mcp.NewTool(
		PullRequestReviewWriteToolName,
		mcp.WithDescription("Manage pull request reviews: create, submit, delete, or dismiss."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("create", "submit", "delete", "dismiss")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithNumber("index", mcp.Required(), mcp.Description("pull request index")),
		mcp.WithNumber("review_id", mcp.Description("review ID (required for 'submit', 'delete', 'dismiss')")),
		mcp.WithString("state", mcp.Description("review state"), mcp.Enum("APPROVED", "REQUEST_CHANGES", "COMMENT", "PENDING")),
		mcp.WithString("body", mcp.Description("review body/comment")),
		mcp.WithString("commit_id", mcp.Description("commit SHA to review (for 'create')")),
		mcp.WithString("message", mcp.Description("dismissal reason (for 'dismiss')")),
		mcp.WithArray("comments", mcp.Description("inline review comments (for 'create')"), mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":         map[string]any{"type": "string", "description": "file path to comment on"},
				"body":         map[string]any{"type": "string", "description": "comment body"},
				"old_line_num": map[string]any{"type": "number", "description": "line number in the old file (for deletions/changes)"},
				"new_line_num": map[string]any{"type": "number", "description": "line number in the new file (for additions/changes)"},
			},
		})),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    ListRepoPullRequestsTool,
		Handler: listRepoPullRequestsFn,
	})
	Tool.RegisterRead(server.ServerTool{
		Tool:    PullRequestReadTool,
		Handler: pullRequestReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    PullRequestWriteTool,
		Handler: pullRequestWriteFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    PullRequestReviewWriteTool,
		Handler: pullRequestReviewWriteFn,
	})
}

func pullRequestReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "get":
		return getPullRequestByIndexFn(ctx, req)
	case "get_diff":
		return getPullRequestDiffFn(ctx, req)
	case "get_reviews":
		return listPullRequestReviewsFn(ctx, req)
	case "get_review":
		return getPullRequestReviewFn(ctx, req)
	case "get_review_comments":
		return listPullRequestReviewCommentsFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func pullRequestWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create":
		return createPullRequestFn(ctx, req)
	case "update":
		return editPullRequestFn(ctx, req)
	case "merge":
		return mergePullRequestFn(ctx, req)
	case "add_reviewers":
		return createPullRequestReviewerFn(ctx, req)
	case "remove_reviewers":
		return deletePullRequestReviewerFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func pullRequestReviewWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create":
		return createPullRequestReviewFn(ctx, req)
	case "submit":
		return submitPullRequestReviewFn(ctx, req)
	case "delete":
		return deletePullRequestReviewFn(ctx, req)
	case "dismiss":
		return dismissPullRequestReviewFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func getPullRequestByIndexFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getPullRequestByIndexFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	pr, _, err := client.GetPullRequest(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimPullRequest(pr))
}

func getPullRequestDiffFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getPullRequestDiffFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	binary, _ := args["binary"].(bool)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	diffBytes, _, err := client.GetPullRequestDiff(owner, repo, index, gitea_sdk.PullRequestDiffOptions{
		Binary: binary,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/pr/%v diff err: %v", owner, repo, index, err))
	}

	return to.TextResult(string(diffBytes))
}

func listRepoPullRequestsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called ListRepoPullRequests")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	state, _ := args["state"].(string)
	sort := params.GetOptionalString(args, "sort", "recentupdate")
	milestone := params.GetOptionalInt(args, "milestone", 0)
	page, pageSize := params.GetPagination(args, 30)
	opt := gitea_sdk.ListPullRequestsOptions{
		State:     gitea_sdk.StateType(state),
		Sort:      sort,
		Milestone: milestone,
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	pullRequests, _, err := client.ListRepoPullRequests(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list %v/%v/pull_requests err: %v", owner, repo, err))
	}

	return to.TextResult(slimPullRequests(pullRequests))
}

// defaultWIPPrefixes are the default Gitea title prefixes that mark a PR as
// work-in-progress / draft. Gitea matches these case-insensitively.
var defaultWIPPrefixes = []string{"WIP:", "[WIP]"}

// applyDraftPrefix adds or removes a WIP title prefix that Gitea uses to mark
// pull requests as drafts. When the title already carries a recognized prefix
// and isDraft is true, the title is returned unchanged to avoid normalization.
func applyDraftPrefix(title string, isDraft bool) string {
	for _, prefix := range defaultWIPPrefixes {
		if len(title) >= len(prefix) && strings.EqualFold(title[:len(prefix)], prefix) {
			if isDraft {
				return title
			}
			return strings.TrimLeft(title[len(prefix):], " ")
		}
	}
	if isDraft {
		return "WIP: " + title
	}
	return title
}

func createPullRequestFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createPullRequestFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	title, err := params.GetString(args, "title")
	if err != nil {
		return to.ErrorResult(err)
	}
	body, err := params.GetString(args, "body")
	if err != nil {
		return to.ErrorResult(err)
	}
	head, err := params.GetString(args, "head")
	if err != nil {
		return to.ErrorResult(err)
	}
	base, err := params.GetString(args, "base")
	if err != nil {
		return to.ErrorResult(err)
	}

	if draft, ok := args["draft"].(bool); ok {
		title = applyDraftPrefix(title, draft)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	opt := gitea_sdk.CreatePullRequestOption{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}
	if labelIDs, err := params.GetInt64Slice(args, "labels"); err == nil {
		opt.Labels = labelIDs
	}
	opt.Deadline = params.GetOptionalTime(args, "deadline")
	pr, _, err := client.CreatePullRequest(owner, repo, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create %v/%v/pull_request err: %v", owner, repo, err))
	}

	return to.TextResult(slimPullRequest(pr))
}

func createPullRequestReviewerFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createPullRequestReviewerFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	reviewers := params.GetStringSlice(args, "reviewers")
	teamReviewers := params.GetStringSlice(args, "team_reviewers")

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	_, err = client.CreateReviewRequests(owner, repo, index, gitea_sdk.PullReviewRequestOptions{
		Reviewers:     reviewers,
		TeamReviewers: teamReviewers,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create review requests for %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	successMsg := map[string]any{
		"message":        "Successfully created review requests",
		"reviewers":      reviewers,
		"team_reviewers": teamReviewers,
		"pr_index":       index,
		"repository":     fmt.Sprintf("%s/%s", owner, repo),
	}

	return to.TextResult(successMsg)
}

func deletePullRequestReviewerFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deletePullRequestReviewerFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	reviewers := params.GetStringSlice(args, "reviewers")
	teamReviewers := params.GetStringSlice(args, "team_reviewers")

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	_, err = client.DeleteReviewRequests(owner, repo, index, gitea_sdk.PullReviewRequestOptions{
		Reviewers:     reviewers,
		TeamReviewers: teamReviewers,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete review requests for %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	successMsg := map[string]any{
		"message":        "Successfully deleted review requests",
		"reviewers":      reviewers,
		"team_reviewers": teamReviewers,
		"pr_index":       index,
		"repository":     fmt.Sprintf("%s/%s", owner, repo),
	}

	return to.TextResult(successMsg)
}

func listPullRequestReviewsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listPullRequestReviewsFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(args, 30)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	reviews, _, err := client.ListPullReviews(owner, repo, index, gitea_sdk.ListPullReviewsOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list reviews for %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimReviews(reviews))
}

func getPullRequestReviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getPullRequestReviewFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	reviewID, err := params.GetIndex(args, "review_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	review, _, err := client.GetPullReview(owner, repo, index, reviewID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get review %v for %v/%v/pr/%v err: %v", reviewID, owner, repo, index, err))
	}

	return to.TextResult(slimReview(review))
}

func listPullRequestReviewCommentsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listPullRequestReviewCommentsFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	reviewID, err := params.GetIndex(args, "review_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	comments, _, err := client.ListPullReviewComments(owner, repo, index, reviewID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list review comments for review %v on %v/%v/pr/%v err: %v", reviewID, owner, repo, index, err))
	}

	return to.TextResult(slimReviewComments(comments))
}

func createPullRequestReviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createPullRequestReviewFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.CreatePullReviewOptions{}

	if state, ok := args["state"].(string); ok {
		opt.State = gitea_sdk.ReviewStateType(state)
	}
	if body, ok := args["body"].(string); ok {
		opt.Body = body
	}
	if commitID, ok := args["commit_id"].(string); ok {
		opt.CommitID = commitID
	}

	// Parse inline comments
	if commentsArg, exists := args["comments"]; exists {
		if commentsSlice, ok := commentsArg.([]any); ok {
			for _, comment := range commentsSlice {
				if commentMap, ok := comment.(map[string]any); ok {
					reviewComment := gitea_sdk.CreatePullReviewComment{}
					if path, ok := commentMap["path"].(string); ok {
						reviewComment.Path = path
					}
					if body, ok := commentMap["body"].(string); ok {
						reviewComment.Body = body
					}
					if oldLineNum, ok := params.ToInt64(commentMap["old_line_num"]); ok {
						reviewComment.OldLineNum = oldLineNum
					}
					if newLineNum, ok := params.ToInt64(commentMap["new_line_num"]); ok {
						reviewComment.NewLineNum = newLineNum
					}
					opt.Comments = append(opt.Comments, reviewComment)
				}
			}
		}
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	review, _, err := client.CreatePullReview(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create review for %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimReview(review))
}

func submitPullRequestReviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called submitPullRequestReviewFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	reviewID, err := params.GetIndex(args, "review_id")
	if err != nil {
		return to.ErrorResult(err)
	}
	state, err := params.GetString(args, "state")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.SubmitPullReviewOptions{
		State: gitea_sdk.ReviewStateType(state),
	}
	if body, ok := args["body"].(string); ok {
		opt.Body = body
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	review, _, err := client.SubmitPullReview(owner, repo, index, reviewID, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("submit review %v for %v/%v/pr/%v err: %v", reviewID, owner, repo, index, err))
	}

	return to.TextResult(slimReview(review))
}

func deletePullRequestReviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deletePullRequestReviewFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	reviewID, err := params.GetIndex(args, "review_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	_, err = client.DeletePullReview(owner, repo, index, reviewID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete review %v for %v/%v/pr/%v err: %v", reviewID, owner, repo, index, err))
	}

	successMsg := map[string]any{
		"message":    "Successfully deleted review",
		"review_id":  reviewID,
		"pr_index":   index,
		"repository": fmt.Sprintf("%s/%s", owner, repo),
	}

	return to.TextResult(successMsg)
}

func dismissPullRequestReviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called dismissPullRequestReviewFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}
	reviewID, err := params.GetIndex(args, "review_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.DismissPullReviewOptions{}
	if message, ok := args["message"].(string); ok {
		opt.Message = message
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	_, err = client.DismissPullReview(owner, repo, index, reviewID, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("dismiss review %v for %v/%v/pr/%v err: %v", reviewID, owner, repo, index, err))
	}

	successMsg := map[string]any{
		"message":    "Successfully dismissed review",
		"review_id":  reviewID,
		"pr_index":   index,
		"repository": fmt.Sprintf("%s/%s", owner, repo),
	}

	return to.TextResult(successMsg)
}

func mergePullRequestFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called mergePullRequestFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	mergeStyle := params.GetOptionalString(args, "merge_style", "merge")
	title, _ := args["title"].(string)
	message, _ := args["message"].(string)
	deleteBranch, _ := args["delete_branch"].(bool)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	forceMerge, _ := args["force_merge"].(bool)
	mergeWhenChecksSucceed, _ := args["merge_when_checks_succeed"].(bool)
	headCommitID, _ := args["head_commit_id"].(string)

	opt := gitea_sdk.MergePullRequestOption{
		Style:                  gitea_sdk.MergeStyle(mergeStyle),
		Title:                  title,
		Message:                message,
		DeleteBranchAfterMerge: deleteBranch,
		ForceMerge:             forceMerge,
		MergeWhenChecksSucceed: mergeWhenChecksSucceed,
		HeadCommitId:           headCommitID,
	}

	merged, resp, err := client.MergePullRequest(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("merge %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	if !merged && resp != nil && resp.StatusCode >= 400 {
		return to.ErrorResult(fmt.Errorf("merge %v/%v/pr/%v failed: HTTP %d %s", owner, repo, index, resp.StatusCode, resp.Status))
	}

	if !merged {
		return to.ErrorResult(fmt.Errorf("merge %v/%v/pr/%v returned merged=false", owner, repo, index))
	}

	successMsg := map[string]any{
		"merged":         merged,
		"pr_index":       index,
		"repository":     fmt.Sprintf("%s/%s", owner, repo),
		"merge_style":    mergeStyle,
		"branch_deleted": deleteBranch,
	}

	return to.TextResult(successMsg)
}

func editPullRequestFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called editPullRequestFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "index")
	if err != nil {
		return to.ErrorResult(err)
	}

	opt := gitea_sdk.EditPullRequestOption{}

	if title, ok := args["title"].(string); ok {
		opt.Title = title
	}
	if draft, ok := args["draft"].(bool); ok {
		if opt.Title == "" {
			// Fetch current title so the caller doesn't have to provide it
			// just to toggle draft status.
			client, err := gitea.ClientFromContext(ctx)
			if err != nil {
				return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
			}
			pr, _, err := client.GetPullRequest(owner, repo, index)
			if err != nil {
				return to.ErrorResult(fmt.Errorf("get %v/%v/pr/%v err: %v", owner, repo, index, err))
			}
			opt.Title = pr.Title
		}
		opt.Title = applyDraftPrefix(opt.Title, draft)
	}
	if body, ok := args["body"].(string); ok {
		opt.Body = new(body)
	}
	if base, ok := args["base"].(string); ok {
		opt.Base = base
	}
	if assignee, ok := args["assignee"].(string); ok {
		opt.Assignee = assignee
	}
	if assignees := params.GetStringSlice(args, "assignees"); assignees != nil {
		opt.Assignees = assignees
	}
	if val, exists := args["milestone"]; exists {
		if milestone, ok := params.ToInt64(val); ok {
			opt.Milestone = milestone
		}
	}
	if state, ok := args["state"].(string); ok {
		opt.State = new(gitea_sdk.StateType(state))
	}
	if allowMaintainerEdit, ok := args["allow_maintainer_edit"].(bool); ok {
		opt.AllowMaintainerEdit = new(allowMaintainerEdit)
	}
	if labelIDs, err := params.GetInt64Slice(args, "labels"); err == nil {
		opt.Labels = labelIDs
	}
	opt.Deadline = params.GetOptionalTime(args, "deadline")
	if removeDeadline, ok := args["remove_deadline"].(bool); ok {
		opt.RemoveDeadline = &removeDeadline
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	pr, _, err := client.EditPullRequest(owner, repo, index, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("edit %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimPullRequest(pr))
}
