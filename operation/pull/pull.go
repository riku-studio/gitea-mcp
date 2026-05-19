package pull

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
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
	ListRepoPullRequestsToolName   = "list_pull_requests"
	PullRequestReadToolName        = "pull_request_read"
	PullRequestWriteToolName       = "pull_request_write"
	PullRequestReviewWriteToolName = "pull_request_review_write"
)

var (
	ListRepoPullRequestsTool = mcp.NewTool(
		ListRepoPullRequestsToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("List pull requests")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("state", mcp.Enum("open", "closed", "all"), mcp.DefaultString("all")),
		mcp.WithString("sort", mcp.Enum("oldest", "recentupdate", "leastupdate", "mostcomment", "leastcomment", "priority"), mcp.DefaultString("recentupdate")),
		mcp.WithNumber("milestone"),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	PullRequestReadTool = mcp.NewTool(
		PullRequestReadToolName,
		mcp.WithDescription("Read pull request: details, diff, changed files, head commit status, reviews."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read pull request details")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("get", "get_diff", "get_files", "get_status", "get_reviews", "get_review", "get_review_comments")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("pull_number", mcp.Required()),
		mcp.WithNumber("review_id", mcp.Description("for 'get_review'/'get_review_comments'")),
		mcp.WithBoolean("binary", mcp.Description("include binary diff")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	PullRequestWriteTool = mcp.NewTool(
		PullRequestWriteToolName,
		mcp.WithDescription("Write pull requests: create, update, close, reopen, merge, update branch from base, manage reviewers."),
		mcp.WithToolAnnotation(annotation.Write("Create, update, close, reopen, or merge pull requests")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("create", "update", "close", "reopen", "merge", "update_branch", "add_reviewers", "remove_reviewers")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("pull_number", mcp.Description("required except for 'create'")),
		mcp.WithString("title", mcp.Description("required for 'create'; optional for 'update'/'merge'")),
		mcp.WithString("body", mcp.Description("required for 'create'; optional for 'update'")),
		mcp.WithString("head", mcp.Description("head branch (required for 'create')")),
		mcp.WithString("base", mcp.Description("base branch (required for 'create')")),
		mcp.WithString("assignee", mcp.Description("for 'update'")),
		mcp.WithArray("assignees", mcp.Description("for 'update'"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithNumber("milestone", mcp.Description("for 'update'")),
		mcp.WithString("state", mcp.Description("for 'update'"), mcp.Enum("open", "closed")),
		mcp.WithBoolean("allow_maintainer_edit", mcp.Description("for 'update'")),
		mcp.WithArray("labels", mcp.Description("label IDs"), mcp.Items(map[string]any{"type": "number"})),
		mcp.WithString("deadline", mcp.Description("ISO 8601")),
		mcp.WithBoolean("remove_deadline", mcp.Description("for 'update'")),
		mcp.WithString("merge_style", mcp.Description("for 'merge'"), mcp.Enum("merge", "rebase", "rebase-merge", "squash", "fast-forward-only"), mcp.DefaultString("merge")),
		mcp.WithString("message", mcp.Description("merge commit message or dismissal reason")),
		mcp.WithBoolean("delete_branch", mcp.Description("for 'merge'")),
		mcp.WithBoolean("force_merge", mcp.Description("merge even if checks fail")),
		mcp.WithBoolean("merge_when_checks_succeed", mcp.Description("for 'merge'")),
		mcp.WithString("head_commit_id", mcp.Description("expected head SHA for conflict detection")),
		mcp.WithArray("reviewers", mcp.Description("for 'add_reviewers'/'remove_reviewers'"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("team_reviewers", mcp.Description("for 'add_reviewers'/'remove_reviewers'"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithBoolean("draft", mcp.Description("uses 'WIP: ' title prefix")),
	)

	PullRequestReviewWriteTool = mcp.NewTool(
		PullRequestReviewWriteToolName,
		mcp.WithDescription("Write PR reviews: create, submit, delete, dismiss."),
		mcp.WithToolAnnotation(annotation.Write("Submit a pull request review")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("create", "submit", "delete", "dismiss")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithNumber("pull_number", mcp.Required()),
		mcp.WithNumber("review_id", mcp.Description("required except for 'create'")),
		mcp.WithString("state", mcp.Enum("APPROVED", "REQUEST_CHANGES", "COMMENT", "PENDING")),
		mcp.WithString("body"),
		mcp.WithString("commit_id", mcp.Description("for 'create'")),
		mcp.WithString("message", mcp.Description("dismissal reason")),
		mcp.WithArray("comments", mcp.Description("inline comments (for 'create')"), mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":         map[string]any{"type": "string"},
				"body":         map[string]any{"type": "string"},
				"old_line_num": map[string]any{"type": "number", "description": "old-file line (deletions)"},
				"new_line_num": map[string]any{"type": "number", "description": "new-file line (additions)"},
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
	case "get_files":
		return getPullRequestFilesFn(ctx, req)
	case "get_status":
		return getPullRequestStatusFn(ctx, req)
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
	case "close":
		return closePullRequestFn(ctx, req)
	case "reopen":
		return reopenPullRequestFn(ctx, req)
	case "merge":
		return mergePullRequestFn(ctx, req)
	case "update_branch":
		return updatePullRequestBranchFn(ctx, req)
	case "add_reviewers":
		return createPullRequestReviewerFn(ctx, req)
	case "remove_reviewers":
		return deletePullRequestReviewerFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func closePullRequestFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "pull_number")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	state := gitea_sdk.StateClosed
	pr, _, err := client.EditPullRequest(owner, repo, index, gitea_sdk.EditPullRequestOption{
		State: &state,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("close %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimPullRequest(pr))
}

func reopenPullRequestFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(req.GetArguments(), "pull_number")
	if err != nil {
		return to.ErrorResult(err)
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	state := gitea_sdk.StateOpen
	pr, _, err := client.EditPullRequest(owner, repo, index, gitea_sdk.EditPullRequestOption{
		State: &state,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("reopen %v/%v/pr/%v err: %v", owner, repo, index, err))
	}

	return to.TextResult(slimPullRequest(pr))
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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

	// /pulls/{n} omits `assets`; PRs are issues internally, so the issue
	// assets endpoint surfaces description attachments.
	var assets []*gitea_sdk.Attachment
	assetsPath := fmt.Sprintf("repos/%s/%s/issues/%d/assets", url.PathEscape(owner), url.PathEscape(repo), index)
	if _, err := gitea.DoJSON(ctx, "GET", assetsPath, nil, nil, &assets); err != nil {
		log.Debugf("fetch %v/%v/issues/%v/assets err: %v", owner, repo, index, err)
	}

	m := slimPullRequest(pr)
	m["body"] = slim.BodyWithAttachments(pr.Body, assets)
	return to.TextResult(m)
}

func getPullRequestDiffFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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

type reviewerOp func(client *gitea_sdk.Client, owner, repo string, index int64, opt gitea_sdk.PullReviewRequestOptions) (*gitea_sdk.Response, error)

func pullRequestReviewerFn(ctx context.Context, req mcp.CallToolRequest, verb string, op reviewerOp) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
	if err != nil {
		return to.ErrorResult(err)
	}

	reviewers := params.GetStringSlice(args, "reviewers")
	teamReviewers := params.GetStringSlice(args, "team_reviewers")

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	if _, err := op(client, owner, repo, index, gitea_sdk.PullReviewRequestOptions{
		Reviewers:     reviewers,
		TeamReviewers: teamReviewers,
	}); err != nil {
		return to.ErrorResult(fmt.Errorf("%s review requests for %v/%v/pr/%v err: %v", verb, owner, repo, index, err))
	}

	return to.TextResult(map[string]any{
		"message":        fmt.Sprintf("Successfully %sd review requests", verb),
		"reviewers":      reviewers,
		"team_reviewers": teamReviewers,
		"pr_index":       index,
		"repository":     fmt.Sprintf("%s/%s", owner, repo),
	})
}

func createPullRequestReviewerFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return pullRequestReviewerFn(ctx, req, "create", (*gitea_sdk.Client).CreateReviewRequests)
}

func deletePullRequestReviewerFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return pullRequestReviewerFn(ctx, req, "delete", (*gitea_sdk.Client).DeleteReviewRequests)
}

func listPullRequestReviewsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	opt.Body = params.GetPresentStringPtr(args, "body")
	opt.AllowMaintainerEdit = params.GetOptionalBoolPtr(args, "allow_maintainer_edit")
	opt.RemoveDeadline = params.GetOptionalBoolPtr(args, "remove_deadline")
	opt.Deadline = params.GetOptionalTime(args, "deadline")
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
		s := gitea_sdk.StateType(state)
		opt.State = &s
	}
	if labelIDs, err := params.GetInt64Slice(args, "labels"); err == nil {
		opt.Labels = labelIDs
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

func updatePullRequestBranchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
	if err != nil {
		return to.ErrorResult(err)
	}

	path := fmt.Sprintf("repos/%s/%s/pulls/%d/update", url.PathEscape(owner), url.PathEscape(repo), index)
	if _, err := gitea.DoJSON(ctx, "POST", path, nil, nil, nil); err != nil {
		return to.ErrorResult(fmt.Errorf("update %v/%v/pr/%v branch err: %v", owner, repo, index, err))
	}
	return to.TextResult(map[string]any{"message": "branch updated from base"})
}

func getPullRequestFilesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(args, 30)
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	files, _, err := client.ListPullRequestFiles(owner, repo, index, gitea_sdk.ListPullRequestFilesOptions{
		ListOptions: gitea_sdk.ListOptions{Page: page, PageSize: pageSize},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/pr/%v files err: %v", owner, repo, index, err))
	}
	return to.TextResult(files)
}

func getPullRequestStatusFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	index, err := params.GetIndex(args, "pull_number")
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
	if pr.Head == nil || pr.Head.Sha == "" {
		return to.ErrorResult(fmt.Errorf("pr %v/%v/%v has no head SHA", owner, repo, index))
	}

	status, _, err := client.GetCombinedStatus(owner, repo, pr.Head.Sha)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get %v/%v/pr/%v status err: %v", owner, repo, index, err))
	}
	return to.TextResult(status)
}
