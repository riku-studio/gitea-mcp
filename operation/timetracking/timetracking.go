// Package timetracking provides MCP tools for Gitea time tracking operations
package timetracking

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
	TimetrackingReadToolName  = "timetracking_read"
	TimetrackingWriteToolName = "timetracking_write"
)

var (
	TimetrackingReadTool = mcp.NewTool(
		TimetrackingReadToolName,
		mcp.WithDescription("Read time tracking data. Use method 'list_issue_times' for issue times, 'list_repo_times' for repository times, 'get_my_stopwatches' for active stopwatches, 'get_my_times' for all your tracked times."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read tracked time")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("list_issue_times", "list_repo_times", "get_my_stopwatches", "get_my_times")),
		mcp.WithString("owner", mcp.Description("repository owner (required for 'list_issue_times', 'list_repo_times')")),
		mcp.WithString("repo", mcp.Description("repository name (required for 'list_issue_times', 'list_repo_times')")),
		mcp.WithNumber("index", mcp.Description("issue index (required for 'list_issue_times')")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	TimetrackingWriteTool = mcp.NewTool(
		TimetrackingWriteToolName,
		mcp.WithDescription("Manage time tracking: stopwatches and tracked time entries."),
		mcp.WithToolAnnotation(annotation.Write("Add or manage tracked time")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("start_stopwatch", "stop_stopwatch", "delete_stopwatch", "add_time", "delete_time")),
		mcp.WithString("owner", mcp.Description("repository owner (required for all methods)")),
		mcp.WithString("repo", mcp.Description("repository name (required for all methods)")),
		mcp.WithNumber("index", mcp.Description("issue index (required for all methods)")),
		mcp.WithNumber("time", mcp.Description("time to add in seconds (required for 'add_time')")),
		mcp.WithNumber("id", mcp.Description("tracked time entry ID (required for 'delete_time')")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{Tool: TimetrackingReadTool, Handler: readFn})
	Tool.RegisterWrite(server.ServerTool{Tool: TimetrackingWriteTool, Handler: writeFn})
}

func readFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list_issue_times":
		return listTrackedTimesFn(ctx, req)
	case "list_repo_times":
		return listRepoTimesFn(ctx, req)
	case "get_my_stopwatches":
		return getMyStopwatchesFn(ctx, req)
	case "get_my_times":
		return getMyTimesFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func writeFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "start_stopwatch":
		return startStopwatchFn(ctx, req)
	case "stop_stopwatch":
		return stopStopwatchFn(ctx, req)
	case "delete_stopwatch":
		return deleteStopwatchFn(ctx, req)
	case "add_time":
		return addTrackedTimeFn(ctx, req)
	case "delete_time":
		return deleteTrackedTimeFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

// Stopwatch handler functions

func startStopwatchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called startStopwatchFn")
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
	_, err = client.StartIssueStopWatch(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("start stopwatch on %s/%s#%d err: %v", owner, repo, index, err))
	}
	return to.TextResult(fmt.Sprintf("Stopwatch started on issue %s/%s#%d", owner, repo, index))
}

func stopStopwatchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called stopStopwatchFn")
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
	_, err = client.StopIssueStopWatch(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("stop stopwatch on %s/%s#%d err: %v", owner, repo, index, err))
	}
	return to.TextResult(fmt.Sprintf("Stopwatch stopped on issue %s/%s#%d - time recorded", owner, repo, index))
}

func deleteStopwatchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deleteStopwatchFn")
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
	_, err = client.DeleteIssueStopwatch(owner, repo, index)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete stopwatch on %s/%s#%d err: %v", owner, repo, index, err))
	}
	return to.TextResult(fmt.Sprintf("Stopwatch deleted/cancelled on issue %s/%s#%d", owner, repo, index))
}

func getMyStopwatchesFn(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getMyStopwatchesFn")
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	stopwatches, _, err := client.ListMyStopwatches(gitea_sdk.ListStopwatchesOptions{})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get stopwatches err: %v", err))
	}
	if len(stopwatches) == 0 {
		return to.TextResult("No active stopwatches")
	}
	return to.TextResult(slimStopWatches(stopwatches))
}

// Tracked time handler functions

func listTrackedTimesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listTrackedTimesFn")
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
	page, pageSize := params.GetPagination(req.GetArguments(), 30)
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	times, _, err := client.ListIssueTrackedTimes(owner, repo, index, gitea_sdk.ListTrackedTimesOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list tracked times for %s/%s#%d err: %v", owner, repo, index, err))
	}
	if len(times) == 0 {
		return to.TextResult(fmt.Sprintf("No tracked times for issue %s/%s#%d", owner, repo, index))
	}
	return to.TextResult(slimTrackedTimes(times))
}

func addTrackedTimeFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called addTrackedTimeFn")
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

	timeSeconds, err := params.GetIndex(req.GetArguments(), "time")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	trackedTime, _, err := client.AddTime(owner, repo, index, gitea_sdk.AddTimeOption{
		Time: timeSeconds,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("add tracked time to %s/%s#%d err: %v", owner, repo, index, err))
	}
	return to.TextResult(slimTrackedTime(trackedTime))
}

func deleteTrackedTimeFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deleteTrackedTimeFn")
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
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.DeleteTime(owner, repo, index, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete tracked time %d from %s/%s#%d err: %v", id, owner, repo, index, err))
	}
	return to.TextResult(fmt.Sprintf("Tracked time entry %d deleted from issue %s/%s#%d", id, owner, repo, index))
}

func listRepoTimesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listRepoTimesFn")
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
	times, _, err := client.ListRepoTrackedTimes(owner, repo, gitea_sdk.ListTrackedTimesOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list repo tracked times for %s/%s err: %v", owner, repo, err))
	}
	if len(times) == 0 {
		return to.TextResult(fmt.Sprintf("No tracked times for repository %s/%s", owner, repo))
	}
	return to.TextResult(slimTrackedTimes(times))
}

func getMyTimesFn(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getMyTimesFn")
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	times, _, err := client.ListMyTrackedTimes(gitea_sdk.ListTrackedTimesOptions{})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get tracked times err: %v", err))
	}
	if len(times) == 0 {
		return to.TextResult("No tracked times found")
	}
	return to.TextResult(slimTrackedTimes(times))
}
