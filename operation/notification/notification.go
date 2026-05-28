package notification

import (
	"context"
	"fmt"
	"time"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
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
		mcp.WithDescription("Read notifications: list (optionally scoped to a repo) or get a thread by ID."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read notifications")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("list", "get")),
		mcp.WithString("owner", mcp.Description("scope 'list' to a repo")),
		mcp.WithString("repo", mcp.Description("scope 'list' to a repo")),
		mcp.WithNumber("id", mcp.Description("thread ID (for 'get')")),
		mcp.WithString("status", mcp.Enum("unread", "read", "pinned")),
		mcp.WithString("subject_type", mcp.Enum("Issue", "Pull", "Commit", "Repository")),
		mcp.WithString("since", mcp.Description("updated after ISO 8601")),
		mcp.WithString("before", mcp.Description("updated before ISO 8601")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30)),
	)

	NotificationWriteTool = mcp.NewTool(
		NotificationWriteToolName,
		mcp.WithDescription("Mark a notification or all notifications as read."),
		mcp.WithToolAnnotation(annotation.Write("Manage notifications")),
		mcp.WithString("method", mcp.Required(), mcp.Enum("mark_read", "mark_all_read")),
		mcp.WithNumber("id", mcp.Description("thread ID (for 'mark_read')")),
		mcp.WithString("owner", mcp.Description("scope 'mark_all_read' to a repo")),
		mcp.WithString("repo", mcp.Description("scope 'mark_all_read' to a repo")),
		mcp.WithString("last_read_at", mcp.Description("ISO 8601; defaults to now")),
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
	args := req.GetArguments()
	page, pageSize := params.GetPagination(args, 30)
	opt := gitea_sdk.ListNotificationOptions{
		ListOptions: gitea_sdk.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
	}
	if status, ok := args["status"].(string); ok {
		opt.Status = []gitea_sdk.NotificationStatus{gitea_sdk.NotificationStatus(status)}
	}
	if subjectType, ok := args["subject_type"].(string); ok {
		opt.SubjectTypes = []gitea_sdk.NotificationSubjectType{gitea_sdk.NotificationSubjectType(subjectType)}
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
		threads, _, err := client.Notifications.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("list %v/%v/notifications err: %v", owner, repo, err))
		}
		return to.TextResult(slimThreads(threads))
	}

	threads, _, err := client.Notifications.List(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list notifications err: %v", err))
	}
	return to.TextResult(slimThreads(threads))
}

func getNotificationFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	thread, _, err := client.Notifications.GetByID(ctx, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get notification/%v err: %v", id, err))
	}
	return to.TextResult(slimThread(thread))
}

func markNotificationReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := params.GetIndex(req.GetArguments(), "id")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	thread, _, err := client.Notifications.MarkReadByID(ctx, id)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("mark notification/%v read err: %v", id, err))
	}
	if thread != nil {
		return to.TextResult(slimThread(thread))
	}
	return to.TextResult("Notification marked as read")
}

func markAllNotificationsReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		threads, _, err := client.Notifications.MarkReadByRepo(ctx, owner, repo, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("mark %v/%v/notifications read err: %v", owner, repo, err))
		}
		if threads != nil {
			return to.TextResult(slimThreads(threads))
		}
		return to.TextResult("All repository notifications marked as read")
	}

	threads, _, err := client.Notifications.MarkRead(ctx, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("mark all notifications read err: %v", err))
	}
	if threads != nil {
		return to.TextResult(slimThreads(threads))
	}
	return to.TextResult("All notifications marked as read")
}
