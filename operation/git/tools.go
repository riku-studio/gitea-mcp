package git

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitea.com/gitea/gitea-mcp/pkg/annotation"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	StatusToolName   = "git_status"
	DiffToolName     = "git_diff"
	AddToolName      = "git_add"
	CommitToolName   = "git_commit"
	PushToolName     = "git_push"
	PullToolName     = "git_pull"
	FetchToolName    = "git_fetch"
	BranchToolName   = "git_branch"
	CheckoutToolName = "git_checkout"
	LogToolName      = "git_log"

	repoPathDesc = "absolute path of the local git working copy (must be inside " + AllowedRootsEnv + ")"
)

func repoPathParam() mcp.ToolOption {
	return mcp.WithString("repo_path", mcp.Required(), mcp.Description(repoPathDesc))
}

func stringArray(name, desc string) mcp.ToolOption {
	return mcp.WithArray(name, mcp.Description(desc), mcp.Items(map[string]any{"type": "string"}))
}

var (
	StatusTool = mcp.NewTool(StatusToolName,
		mcp.WithDescription("Show the working tree status of a local repository (branch, ahead/behind, staged/unstaged/untracked files)."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Local git status")),
		repoPathParam(),
	)

	DiffTool = mcp.NewTool(DiffToolName,
		mcp.WithDescription("Show changes in a local repository. Default: unstaged changes. Use staged=true for what will be committed, or target to compare against a commit/branch."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Local git diff")),
		repoPathParam(),
		mcp.WithBoolean("staged", mcp.Description("diff the index (staged changes) instead of the working tree")),
		mcp.WithString("target", mcp.Description("commit, branch or range to diff against (e.g. HEAD~1, main, main...feature)")),
		stringArray("paths", "limit the diff to these paths (relative to repo root)"),
		mcp.WithBoolean("stat", mcp.Description("only show a diffstat summary")),
		mcp.WithBoolean("name_only", mcp.Description("only list changed file names")),
		mcp.WithNumber("context_lines", mcp.Description("lines of context (default 3)")),
	)

	AddTool = mcp.NewTool(AddToolName,
		mcp.WithDescription("Stage files for the next commit. Give paths, or all=true to stage every change including deletions and untracked files."),
		mcp.WithToolAnnotation(annotation.Write("Local git add")),
		repoPathParam(),
		stringArray("paths", "paths to stage (relative to repo root; directories and globs allowed)"),
		mcp.WithBoolean("all", mcp.Description("stage all changes (git add -A)")),
	)

	CommitTool = mcp.NewTool(CommitToolName,
		mcp.WithDescription("Create a commit from the staged changes in a local repository. Returns the new commit hash and stat."),
		mcp.WithToolAnnotation(annotation.Write("Local git commit")),
		repoPathParam(),
		mcp.WithString("message", mcp.Required(), mcp.Description("full commit message (subject, blank line, body, trailers)")),
		mcp.WithBoolean("all", mcp.Description("also stage modified/deleted tracked files first (git commit -a); untracked files are not added")),
		mcp.WithString("author_name", mcp.Description("override author name (default: git config user.name)")),
		mcp.WithString("author_email", mcp.Description("override author email (default: git config user.email)")),
	)

	PushTool = mcp.NewTool(PushToolName,
		mcp.WithDescription("Push a local branch to a remote. Force push is not supported. If the remote is https on the configured Gitea host, the server's Gitea token is used automatically."),
		mcp.WithToolAnnotation(annotation.Write("Local git push")),
		repoPathParam(),
		mcp.WithString("remote", mcp.Description("remote name (default: origin)")),
		mcp.WithString("branch", mcp.Description("local branch to push (default: current branch)")),
		mcp.WithString("remote_branch", mcp.Description("destination branch name on the remote (default: same as branch)")),
		mcp.WithBoolean("set_upstream", mcp.Description("set the pushed branch as upstream (-u)")),
		mcp.WithBoolean("tags", mcp.Description("also push tags reachable from the pushed commits (--follow-tags)")),
	)

	PullTool = mcp.NewTool(PullToolName,
		mcp.WithDescription("Fetch from a remote and integrate into the current branch. Fast-forward only by default; set rebase=true to rebase local commits instead. Never creates merge commits."),
		mcp.WithToolAnnotation(annotation.Write("Local git pull")),
		repoPathParam(),
		mcp.WithString("remote", mcp.Description("remote name (default: origin)")),
		mcp.WithString("branch", mcp.Description("remote branch (default: the current branch's upstream)")),
		mcp.WithBoolean("rebase", mcp.Description("rebase local commits onto the remote branch instead of fast-forward only")),
	)

	FetchTool = mcp.NewTool(FetchToolName,
		mcp.WithDescription("Download objects and refs from a remote without changing the working tree."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Local git fetch")),
		repoPathParam(),
		mcp.WithString("remote", mcp.Description("remote name (default: origin)")),
		mcp.WithBoolean("prune", mcp.Description("remove remote-tracking refs that no longer exist on the remote")),
		mcp.WithBoolean("tags", mcp.Description("also fetch all tags")),
	)

	BranchTool = mcp.NewTool(BranchToolName,
		mcp.WithDescription("List, create or delete local branches. Delete refuses branches that are not fully merged."),
		mcp.WithToolAnnotation(annotation.Write("Local git branch")),
		repoPathParam(),
		mcp.WithString("action", mcp.Description("list (default) | create | delete"), mcp.Enum("list", "create", "delete")),
		mcp.WithString("name", mcp.Description("branch name, for create/delete")),
		mcp.WithString("start_point", mcp.Description("commit/branch to start the new branch from (default: HEAD), for create")),
		mcp.WithBoolean("remotes", mcp.Description("for list: include remote-tracking branches")),
	)

	CheckoutTool = mcp.NewTool(CheckoutToolName,
		mcp.WithDescription("Switch to a branch (git switch). With create=true, create it first. Refuses to switch if it would overwrite uncommitted changes. Does not discard file changes."),
		mcp.WithToolAnnotation(annotation.Write("Local git checkout")),
		repoPathParam(),
		mcp.WithString("branch", mcp.Required(), mcp.Description("branch to switch to")),
		mcp.WithBoolean("create", mcp.Description("create the branch (git switch -c)")),
		mcp.WithString("start_point", mcp.Description("with create: commit/branch to start from (default: HEAD)")),
	)

	LogTool = mcp.NewTool(LogToolName,
		mcp.WithDescription("Show commit history of a local repository."),
		mcp.WithToolAnnotation(annotation.ReadOnly("Local git log")),
		repoPathParam(),
		mcp.WithString("ref", mcp.Description("branch, commit or range (default: HEAD)")),
		mcp.WithNumber("max_count", mcp.Description("number of commits (default 20, max 500)")),
		stringArray("paths", "only commits touching these paths"),
		mcp.WithBoolean("stat", mcp.Description("include changed files per commit")),
	)
)

func init() {
	Tool.RegisterRead(server.ServerTool{Tool: StatusTool, Handler: StatusFn})
	Tool.RegisterRead(server.ServerTool{Tool: DiffTool, Handler: DiffFn})
	Tool.RegisterRead(server.ServerTool{Tool: LogTool, Handler: LogFn})
	Tool.RegisterRead(server.ServerTool{Tool: FetchTool, Handler: FetchFn})
	Tool.RegisterWrite(server.ServerTool{Tool: AddTool, Handler: AddFn})
	Tool.RegisterWrite(server.ServerTool{Tool: CommitTool, Handler: CommitFn})
	Tool.RegisterWrite(server.ServerTool{Tool: PushTool, Handler: PushFn})
	Tool.RegisterWrite(server.ServerTool{Tool: PullTool, Handler: PullFn})
	Tool.RegisterWrite(server.ServerTool{Tool: BranchTool, Handler: BranchFn})
	Tool.RegisterWrite(server.ServerTool{Tool: CheckoutTool, Handler: CheckoutFn})
}

// ---- argument helpers ----

func argString(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

func argBool(args map[string]any, key string) bool {
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	}
	return false
}

func argInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func argStrings(args map[string]any, key string) []string {
	var out []string
	switch v := args[key].(type) {
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
	case []string:
		out = append(out, v...)
	case string:
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func checkPaths(paths []string) error {
	for _, p := range paths {
		if strings.ContainsAny(p, "\x00\r\n") {
			return fmt.Errorf("invalid path %q", p)
		}
	}
	return nil
}

func textResult(s string) (*mcp.CallToolResult, error) {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		s = "(no output)"
	}
	return mcp.NewToolResultText(s), nil
}

func errResult(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

func gitErr(op string, r result) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf("git %s failed: %s", op, r.combined())), nil
}

func open(ctx context.Context, req mcp.CallToolRequest) (map[string]any, string, error) {
	args := req.GetArguments()
	dir, err := resolveRepo(ctx, argString(args, "repo_path"))
	return args, dir, err
}

// ---- handlers ----

func StatusFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	r := runGit(ctx, dir, defaultTimeout, nil, "", "status", "--short", "--branch", "--untracked-files=all")
	if r.err != nil {
		return gitErr("status", r)
	}
	out := "repo: " + dir + "\n" + r.stdout
	if strings.Count(strings.TrimSpace(r.stdout), "\n") == 0 {
		out += "(working tree clean)\n"
	}
	return textResult(out)
}

func DiffFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	target := argString(args, "target")
	if err := checkRef("target", target); err != nil {
		return errResult(err)
	}
	paths := argStrings(args, "paths")
	if err := checkPaths(paths); err != nil {
		return errResult(err)
	}
	a := []string{"diff", "--no-color", "--no-ext-diff"}
	if argBool(args, "staged") {
		a = append(a, "--cached")
	}
	switch {
	case argBool(args, "name_only"):
		a = append(a, "--name-status")
	case argBool(args, "stat"):
		a = append(a, "--stat")
	}
	if n := argInt(args, "context_lines", -1); n >= 0 {
		a = append(a, fmt.Sprintf("-U%d", n))
	}
	if target != "" {
		a = append(a, target)
	}
	a = append(a, "--")
	a = append(a, paths...)
	r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
	if r.err != nil {
		return gitErr("diff", r)
	}
	if strings.TrimSpace(r.stdout) == "" {
		return textResult("(no differences)")
	}
	return textResult(r.stdout)
}

func AddFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	paths := argStrings(args, "paths")
	all := argBool(args, "all")
	if !all && len(paths) == 0 {
		return errResult(errors.New("give paths to stage, or all=true"))
	}
	if err := checkPaths(paths); err != nil {
		return errResult(err)
	}
	a := []string{"add"}
	if all {
		a = append(a, "-A")
	}
	a = append(a, "--")
	a = append(a, paths...)
	r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
	if r.err != nil {
		return gitErr("add", r)
	}
	st := runGit(ctx, dir, defaultTimeout, nil, "", "status", "--short", "--branch")
	return textResult("staged.\n" + st.stdout)
}

func CommitFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	msg, _ := args["message"].(string)
	if strings.TrimSpace(msg) == "" {
		return errResult(errors.New("message is required"))
	}
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	a := []string{"commit", "--file=-", "--cleanup=strip"}
	if argBool(args, "all") {
		a = append(a, "--all")
	}
	name, email := argString(args, "author_name"), argString(args, "author_email")
	if name != "" || email != "" {
		if name == "" || email == "" {
			return errResult(errors.New("author_name and author_email must be given together"))
		}
		if strings.ContainsAny(name+email, "<>\x00\r\n") {
			return errResult(errors.New("invalid author"))
		}
		a = append(a, fmt.Sprintf("--author=%s <%s>", name, email))
	}
	r := runGit(ctx, dir, defaultTimeout, nil, msg, a...)
	if r.err != nil {
		return gitErr("commit", r)
	}
	show := runGit(ctx, dir, defaultTimeout, nil, "", "show", "--stat", "--no-color", "--format=commit %H%nAuthor: %an <%ae>%nDate:   %ad%n%n    %s%n", "HEAD")
	return textResult(show.stdout)
}

func PushFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	remote := argString(args, "remote")
	if remote == "" {
		remote = "origin"
	}
	branch := argString(args, "branch")
	remoteBranch := argString(args, "remote_branch")
	for k, v := range map[string]string{"remote": remote, "branch": branch, "remote_branch": remoteBranch} {
		if err := checkRef(k, v); err != nil {
			return errResult(err)
		}
	}
	if branch == "" {
		if branch, err = currentBranch(ctx, dir); err != nil {
			return errResult(err)
		}
	}
	if strings.ContainsAny(branch+remoteBranch, ":+*") {
		return errResult(errors.New("branch names must not contain ':', '+' or '*' (refspec syntax and force push are not allowed)"))
	}
	refspec := branch
	if remoteBranch != "" {
		refspec = branch + ":refs/heads/" + remoteBranch
	}
	a := []string{"push", "--porcelain"}
	if argBool(args, "set_upstream") {
		a = append(a, "--set-upstream")
	}
	if argBool(args, "tags") {
		a = append(a, "--follow-tags")
	}
	a = append(a, remote, refspec)
	env := authEnv(remoteURL(ctx, dir, remote, true))
	r := runGit(ctx, dir, networkTimeout, env, "", a...)
	if r.err != nil {
		return gitErr("push", r)
	}
	return textResult(r.combined())
}

func PullFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	remote := argString(args, "remote")
	branch := argString(args, "branch")
	if err := checkRef("remote", remote); err != nil {
		return errResult(err)
	}
	if err := checkRef("branch", branch); err != nil {
		return errResult(err)
	}
	if branch != "" && remote == "" {
		remote = "origin"
	}
	a := []string{"pull", "--no-edit"}
	if argBool(args, "rebase") {
		a = append(a, "--rebase")
	} else {
		a = append(a, "--ff-only")
	}
	urlRemote := remote
	if remote != "" {
		a = append(a, remote)
		if branch != "" {
			a = append(a, branch)
		}
	} else {
		urlRemote = upstreamRemote(ctx, dir)
	}
	env := authEnv(remoteURL(ctx, dir, urlRemote, false))
	r := runGit(ctx, dir, networkTimeout, env, "", a...)
	if r.err != nil {
		return gitErr("pull", r)
	}
	return textResult(r.combined())
}

func upstreamRemote(ctx context.Context, dir string) string {
	b, err := currentBranch(ctx, dir)
	if err == nil {
		r := runGit(ctx, dir, defaultTimeout, nil, "", "config", "--get", "branch."+b+".remote")
		if v := strings.TrimSpace(r.stdout); r.err == nil && v != "" {
			return v
		}
	}
	return "origin"
}

func FetchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	remote := argString(args, "remote")
	if remote == "" {
		remote = "origin"
	}
	if err := checkRef("remote", remote); err != nil {
		return errResult(err)
	}
	a := []string{"fetch"}
	if argBool(args, "prune") {
		a = append(a, "--prune")
	}
	if argBool(args, "tags") {
		a = append(a, "--tags")
	}
	a = append(a, remote)
	env := authEnv(remoteURL(ctx, dir, remote, false))
	r := runGit(ctx, dir, networkTimeout, env, "", a...)
	if r.err != nil {
		return gitErr("fetch", r)
	}
	out := r.combined()
	if out == "" {
		out = "fetched " + remote + " (already up to date)"
	}
	return textResult(out)
}

func BranchFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	action := argString(args, "action")
	name := argString(args, "name")
	start := argString(args, "start_point")
	if err := checkRef("name", name); err != nil {
		return errResult(err)
	}
	if err := checkRef("start_point", start); err != nil {
		return errResult(err)
	}
	switch action {
	case "", "list":
		a := []string{"branch", "--no-color", "-vv"}
		if argBool(args, "remotes") {
			a = append(a, "--all")
		}
		r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
		if r.err != nil {
			return gitErr("branch", r)
		}
		return textResult(r.stdout)
	case "create":
		if name == "" {
			return errResult(errors.New("name is required for create"))
		}
		a := []string{"branch", name}
		if start != "" {
			a = append(a, start)
		}
		r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
		if r.err != nil {
			return gitErr("branch", r)
		}
		return textResult("created branch " + name)
	case "delete":
		if name == "" {
			return errResult(errors.New("name is required for delete"))
		}
		r := runGit(ctx, dir, defaultTimeout, nil, "", "branch", "-d", name)
		if r.err != nil {
			return gitErr("branch -d", r)
		}
		return textResult(r.combined())
	default:
		return errResult(fmt.Errorf("unknown action %q (use list, create or delete)", action))
	}
}

func CheckoutFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	branch := argString(args, "branch")
	start := argString(args, "start_point")
	if branch == "" {
		return errResult(errors.New("branch is required"))
	}
	if err := checkRef("branch", branch); err != nil {
		return errResult(err)
	}
	if err := checkRef("start_point", start); err != nil {
		return errResult(err)
	}
	a := []string{"switch"}
	if argBool(args, "create") {
		a = append(a, "-c", branch)
		if start != "" {
			a = append(a, start)
		}
	} else {
		if start != "" {
			return errResult(errors.New("start_point is only valid with create=true"))
		}
		a = append(a, branch)
	}
	r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
	if r.err != nil {
		return gitErr("switch", r)
	}
	st := runGit(ctx, dir, defaultTimeout, nil, "", "status", "--short", "--branch")
	return textResult(strings.TrimSpace(r.combined()) + "\n" + st.stdout)
}

func LogFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, dir, err := open(ctx, req)
	if err != nil {
		return errResult(err)
	}
	ref := argString(args, "ref")
	if err := checkRef("ref", ref); err != nil {
		return errResult(err)
	}
	paths := argStrings(args, "paths")
	if err := checkPaths(paths); err != nil {
		return errResult(err)
	}
	n := argInt(args, "max_count", 20)
	if n <= 0 {
		n = 20
	}
	if n > 500 {
		n = 500
	}
	a := []string{"log", "--no-color", fmt.Sprintf("--max-count=%d", n),
		"--date=iso-strict", "--format=%h %ad %an%d%n    %s"}
	if argBool(args, "stat") {
		a = append(a, "--name-status")
	}
	if ref != "" {
		a = append(a, ref)
	}
	a = append(a, "--")
	a = append(a, paths...)
	r := runGit(ctx, dir, defaultTimeout, nil, "", a...)
	if r.err != nil {
		return gitErr("log", r)
	}
	if strings.TrimSpace(r.stdout) == "" {
		return textResult("(no commits)")
	}
	return textResult(r.stdout)
}
