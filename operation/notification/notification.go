package notification

import (
	"context"
	"fmt"
	"time"

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
	NotificationReadToolName  = "notification_read"
	NotificationWriteToolName = "notification_write"
)

var (
	NotificationReadTool = mcp.NewTool(
		NotificationReadToolName,
		mcp.WithDescription("Get notifications. Use method 'list' to list notifications (optionally scoped to a repo), 'get' to get a single notification thread by ID."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("list", "get")),
		mcp.WithString("owner", mcp.Description("repository owner (for 'list' to scope to a repo)")),
		mcp.WithString("repo", mcp.Description("repository name (for 'list' to scope to a repo)")),
		mcp.WithNumber("id", mcp.Description("notification thread ID (required for 'get')")),
		mcp.WithString("status", mcp.Description("filter by status (for 'list')"), mcp.Enum("unread", "read", "pinned")),
		mcp.WithString("subject_type", mcp.Description("filter by subject type (for 'list')"), mcp.Enum("Issue", "Pull", "Commit", "Repository")),
		mcp.WithString("since", mcp.Description("filter notifications updated after this ISO 8601 timestamp (for 'list')")),
		mcp.WithString("before", mcp.Description("filter notifications updated before this ISO 8601 timestamp (for 'list')")),
		mcp.WithNumber("page", mcp.Description("page number"), mcp.DefaultNumber(1)),
		mcp.WithNumber("perPage", mcp.Description("results per page (may be capped by the server's MAX_RESPONSE_ITEMS setting, default 50)"), mcp.DefaultNumber(30)),
	)

	NotificationWriteTool = mcp.NewTool(
		NotificationWriteToolName,
		mcp.WithDescription("Manage notifications. Use method 'mark_read' to mark a single notification as read, 'mark_all_read' to mark all notifications as read (optionally scoped to a repo)."),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("mark_read", "mark_all_read")),
		mcp.WithNumber("id", mcp.Description("notification thread ID (required for 'mark_read')")),
		mcp.WithString("owner", mcp.Description("repository owner (for 'mark_all_read' to scope to a repo)")),
		mcp.WithString("repo", mcp.Description("repository name (for 'mark_all_read' to scope to a repo)")),
		mcp.WithString("last_read_at", mcp.Description("ISO 8601 timestamp, marks notifications before this time as read (for 'mark_all_read', defaults to now)")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    NotificationReadTool,
		Handler: notificationReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    NotificationWriteTool,
		Handler: notificationWriteFn,
	})
}

func notificationReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list":
		return listNotificationsFn(ctx, req)
	case "get":
		return getNotificationFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func notificationWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "mark_read":
		return markNotificationReadFn(ctx, req)
	case "mark_all_read":
		return markAllNotificationsReadFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func listNotificationsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listNotificationsFn")
	args := req.GetArguments()
	page, pageSize := params.GetPagination(args, 30)
	opt := gitea_sdk.ListNotificationOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if status, ok := args["status"].(string); ok {
		opt.Status = []gitea_sdk.NotifyStatus{gitea_sdk.NotifyStatus(status)}
	}
	if subjectType, ok := args["subject_type"].(string); ok {
		opt.SubjectTypes = []gitea_sdk.NotifySubjectType{gitea_sdk.NotifySubjectType(subjectType)}
	}
	if t := params.GetOptionalTime(args, "since"); t != nil {
		opt.Since = *t
	}
	if t := params.GetOptionalTime(args, "before"); t != nil {
		opt.Before = *t
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	owner := params.GetOptionalString(args, "owner", "")
	repo := params.GetOptionalString(args, "repo", "")
	if owner != "" && repo != "" {
		threads, _, err := client.ListRepoNotifications(owner, repo, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("list %v/%v/notifications err: %v", owner, repo, err))
		}
		return to.TextResult(slimThreads(threads))
	}

	threads, _, err := client.ListNotifications(opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list notifications err: %v", err))
	}
	return to.TextResult(slimThreads(threads))
}

func getNotificationFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getNotificationFn")
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	thread, _, err := client.GetNotification(id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get notification/%v err: %v", id, err))
	}
	return to.TextResult(slimThread(thread))
}

func markNotificationReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called markNotificationReadFn")
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	thread, _, err := client.ReadNotification(id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("mark notification/%v read err: %v", id, err))
	}
	if thread != nil {
		return to.TextResult(slimThread(thread))
	}
	return to.TextResult("Notification marked as read")
}

func markAllNotificationsReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called markAllNotificationsReadFn")
	args := req.GetArguments()
	lastReadAt := time.Now()
	if t := params.GetOptionalTime(args, "last_read_at"); t != nil {
		lastReadAt = *t
	}
	opt := gitea_sdk.MarkNotificationOptions{
		LastReadAt: lastReadAt,
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	owner := params.GetOptionalString(args, "owner", "")
	repo := params.GetOptionalString(args, "repo", "")
	if owner != "" && repo != "" {
		threads, _, err := client.ReadRepoNotifications(owner, repo, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("mark %v/%v/notifications read err: %v", owner, repo, err))
		}
		if threads != nil {
			return to.TextResult(slimThreads(threads))
		}
		return to.TextResult("All repository notifications marked as read")
	}

	threads, _, err := client.ReadNotifications(opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("mark all notifications read err: %v", err))
	}
	if threads != nil {
		return to.TextResult(slimThreads(threads))
	}
	return to.TextResult("All notifications marked as read")
}
