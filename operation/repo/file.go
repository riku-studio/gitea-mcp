package repo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/gitea"
	"gitea.com/gitea/gitea-mcp/pkg/params"
	"gitea.com/gitea/gitea-mcp/pkg/to"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	gitea_sdk "gitea.dev/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	GetFileContentTool = mcp.NewTool(
		GetFileToolName,
		mcp.WithDescription("Get file content and metadata"),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get file content")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("ref", mcp.Required(), mcp.Description("branch, tag, or commit SHA")),
		mcp.WithString("path", mcp.Required()),
		mcp.WithBoolean("withLines", mcp.Description("return numbered lines")),
	)

	GetDirContentTool = mcp.NewTool(
		GetDirToolName,
		mcp.WithDescription("List the entries (files and subdirectories) in a repository directory at a given ref (branch, tag, or commit SHA)."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Get directory contents")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("ref", mcp.Required(), mcp.Description("branch, tag, or commit SHA")),
		mcp.WithString("path", mcp.Required()),
	)

	CreateOrUpdateFileTool = mcp.NewTool(
		CreateOrUpdateFileToolName,
		mcp.WithDescription("Create or update a file (provide sha to update an existing file)."),
		mcp.WithToolAnnotation(annotation.Write("Create or update a file")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("path", mcp.Required()),
		mcp.WithString("content", mcp.Required()),
		mcp.WithString("message", mcp.Required(), mcp.Description("commit message")),
		mcp.WithString("branch_name", mcp.Required()),
		mcp.WithString("sha", mcp.Description("existing file SHA (omit to create)")),
		mcp.WithString("new_branch_name", mcp.Description("new branch (create only)")),
	)

	DeleteFileTool = mcp.NewTool(
		DeleteFileToolName,
		mcp.WithDescription("Delete a file from a repository by committing the removal to a branch. Requires the file's current SHA and a commit message."),
		mcp.WithToolAnnotation(annotation.Destructive("Delete a file")),
		mcp.WithString("owner", mcp.Required(), mcp.Description(params.OwnerDesc)),
		mcp.WithString("repo", mcp.Required(), mcp.Description(params.RepoDesc)),
		mcp.WithString("path", mcp.Required()),
		mcp.WithString("message", mcp.Required(), mcp.Description("commit message")),
		mcp.WithString("branch_name", mcp.Required()),
		mcp.WithString("sha", mcp.Required()),
	)
)

func init() {
	FileTool.RegisterRead(server.ServerTool{
		Tool:    GetFileContentTool,
		Handler: GetFileContentFn,
	})
	FileTool.RegisterRead(server.ServerTool{
		Tool:    GetDirContentTool,
		Handler: GetDirContentFn,
	})
	FileTool.RegisterWrite(server.ServerTool{
		Tool:    CreateOrUpdateFileTool,
		Handler: CreateOrUpdateFileFn,
	})
	FileTool.RegisterWrite(server.ServerTool{
		Tool:    DeleteFileTool,
		Handler: DeleteFileFn,
	})
}

type ContentLine struct {
	LineNumber int    `json:"line"`
	Content    string `json:"content"`
}

func GetFileContentFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
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
	if withLines {
		rawContent, err := base64.StdEncoding.DecodeString(*content.Content)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("decode base64 content err: %v", err))
		}

		contentLines := make([]ContentLine, 0)
		line := 0

		scanner := bufio.NewScanner(bytes.NewReader(rawContent))

		for scanner.Scan() {
			line++

			contentLines = append(contentLines, ContentLine{
				LineNumber: line,
				Content:    scanner.Text(),
			})
		}
		if err := scanner.Err(); err != nil {
			return to.ErrorResult(fmt.Errorf("scan content err: %v", err))
		}

		// remove the last blank line if exists
		// git does not consider the last line as a new line
		if len(contentLines) > 0 && contentLines[len(contentLines)-1].Content == "" {
			contentLines = contentLines[:len(contentLines)-1]
		}

		contentBytes, err := json.MarshalIndent(contentLines, "", "  ")
		if err != nil {
			return to.ErrorResult(fmt.Errorf("marshal content lines err: %v", err))
		}
		contentStr := string(contentBytes)
		content.Content = &contentStr
	}
	return to.TextResult(slimContents(content))
}

func GetDirContentFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
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

func CreateOrUpdateFileFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
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
	content, _ := args["content"].(string)
	message, _ := args["message"].(string)
	branchName, _ := args["branch_name"].(string)
	sha, _ := args["sha"].(string)

	client, err := gitea.ClientFromContext(ctx)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("get gitea client err: %v", err))
	}

	if sha != "" {
		// Update existing file
		opt := gitea_sdk.UpdateFileOptions{
			SHA:     sha,
			Content: base64.StdEncoding.EncodeToString([]byte(content)),
			FileOptions: gitea_sdk.FileOptions{
				Message:    message,
				BranchName: branchName,
			},
		}
		_, _, err = client.Repositories.UpdateFile(ctx, owner, repo, filePath, opt)
		if err != nil {
			return to.ErrorResult(fmt.Errorf("update file err: %v", err))
		}
		return to.TextResult("Update file success")
	}

	// Create new file
	opt := gitea_sdk.CreateFileOptions{
		Content: base64.StdEncoding.EncodeToString([]byte(content)),
		FileOptions: gitea_sdk.FileOptions{
			Message:    message,
			BranchName: branchName,
		},
	}
	if newBranch, ok := args["new_branch_name"].(string); ok && newBranch != "" {
		opt.NewBranchName = newBranch
	}
	_, _, err = client.Repositories.CreateFile(ctx, owner, repo, filePath, opt)
	if err != nil {
		return to.ErrorResult(fmt.Errorf("create file err: %v", err))
	}
	return to.TextResult("Create file success")
}

func DeleteFileFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
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
		FileOptions: gitea_sdk.FileOptions{
			Message:    message,
			BranchName: branchName,
		},
		SHA: sha,
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
