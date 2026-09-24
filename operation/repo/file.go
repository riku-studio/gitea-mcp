package repo

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FileTool holds the file-related tools (scope "file").
var FileTool = tool.New("file")

const (
	GetFileToolName            = "get_file_contents"
	GetDirToolName             = "get_dir_contents"
	CreateOrUpdateFileToolName = "create_or_update_file"
	DeleteFileToolName         = "delete_file"
)

var (
	fileEditSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"old_string":  map[string]any{"type": "string"},
			"new_string":  map[string]any{"type": "string"},
			"replace_all": map[string]any{"type": "boolean"},
		},
		"required":             []string{"old_string", "new_string"},
		"additionalProperties": false,
	}

	GetFileContentTool = tool.NewDefinition(
		GetFileToolName,
		"Get file content and metadata",
		annotation.ReadOnly("Get file content"),
		tool.String("owner", tool.Required(), tool.Description(params.OwnerDesc)),
		tool.String("repo", tool.Required(), tool.Description(params.RepoDesc)),
		tool.String("ref", tool.Required(), tool.Description("branch, tag, or commit SHA")),
		tool.String("path", tool.Required(), tool.Description(params.PathDesc)),
		tool.Boolean("withLines", tool.Description("return numbered lines")),
		tool.Number("start_line", tool.Minimum(1)),
		tool.Number("end_line", tool.Minimum(1)),
	)

	GetDirContentTool = tool.NewDefinition(
		GetDirToolName,
		"List the entries (files and subdirectories) in a repository directory at a given ref (branch, tag, or commit SHA).",
		annotation.ReadOnly("Get directory contents"),
		tool.String("owner", tool.Required(), tool.Description(params.OwnerDesc)),
		tool.String("repo", tool.Required(), tool.Description(params.RepoDesc)),
		tool.String("ref", tool.Required(), tool.Description("branch, tag, or commit SHA")),
		tool.String("path", tool.Required(), tool.Description(params.PathDesc)),
	)

	CreateOrUpdateFileTool = tool.NewDefinition(
		CreateOrUpdateFileToolName,
		"Write files in one commit: create, update, rename, delete.",
		annotation.Write("Create, update, rename, or delete files"),
		tool.String("owner", tool.Required(), tool.Description(params.OwnerDesc)),
		tool.String("repo", tool.Required(), tool.Description(params.RepoDesc)),
		tool.String("path", tool.Description(params.PathDesc)),
		tool.String("content", tool.Description(params.FileContentDesc)),
		tool.String("message", tool.Required(), tool.Description("commit message")),
		tool.String("branch_name", tool.Required(), tool.Description("branch to commit to")),
		tool.String("sha", tool.Description("existing file SHA (omit to create)")),
		tool.String("new_branch_name", tool.Description("branch to create from branch_name and commit to")),
		tool.Array("edits", tool.Items(fileEditSchema)),
		tool.Array("files", tool.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":      map[string]any{"type": "string", "description": params.PathDesc},
				"content":   map[string]any{"type": "string", "description": params.FileContentDesc},
				"edits":     map[string]any{"type": "array", "items": fileEditSchema},
				"sha":       map[string]any{"type": "string", "description": "existing file SHA, of from_path when renaming (omit to create)"},
				"from_path": map[string]any{"type": "string", "description": "rename source"},
				"delete":    map[string]any{"type": "boolean"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		})),
	)

	DeleteFileTool = tool.NewDefinition(
		DeleteFileToolName,
		"Delete a file from a repository by committing the removal to a branch. Requires the file's current SHA and a commit message.",
		annotation.Destructive("Delete a file"),
		tool.String("owner", tool.Required(), tool.Description(params.OwnerDesc)),
		tool.String("repo", tool.Required(), tool.Description(params.RepoDesc)),
		tool.String("path", tool.Required(), tool.Description(params.PathDesc)),
		tool.String("message", tool.Required(), tool.Description("commit message")),
		tool.String("branch_name", tool.Required(), tool.Description("branch to commit to")),
		tool.String("sha", tool.Required(), tool.Description("existing file SHA")),
	)
)

func init() {
	FileTool.RegisterRead(tool.ServerTool{
		Tool:    GetFileContentTool,
		Handler: GetFileContentFn,
	})
	FileTool.RegisterRead(tool.ServerTool{
		Tool:    GetDirContentTool,
		Handler: GetDirContentFn,
	})
	FileTool.RegisterWrite(tool.ServerTool{
		Tool:    CreateOrUpdateFileTool,
		Handler: CreateOrUpdateFileFn,
	})
	FileTool.RegisterWrite(tool.ServerTool{
		Tool:    DeleteFileTool,
		Handler: DeleteFileFn,
	})
}

type ContentLine struct {
	LineNumber int    `json:"line"`
	Content    string `json:"content"`
}

type lineSelection struct {
	Bytes              []byte
	First, Last, Total int
}

// selectLines returns lines start..end (1-based, 0 means unbounded) as a sub-slice of raw, counting lines like git.
func selectLines(raw []byte, start, end int) (lineSelection, error) {
	total := bytes.Count(raw, []byte{'\n'})
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		total++
	}
	if total == 0 {
		return lineSelection{First: 1}, nil
	}
	first, last := max(start, 1), total
	if end > 0 {
		last = min(end, total)
	}
	if first > total {
		return lineSelection{}, fmt.Errorf("start_line %d is past the end of the file (%d lines)", first, total)
	}
	if last < first {
		return lineSelection{}, fmt.Errorf("end_line %d is before start_line %d", last, first)
	}

	begin := 0
	for range first - 1 {
		begin += bytes.IndexByte(raw[begin:], '\n') + 1
	}
	stop := begin
	for range last - first + 1 {
		newline := bytes.IndexByte(raw[stop:], '\n')
		if newline < 0 {
			stop = len(raw)
			break
		}
		stop += newline + 1
	}
	selected := raw[begin:stop]
	if trimmed, ok := bytes.CutSuffix(selected, []byte("\n")); ok {
		selected = bytes.TrimSuffix(trimmed, []byte("\r"))
	}
	return lineSelection{Bytes: selected, First: first, Last: last, Total: total}, nil
}

func GetFileContentFn(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	ref, _ := args["ref"].(string)
	filePath, err := params.GetString(args, "path")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	content, _, err := client.Repositories.GetContents(ctx, owner, repo, ref, filePath)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get file err: %v", err))
	}

	withLines, _ := args["withLines"].(bool)
	startLine := int(params.GetOptionalInt(args, "start_line", 0))
	endLine := int(params.GetOptionalInt(args, "end_line", 0))
	rangeRequested := startLine > 0 || endLine > 0
	if !withLines && !rangeRequested {
		return to.TextResult(slimContents(content))
	}

	raw, err := decodeContent(content)
	if err != nil {
		return to.ErrorResult(err)
	}
	selection, err := selectLines(raw, startLine, endLine)
	if err != nil {
		return to.ErrorResult(err)
	}

	text, encoding := string(selection.Bytes), "utf-8"
	if !utf8.Valid(selection.Bytes) {
		text, encoding = base64.StdEncoding.EncodeToString(selection.Bytes), "base64"
	}
	if withLines && encoding == "utf-8" {
		contentLines := make([]ContentLine, 0, selection.Last-selection.First+1)
		if selection.Total > 0 {
			for line := range strings.SplitSeq(text, "\n") {
				contentLines = append(contentLines, ContentLine{LineNumber: selection.First + len(contentLines), Content: strings.TrimSuffix(line, "\r")})
			}
		}
		contentBytes, err := json.MarshalIndent(contentLines, "", "  ")
		if err != nil {
			return to.ErrorResult(fmt.Errorf("marshal content lines err: %v", err))
		}
		text = string(contentBytes)
	}
	content.Content = &text
	content.Encoding = &encoding

	result := slimContents(content)
	result["total_lines"] = selection.Total
	if rangeRequested {
		result["start_line"] = selection.First
		result["end_line"] = selection.Last
		result["truncated"] = selection.First > 1 || selection.Last < selection.Total
	}
	return to.TextResult(result)
}

func decodeContent(content *gitea_sdk.ContentsResponse) ([]byte, error) {
	if content.Content == nil {
		return nil, fmt.Errorf("%s has no inline content, it may be a directory, submodule or too large", content.Path)
	}
	raw, err := base64.StdEncoding.DecodeString(*content.Content)
	if err != nil {
		return nil, fmt.Errorf("decode base64 content err: %v", err)
	}
	return raw, nil
}

func GetDirContentFn(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	ref, _ := args["ref"].(string)
	filePath, err := params.GetString(args, "path")
	if err != nil {
		return to.ErrorResult(err)
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	content, _, err := client.Repositories.ListContents(ctx, owner, repo, ref, filePath)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get dir content err: %v", err))
	}
	return to.TextResult(slimDirEntries(content))
}

func CreateOrUpdateFileFn(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	message, _ := args["message"].(string)
	branchName, _ := args["branch_name"].(string)
	newBranchName, _ := args["new_branch_name"].(string)

	files, _ := args["files"].([]any)
	if params.GetOptionalString(args, "path", "") != "" {
		files = append([]any{args}, files...)
	}
	if len(files) == 0 {
		return to.ErrorResult(errors.New("path or files is required"))
	}

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	operations := make([]*gitea_sdk.ChangeFileOperation, 0, len(files))
	touched := make(map[string]bool, len(files))
	for _, file := range files {
		fileArgs, _ := file.(map[string]any)
		operation, err := changeFileOperation(ctx, client, owner, repo, branchName, fileArgs)
		if err != nil {
			return to.ErrorResult(err)
		}
		for _, touchedPath := range []string{operation.Path, operation.FromPath} {
			if touched[touchedPath] {
				return to.ErrorResult(fmt.Errorf("%s: appears in more than one file entry", touchedPath))
			}
			touched[touchedPath] = touchedPath != ""
		}
		operations = append(operations, operation)
	}

	_, _, err = client.Repositories.ChangeFiles(ctx, owner, repo, gitea_sdk.ChangeFilesOptions{
		Files:     operations,
		Message:   message,
		Branch:    branchName,
		NewBranch: newBranchName,
	})
	if err != nil {
		return to.ErrorResult(fmt.Errorf("change files err: %v", err))
	}
	return to.TextResult(fmt.Sprintf("Committed %d file(s) to branch %s", len(operations), cmp.Or(newBranchName, branchName)))
}

func changeFileOperation(ctx context.Context, client *gitea_sdk.Client, owner, repo, ref string, file map[string]any) (*gitea_sdk.ChangeFileOperation, error) {
	filePath, err := params.GetString(file, "path")
	if err != nil {
		return nil, err
	}
	content, hasContent := file["content"].(string)
	sha, _ := file["sha"].(string)
	fromPath, _ := file["from_path"].(string)
	edits, _ := file["edits"].([]any)
	operation := "update"
	switch deleteFile, _ := file["delete"].(bool); {
	case deleteFile:
		if hasContent || len(edits) > 0 || fromPath != "" {
			return nil, fmt.Errorf("%s: delete excludes content, edits and from_path", filePath)
		}
		operation = "delete"
	case len(edits) > 0:
		if hasContent {
			return nil, fmt.Errorf("%s: content and edits are mutually exclusive", filePath)
		}
		sourcePath := cmp.Or(fromPath, filePath)
		current, _, err := client.Repositories.GetContents(ctx, owner, repo, ref, sourcePath)
		if err != nil {
			return nil, fmt.Errorf("get %s err: %v", sourcePath, err)
		}
		raw, err := decodeContent(current)
		if err != nil {
			return nil, err
		}
		if content, err = applyEdits(string(raw), edits); err != nil {
			return nil, fmt.Errorf("%s: %w", filePath, err)
		}
		sha = cmp.Or(sha, current.SHA)
	case hasContent:
		if sha == "" && fromPath == "" {
			operation = "create"
		}
	case fromPath != "":
		operation = "rename"
	default:
		return nil, fmt.Errorf("%s: content, edits, from_path or delete is required", filePath)
	}
	if sha == "" && operation != "create" { // Gitea skips its SHA check when none is given
		return nil, fmt.Errorf("%s: sha is required", filePath)
	}

	return &gitea_sdk.ChangeFileOperation{
		Operation: operation,
		Path:      filePath,
		Content:   base64.StdEncoding.EncodeToString([]byte(content)),
		SHA:       sha,
		FromPath:  fromPath,
	}, nil
}

func applyEdits(text string, edits []any) (string, error) {
	for i, raw := range edits {
		edit, _ := raw.(map[string]any)
		oldString, _ := edit["old_string"].(string)
		newString, hasNewString := edit["new_string"].(string)
		replaceAll, _ := edit["replace_all"].(bool)
		if oldString == "" || !hasNewString {
			return "", fmt.Errorf("edit %d: old_string and new_string are required", i+1)
		}
		switch count := strings.Count(text, oldString); {
		case count == 0:
			return "", fmt.Errorf("edit %d: old_string not found, it must match exactly", i+1)
		case count > 1 && !replaceAll:
			return "", fmt.Errorf("edit %d: old_string occurs %d times, add context or set replace_all", i+1, count)
		}
		text = strings.ReplaceAll(text, oldString, newString)
	}
	return text, nil
}

func DeleteFileFn(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	owner, err := params.GetString(args, "owner")
	if err != nil {
		return to.ErrorResult(err)
	}
	repo, err := params.GetString(args, "repo")
	if err != nil {
		return to.ErrorResult(err)
	}
	filePath, err := params.GetString(args, "path")
	if err != nil {
		return to.ErrorResult(err)
	}
	message, _ := args["message"].(string)
	branchName, _ := args["branch_name"].(string)
	sha, err := params.GetString(args, "sha")
	if err != nil {
		return to.ErrorResult(err)
	}
	opt := gitea_sdk.DeleteFileOptions{
		Message:    message,
		BranchName: branchName,
		SHA:        sha,
	}
	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}
	_, err = client.Repositories.DeleteFile(ctx, owner, repo, filePath, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("delete file err: %v", err))
	}
	return to.TextResult("Delete file success")
}
