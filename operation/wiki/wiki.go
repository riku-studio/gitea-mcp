package wiki

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/log"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	WikiReadToolName  = "wiki_read"
	WikiWriteToolName = "wiki_write"
)

var (
	WikiReadTool = mcp.NewTool(
		WikiReadToolName,
		mcp.WithDescription("Read wiki page information. Use method 'list' to list pages, 'get' to get page content, 'get_revisions' for revision history."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Read wiki pages")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("list", "get", "get_revisions")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("pageName", mcp.Description("wiki page name (required for 'get', 'get_revisions')")),
	)

	WikiWriteTool = mcp.NewTool(
		WikiWriteToolName,
		mcp.WithDescription("Create, update, or delete wiki pages."),
		mcp.WithToolAnnotation(annotation.Destructive("Create, update, or delete wiki pages")),
		mcp.WithString("method", mcp.Required(), mcp.Description("operation to perform"), mcp.Enum("create", "update", "delete")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("repository owner")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("repository name")),
		mcp.WithString("pageName", mcp.Description("wiki page name (required for 'update', 'delete')")),
		mcp.WithString("title", mcp.Description("wiki page title (required for 'create', optional for 'update')")),
		mcp.WithString("content", mcp.Description("page content (required for 'create', 'update')")),
		mcp.WithString("message", mcp.Description("commit message")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    WikiReadTool,
		Handler: wikiReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    WikiWriteTool,
		Handler: wikiWriteFn,
	})
}

func wikiReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list":
		return listWikiPagesFn(ctx, req)
	case "get":
		return getWikiPageFn(ctx, req)
	case "get_revisions":
		return getWikiRevisionsFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func wikiWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	method, err := params.GetString(req.GetArguments(), "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "create":
		return createWikiPageFn(ctx, req)
	case "update":
		return updateWikiPageFn(ctx, req)
	case "delete":
		return deleteWikiPageFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func listWikiPagesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called listWikiPagesFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("repos/%s/%s/wiki/pages", url.PathEscape(owner), url.PathEscape(repo)), nil, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list wiki pages err: %v", err))
	}

	return to.TextResult(result)
}

func getWikiPageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getWikiPageFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	pageName, err := params.GetString(args, "pageName")
	if err != nil {
		return to.ErrorResult(err)
	}

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("repos/%s/%s/wiki/page/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(pageName)), nil, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get wiki page err: %v", err))
	}

	return to.TextResult(result)
}

func getWikiRevisionsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called getWikiRevisionsFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	pageName, err := params.GetString(args, "pageName")
	if err != nil {
		return to.ErrorResult(err)
	}

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("repos/%s/%s/wiki/revisions/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(pageName)), nil, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get wiki revisions err: %v", err))
	}

	return to.TextResult(result)
}

func createWikiPageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called createWikiPageFn")
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
	content, err := params.GetString(args, "content")
	if err != nil {
		return to.ErrorResult(err)
	}

	message, _ := args["message"].(string)
	if message == "" {
		message = fmt.Sprintf("Create wiki page '%s'", title)
	}

	requestBody := map[string]string{
		"title":          title,
		"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
		"message":        message,
	}

	var result any
	_, err = gitea.DoJSON(ctx, "POST", fmt.Sprintf("repos/%s/%s/wiki/new", url.PathEscape(owner), url.PathEscape(repo)), nil, requestBody, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create wiki page err: %v", err))
	}

	return to.TextResult(result)
}

func updateWikiPageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called updateWikiPageFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	pageName, err := params.GetString(args, "pageName")
	if err != nil {
		return to.ErrorResult(err)
	}
	content, err := params.GetString(args, "content")
	if err != nil {
		return to.ErrorResult(err)
	}

	requestBody := map[string]string{
		"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
	}

	// If title is given, use it. Otherwise, keep current page name
	if title, ok := args["title"].(string); ok && title != "" {
		requestBody["title"] = title
	} else {
		requestBody["title"] = pageName
	}

	if message, ok := args["message"].(string); ok && message != "" {
		requestBody["message"] = message
	} else {
		requestBody["message"] = fmt.Sprintf("Update wiki page '%s'", pageName)
	}

	var result any
	_, err = gitea.DoJSON(ctx, "PATCH", fmt.Sprintf("repos/%s/%s/wiki/page/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(pageName)), nil, requestBody, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("update wiki page err: %v", err))
	}

	return to.TextResult(result)
}

func deleteWikiPageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("Called deleteWikiPageFn")
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	pageName, err := params.GetString(args, "pageName")
	if err != nil {
		return to.ErrorResult(err)
	}

	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("repos/%s/%s/wiki/page/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(pageName)), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete wiki page err: %v", err))
	}

	return to.TextResult(map[string]string{"message": "Wiki page deleted successfully"})
}
