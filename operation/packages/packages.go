package packages

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New("packages")

const (
	PackageReadToolName  = "package_read"
	PackageWriteToolName = "package_write"
)

var (
	PackageReadTool = mcp.NewTool(
		PackageReadToolName,
		mcp.WithToolAnnotation(annotation.ReadOnly("Read package registry")),
		mcp.WithDescription("Read package registry: list packages (one entry per version, filter via 'q'/'type'), list versions, or get a version."),
		mcp.WithString("method", mcp.Required(), mcp.Enum("list", "list_versions", "get")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("user or org")),
		mcp.WithString("type", mcp.Description("container/npm/maven/pypi/cargo/generic; required except 'list'")),
		mcp.WithString("name", mcp.Description("slashes auto-encoded; required except 'list'")),
		mcp.WithString("version", mcp.Description("for 'get'")),
		mcp.WithString("q", mcp.Description("search query")),
		mcp.WithNumber("page", mcp.Description(params.PageDesc), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("per_page", mcp.Description(params.PaginationDesc), mcp.DefaultNumber(30), mcp.Min(1)),
	)

	PackageWriteTool = mcp.NewTool(
		PackageWriteToolName,
		mcp.WithToolAnnotation(annotation.Destructive("Delete a package version")),
		mcp.WithDescription("Delete a package version (irreversible)."),
		mcp.WithString("method", mcp.Required(), mcp.Enum("delete")),
		mcp.WithString("owner", mcp.Required(), mcp.Description("user or org")),
		mcp.WithString("type", mcp.Required(), mcp.Description("container/npm/maven/pypi/cargo/generic")),
		mcp.WithString("name", mcp.Required(), mcp.Description("slashes auto-encoded")),
		mcp.WithString("version", mcp.Required()),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{
		Tool:    PackageReadTool,
		Handler: packageReadFn,
	})
	Tool.RegisterWrite(server.ServerTool{
		Tool:    PackageWriteTool,
		Handler: packageWriteFn,
	})
}

func packageReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "list":
		return listPackagesFn(ctx, req)
	case "list_versions":
		return listPackageVersionsFn(ctx, req)
	case "get":
		return getPackageFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

func packageWriteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	method, err := params.GetString(args, "method")
	if err != nil {
		return to.ErrorResult(err)
	}
	switch method {
	case "delete":
		return deletePackageVersionFn(ctx, req)
	default:
		return to.ErrorResult(fmt.Errorf("unknown method: %s", method))
	}
}

// escapePackageName normalises a package name for use in URL paths. It
// accepts both raw names (my-repo/my-image) and pre-encoded names
// (my-repo%2Fmy-image), decoding first to avoid double-encoding. A literal
// '%' followed by two hex digits in a raw name will be folded into its
// decoded form, but package names typically do not contain '%'.
func escapePackageName(name string) string {
	if strings.Contains(name, "%") {
		if decoded, err := url.PathUnescape(name); err == nil {
			name = decoded
		}
	}
	return url.PathEscape(name)
}

func listPackagesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}

	query := url.Values{}
	if typ, ok := args["type"].(string); ok && typ != "" {
		query.Set("type", typ)
	}
	if q, ok := args["q"].(string); ok && q != "" {
		query.Set("q", q)
	}
	page, pageSize := params.GetPagination(args, 30)
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))

	var result any
	_, err = gitea.DoJSON(ctx, "GET", "packages/"+url.PathEscape(owner), query, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list packages err: %v", err))
	}

	return to.TextResult(slimPackages(result))
}

func listPackageVersionsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	typ, err := params.GetString(args, "type")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(args, "name")
	if err != nil {
		return to.ErrorResult(err)
	}

	query := url.Values{}
	page, pageSize := params.GetPagination(args, 30)
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(pageSize))

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("packages/%s/%s/%s", url.PathEscape(owner), url.PathEscape(typ), escapePackageName(name)), query, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("list package versions err: %v", err))
	}

	return to.TextResult(slimPackages(result))
}

func getPackageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	typ, err := params.GetString(args, "type")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(args, "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	version, err := params.GetString(args, "version")
	if err != nil {
		return to.ErrorResult(err)
	}

	var result any
	_, err = gitea.DoJSON(ctx, "GET", fmt.Sprintf("packages/%s/%s/%s/%s", url.PathEscape(owner), url.PathEscape(typ), escapePackageName(name), url.PathEscape(version)), nil, nil, &result)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get package err: %v", err))
	}

	return to.TextResult(slimPackage(result))
}

func deletePackageVersionFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	typ, err := params.GetString(args, "type")
	if err != nil {
		return to.ErrorResult(err)
	}
	name, err := params.GetString(args, "name")
	if err != nil {
		return to.ErrorResult(err)
	}
	version, err := params.GetString(args, "version")
	if err != nil {
		return to.ErrorResult(err)
	}

	_, err = gitea.DoJSON(ctx, "DELETE", fmt.Sprintf("packages/%s/%s/%s/%s", url.PathEscape(owner), url.PathEscape(typ), escapePackageName(name), url.PathEscape(version)), nil, nil, nil)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete package version err: %v", err))
	}

	return to.TextResult("Package version deleted successfully")
}
