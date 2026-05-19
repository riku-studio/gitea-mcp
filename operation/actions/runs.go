package actions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ActionsRunReadToolName  = "actions_run_read"
	ActionsRunWriteToolName = "actions_run_write"
)

var (
	ActionsRunReadTool = mcp.NewTool(
		ActionsRunReadToolName,
		mcp.WithDescription("Read Actions workflows, runs, jobs, and logs."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read Actions workflow, run, and job data")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("list_workflows", "get_workflow", "list_runs", "get_run", "list_jobs", "list_run_jobs", "get_job_log_preview", "download_job_log")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("workflow_id", mcp.Description("ID or filename (for 'get_workflow')")),
		mcp.WithNumber("run_id", mcp.Description("for 'get_run'/'list_run_jobs'")),
		mcp.WithNumber("job_id", mcp.Description("for log methods")),
		mcp.WithString("status", mcp.Description("filter for 'list_runs'/'list_jobs'")),
		mcp.WithNumber("tail_lines", mcp.Description("log tail lines"), mcp.DefaultNumber(200), mcp.Min(1)),
		mcp.WithNumber("max_bytes", mcp.Description("max log bytes"), mcp.DefaultNumber(65536), mcp.Min(1024)),
		mcp.WithString("output_path", mcp.Description("for 'download_job_log'")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	ActionsRunWriteTool = mcp.NewTool(
		ActionsRunWriteToolName,
		mcp.WithDescription("Write Actions runs: dispatch, cancel, rerun."),
		mcp.WithToolAnnotation(annotation.Write("Trigger, cancel, or rerun Actions workflows")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("dispatch_workflow", "cancel_run", "rerun_run")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("workflow_id", mcp.Description("ID or filename (for 'dispatch_workflow')")),
		mcp.WithString("ref", mcp.Description("branch or tag (for 'dispatch_workflow')")),
		mcp.WithObject("inputs", mcp.Description("for 'dispatch_workflow'")),
		mcp.WithNumber("run_id", mcp.Description("for 'cancel_run'/'rerun_run'")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{Tool: ActionsRunReadTool, Handler: runReadFn})
	Tool.RegisterWrite(server.ServerTool{Tool: ActionsRunWriteTool, Handler: runWriteFn})
}

func runReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list_workflows":
		return listRepoActionWorkflowsFn(ctx, req)
	case "get_workflow":
		return getRepoActionWorkflowFn(ctx, req)
	case "list_runs":
		return listRepoActionRunsFn(ctx, req)
	case "get_run":
		return getRepoActionRunFn(ctx, req)
	case "list_jobs":
		return listRepoActionJobsFn(ctx, req)
	case "list_run_jobs":
		return listRepoActionRunJobsFn(ctx, req)
	case "get_job_log_preview":
		return getRepoActionJobLogPreviewFn(ctx, req)
	case "download_job_log":
		return downloadRepoActionJobLogFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func runWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "dispatch_workflow":
		return dispatchRepoActionWorkflowFn(ctx, req)
	case "cancel_run":
		return cancelRepoActionRunFn(ctx, req)
	case "rerun_run":
		return rerunRepoActionRunFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func doJSONWithFallback(ctx context.Context, method string, paths []string, query url.Values, body, respOut any) error {
	var lastErr error
	for _, p := range paths {
		_, err := gitea.DoJSON(ctx, method, p, query, body, respOut)
		if err == nil {
			return nil
		}
		lastErr = err
		var httpErr *gitea.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusNotFound || httpErr.StatusCode == http.StatusMethodNotAllowed) {
			continue
		}
		return err
	}
	return lastErr
}

func listRepoActionWorkflowsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/workflows", url.PathEscape(owner), url.PathEscape(repo)),
		},
		query, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list action workflows err: %v", err))
	}
	return to.TextResult(slimActionWorkflows(result))
}

func getRepoActionWorkflowFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	workflowID, err := params.GetString(req.GetArguments(), "workflow_id")
	if err != nil {
		return to.ErrorResult(err)
	}

	var result any
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/workflows/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflowID)),
		},
		nil, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get action workflow err: %v", err))
	}
	return to.TextResult(slimActionWorkflow(result))
}

func dispatchRepoActionWorkflowFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	workflowID, err := params.GetString(req.GetArguments(), "workflow_id")
	if err != nil {
		return to.ErrorResult(err)
	}
	ref, err := params.GetString(req.GetArguments(), "ref")
	if err != nil {
		return to.ErrorResult(err)
	}

	var inputs map[string]any
	if raw, exists := req.GetArguments()["inputs"]; exists {
		if m, ok := raw.(map[string]any); ok {
			inputs = m
		}
	}

	body := map[string]any{
		"ref": ref,
	}
	if inputs != nil {
		body["inputs"] = inputs
	}

	err = doJSONWithFallback(ctx, "POST",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflowID)),
			fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatch", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflowID)),
		},
		nil, body, nil,
	)
	if err != nil {
		var httpErr *gitea.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusNotFound || httpErr.StatusCode == http.StatusMethodNotAllowed) {
			return to.ErrorResult(fmt.Errorf("workflow dispatch not supported on this Gitea version (endpoint returned %d). Check https://docs.gitea.com/api/1.24/ for available Actions endpoints", httpErr.StatusCode))
		}
		return to.ErrorResult(fmt.Errorf("dispatch action workflow err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "workflow dispatched"})
}

func listRepoActionRunsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	statusFilter, _ := req.GetArguments()["status"].(string)

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))
	if statusFilter != "" {
		query.Set("status", statusFilter)
	}

	var result any
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/runs", url.PathEscape(owner), url.PathEscape(repo)),
		},
		query, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list action runs err: %v", err))
	}
	return to.TextResult(slimActionRuns(result))
}

func getRepoActionRunFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	runID, err := params.GetIndex(req.GetArguments(), "run_id")
	if err != nil || runID <= 0 {
		return to.ErrorResult(errors.New("run_id is required"))
	}

	var result any
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/runs/%d", url.PathEscape(owner), url.PathEscape(repo), runID),
		},
		nil, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get action run err: %v", err))
	}
	return to.TextResult(slimActionRun(result))
}

func cancelRepoActionRunFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	runID, err := params.GetIndex(req.GetArguments(), "run_id")
	if err != nil || runID <= 0 {
		return to.ErrorResult(errors.New("run_id is required"))
	}

	err = doJSONWithFallback(ctx, "POST",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/runs/%d/cancel", url.PathEscape(owner), url.PathEscape(repo), runID),
		},
		nil, nil, nil,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("cancel action run err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "run cancellation requested"})
}

func rerunRepoActionRunFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	runID, err := params.GetIndex(req.GetArguments(), "run_id")
	if err != nil || runID <= 0 {
		return to.ErrorResult(errors.New("run_id is required"))
	}

	err = doJSONWithFallback(ctx, "POST",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/runs/%d/rerun", url.PathEscape(owner), url.PathEscape(repo), runID),
			fmt.Sprintf("repos/%s/%s/actions/runs/%d/rerun-failed-jobs", url.PathEscape(owner), url.PathEscape(repo), runID),
		},
		nil, nil, nil,
	)
	if err != nil {
		var httpErr *gitea.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusNotFound || httpErr.StatusCode == http.StatusMethodNotAllowed) {
			return to.ErrorResult(fmt.Errorf("workflow rerun not supported on this Gitea version (endpoint returned %d). Check https://docs.gitea.com/api/1.24/ for available Actions endpoints", httpErr.StatusCode))
		}
		return to.ErrorResult(fmt.Errorf("rerun action run err: %v", err))
	}
	return to.TextResult(map[string]any{"message": "run rerun requested"})
}

func listRepoActionJobsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	statusFilter, _ := req.GetArguments()["status"].(string)

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))
	if statusFilter != "" {
		query.Set("status", statusFilter)
	}

	var result any
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/jobs", url.PathEscape(owner), url.PathEscape(repo)),
		},
		query, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list action jobs err: %v", err))
	}
	return to.TextResult(slimActionJobs(result))
}

func listRepoActionRunJobsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	runID, err := params.GetIndex(req.GetArguments(), "run_id")
	if err != nil || runID <= 0 {
		return to.ErrorResult(errors.New("run_id is required"))
	}
	page, pageSize := params.GetPagination(req.GetArguments(), 30)

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))

	var result any
	err = doJSONWithFallback(ctx, "GET",
		[]string{
			fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", url.PathEscape(owner), url.PathEscape(repo), runID),
		},
		query, nil, &result,
	)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list action run jobs err: %v", err))
	}
	return to.TextResult(slimActionJobs(result))
}

func logPaths(owner, repo string, jobID int64) []string {
	return []string{
		fmt.Sprintf("repos/%s/%s/actions/jobs/%d/logs", url.PathEscape(owner), url.PathEscape(repo), jobID),
		fmt.Sprintf("repos/%s/%s/actions/jobs/%d/log", url.PathEscape(owner), url.PathEscape(repo), jobID),
		fmt.Sprintf("repos/%s/%s/actions/tasks/%d/log", url.PathEscape(owner), url.PathEscape(repo), jobID),
		fmt.Sprintf("repos/%s/%s/actions/task/%d/log", url.PathEscape(owner), url.PathEscape(repo), jobID),
	}
}

func fetchJobLogBytes(ctx context.Context, owner, repo string, jobID int64) ([]byte, string, error) {
	var lastErr error
	for _, p := range logPaths(owner, repo, jobID) {
		b, _, err := gitea.DoBytes(ctx, "GET", p, nil, nil, "text/plain")
		if err == nil {
			return b, p, nil
		}
		lastErr = err
		var httpErr *gitea.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusNotFound || httpErr.StatusCode == http.StatusMethodNotAllowed) {
			continue
		}
		return nil, p, err
	}
	return nil, "", lastErr
}

func tailByLines(data []byte, tailLines int) []byte {
	if tailLines <= 0 || len(data) == 0 {
		return data
	}
	lines := 0
	i := len(data) - 1
	for i >= 0 {
		if data[i] == '\n' {
			lines++
			if lines > tailLines {
				return data[i+1:]
			}
		}
		i--
	}
	return data
}

func limitBytes(data []byte, maxBytes int) ([]byte, bool) {
	if maxBytes <= 0 {
		return data, false
	}
	if len(data) <= maxBytes {
		return data, false
	}
	return data[len(data)-maxBytes:], true
}

func getRepoActionJobLogPreviewFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	jobID, err := params.GetIndex(req.GetArguments(), "job_id")
	if err != nil {
		return to.ErrorResult(err)
	}
	tailLines := int(params.GetOptionalInt(req.GetArguments(), "tail_lines", 200))
	maxBytes := int(params.GetOptionalInt(req.GetArguments(), "max_bytes", 65536))
	raw, usedPath, err := fetchJobLogBytes(ctx, owner, repo, jobID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get job log err: %v", err))
	}

	tailed := tailByLines(raw, tailLines)
	limited, truncated := limitBytes(tailed, maxBytes)

	return to.TextResult(map[string]any{
		"endpoint":   usedPath,
		"job_id":     jobID,
		"bytes":      len(raw),
		"tail_lines": tailLines,
		"max_bytes":  maxBytes,
		"truncated":  truncated,
		"log":        string(limited),
	})
}

func downloadRepoActionJobLogFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(req.GetArguments(), "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(req.GetArguments(), "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	jobID, err := params.GetIndex(req.GetArguments(), "job_id")
	if err != nil {
		return to.ErrorResult(err)
	}
	outputPath, _ := req.GetArguments()["output_path"].(string)

	raw, usedPath, err := fetchJobLogBytes(ctx, owner, repo, jobID)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("download job log err: %v", err))
	}

	if outputPath == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.TempDir()
		}
		outputPath = filepath.Join(home, ".gitea-mcp", "artifacts", "actions-logs", owner, repo, fmt.Sprintf("%d.log", jobID))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return to.ErrorResult(fmt.Errorf("create output dir err: %v", err))
	}
	if err := os.WriteFile(outputPath, raw, 0o600); err != nil {
		return to.ErrorResult(fmt.Errorf("write log file err: %v", err))
	}

	return to.TextResult(map[string]any{
		"endpoint": usedPath,
		"job_id":   jobID,
		"path":     outputPath,
		"bytes":    len(raw),
	})
}
