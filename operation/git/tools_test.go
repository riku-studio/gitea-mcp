package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gitea.com/gitea/gitea-mcp/pkg/flag"
	"gitea.com/gitea/gitea-mcp/pkg/tool"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func call(t *testing.T, fn tool.Handler, args map[string]any) (string, bool) {
	t.Helper()
	res, err := fn(context.Background(), args)
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func mustOK(t *testing.T, fn tool.Handler, args map[string]any) string {
	t.Helper()
	out, isErr := call(t, fn, args)
	if isErr {
		t.Fatalf("unexpected error: %s", out)
	}
	return out
}

func mustErr(t *testing.T, fn tool.Handler, args map[string]any, want string) {
	t.Helper()
	out, isErr := call(t, fn, args)
	if !isErr {
		t.Fatalf("expected error containing %q, got success: %s", want, out)
	}
	if !strings.Contains(out, want) {
		t.Fatalf("expected error containing %q, got: %s", want, out)
	}
}

// setup creates root/{remote.git,work,other} and points the allowlist at root.
func setup(t *testing.T) (root, work, other, remote string) {
	t.Helper()
	root = t.TempDir()
	remote = filepath.Join(root, "remote.git")
	work = filepath.Join(root, "work")
	other = filepath.Join(root, "other")
	sh(t, root, "init", "-q", "--bare", "-b", "main", remote)
	sh(t, root, "clone", "-q", remote, work)
	for _, d := range []string{work} {
		sh(t, d, "config", "user.name", "Tester")
		sh(t, d, "config", "user.email", "tester@example.com")
		sh(t, d, "symbolic-ref", "HEAD", "refs/heads/main")
	}
	t.Setenv(AllowedRootsEnv, root)
	return
}

func TestDisabledWithoutRoots(t *testing.T) {
	t.Setenv(AllowedRootsEnv, "")
	mustErr(t, StatusFn, map[string]any{"repo_path": t.TempDir()}, "git tools are disabled")
}

func TestOutsideRootsRejected(t *testing.T) {
	setup(t)
	outside := t.TempDir()
	sh(t, outside, "init", "-q")
	mustErr(t, StatusFn, map[string]any{"repo_path": outside}, "outside the allowed roots")
	mustErr(t, StatusFn, map[string]any{"repo_path": "relative/path"}, "must be absolute")
}

func TestFullWorkflow(t *testing.T) {
	root, work, other, _ := setup(t)

	// status on an empty repo
	out := mustOK(t, StatusFn, map[string]any{"repo_path": work})
	if !strings.Contains(out, "main") {
		t.Fatalf("status: %s", out)
	}

	// add + commit a new file (content never travels through the tool call)
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustErr(t, AddFn, map[string]any{"repo_path": work}, "give paths")
	out = mustOK(t, AddFn, map[string]any{"repo_path": work, "paths": []any{"a.txt"}})
	if !strings.Contains(out, "A  a.txt") {
		t.Fatalf("add: %s", out)
	}
	out = mustOK(t, DiffFn, map[string]any{"repo_path": work, "staged": true})
	if !strings.Contains(out, "+hello") {
		t.Fatalf("diff staged: %s", out)
	}
	msg := "feat: add a.txt\n\nBody line.\n\nCo-Authored-By: X <x@example.com>"
	out = mustOK(t, CommitFn, map[string]any{"repo_path": work, "message": msg})
	if !strings.Contains(out, "feat: add a.txt") || !strings.Contains(out, "a.txt") {
		t.Fatalf("commit: %s", out)
	}
	full := sh(t, work, "log", "-1", "--format=%B")
	if !strings.Contains(full, "Co-Authored-By: X <x@example.com>") {
		t.Fatalf("commit message body lost: %q", full)
	}

	// nothing staged -> commit fails cleanly
	mustErr(t, CommitFn, map[string]any{"repo_path": work, "message": "empty"}, "git commit failed")

	// push with upstream
	out = mustOK(t, PushFn, map[string]any{"repo_path": work, "set_upstream": true})
	if !strings.Contains(out, "refs/heads/main") {
		t.Fatalf("push: %s", out)
	}

	// second clone, commit there, then pull --ff-only into work
	sh(t, root, "clone", "-q", filepath.Join(root, "remote.git"), other)
	sh(t, other, "config", "user.name", "Other")
	sh(t, other, "config", "user.email", "other@example.com")
	os.WriteFile(filepath.Join(other, "b.txt"), []byte("b\n"), 0o644)
	sh(t, other, "add", "b.txt")
	sh(t, other, "commit", "-q", "-m", "add b")
	sh(t, other, "push", "-q", "origin", "main")

	mustOK(t, FetchFn, map[string]any{"repo_path": work})
	mustOK(t, PullFn, map[string]any{"repo_path": work})
	if _, err := os.Stat(filepath.Join(work, "b.txt")); err != nil {
		t.Fatalf("pull did not bring b.txt: %v", err)
	}

	// log
	out = mustOK(t, LogFn, map[string]any{"repo_path": work, "max_count": 5})
	if !strings.Contains(out, "add b") || !strings.Contains(out, "feat: add a.txt") {
		t.Fatalf("log: %s", out)
	}

	// branch create / list / checkout / delete
	mustOK(t, BranchFn, map[string]any{"repo_path": work, "action": "create", "name": "feature/x"})
	out = mustOK(t, BranchFn, map[string]any{"repo_path": work})
	if !strings.Contains(out, "feature/x") {
		t.Fatalf("branch list: %s", out)
	}
	mustOK(t, CheckoutFn, map[string]any{"repo_path": work, "branch": "feature/x"})
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("changed\n"), 0o644)
	out = mustOK(t, DiffFn, map[string]any{"repo_path": work, "name_only": true})
	if !strings.Contains(out, "a.txt") {
		t.Fatalf("diff name_only: %s", out)
	}
	mustOK(t, CommitFn, map[string]any{"repo_path": work, "message": "change a", "all": true})
	mustOK(t, PushFn, map[string]any{"repo_path": work, "set_upstream": true})
	if got := sh(t, root, "--git-dir", filepath.Join(root, "remote.git"), "branch", "--list", "feature/x"); !strings.Contains(got, "feature/x") {
		t.Fatalf("remote missing feature/x: %q", got)
	}
	mustOK(t, CheckoutFn, map[string]any{"repo_path": work, "branch": "topic", "create": true, "start_point": "main"})
	mustOK(t, CheckoutFn, map[string]any{"repo_path": work, "branch": "main"})
	// feature/x is pushed, so -d accepts it via its upstream
	mustOK(t, BranchFn, map[string]any{"repo_path": work, "action": "delete", "name": "topic"})

	// status clean
	out = mustOK(t, StatusFn, map[string]any{"repo_path": work})
	if !strings.Contains(out, "working tree clean") {
		t.Fatalf("status clean: %s", out)
	}
}

func TestOptionInjectionRejected(t *testing.T) {
	_, work, _, _ := setup(t)
	mustErr(t, CheckoutFn, map[string]any{"repo_path": work, "branch": "--orphan"}, "must not start with '-'")
	mustErr(t, PushFn, map[string]any{"repo_path": work, "branch": "+main"}, "must not contain")
	mustErr(t, PushFn, map[string]any{"repo_path": work, "branch": "main:refs/heads/x"}, "must not contain")
	mustErr(t, PushFn, map[string]any{"repo_path": work, "remote": "--receive-pack=evil"}, "must not start with '-'")
	mustErr(t, LogFn, map[string]any{"repo_path": work, "ref": "--output=/tmp/x"}, "must not start with '-'")
	mustErr(t, DiffFn, map[string]any{"repo_path": work, "target": "--output=/tmp/x"}, "must not start with '-'")
	mustErr(t, BranchFn, map[string]any{"repo_path": work, "action": "delete", "name": "-D"}, "must not start with '-'")
}

func TestPathsAfterDoubleDash(t *testing.T) {
	_, work, _, _ := setup(t)
	// A file literally named "-A" must be treated as a path, not an option.
	os.WriteFile(filepath.Join(work, "-A"), []byte("x\n"), 0o644)
	os.WriteFile(filepath.Join(work, "keep.txt"), []byte("x\n"), 0o644)
	mustOK(t, AddFn, map[string]any{"repo_path": work, "paths": []any{"-A"}})
	st := sh(t, work, "status", "--short")
	if !strings.Contains(st, "A  -A") || !strings.Contains(st, "?? keep.txt") {
		t.Fatalf("paths were not passed after --: %s", st)
	}
}

func TestAuthEnv(t *testing.T) {
	oldHost, oldTok, oldIns := flag.Host, flag.Token, flag.Insecure
	t.Cleanup(func() { flag.Host, flag.Token, flag.Insecure = oldHost, oldTok, oldIns })
	flag.Host, flag.Token = "https://git.lan", "secret"

	env := authEnv("https://git.lan/riku-studio/x.git")
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "GIT_CONFIG_VALUE_0=Authorization: token secret") || !strings.Contains(joined, "GIT_CONFIG_COUNT=1") {
		t.Fatalf("expected token header env, got %v", env)
	}
	for _, u := range []string{
		"https://github.com/riku-studio/x.git",
		"ssh://git@git.lan:2222/riku-studio/x.git",
		"http://git.lan/riku-studio/x.git",
		"",
	} {
		if e := authEnv(u); e != nil {
			t.Fatalf("token must not be sent to %q: %v", u, e)
		}
	}
	flag.Insecure = true
	if !strings.Contains(strings.Join(authEnv("https://git.lan/a/b.git"), "\n"), "sslVerify") {
		t.Fatal("insecure should disable sslVerify for the gitea host")
	}
}

func TestToolNamesRegistered(t *testing.T) {
	want := []string{StatusToolName, DiffToolName, AddToolName, CommitToolName, PushToolName,
		PullToolName, FetchToolName, BranchToolName, CheckoutToolName, LogToolName}
	got := map[string]bool{}
	for _, st := range Tool.Tools() {
		got[st.Tool.Name] = true
	}
	for _, n := range want {
		if !got[n] {
			t.Errorf("tool %s not registered", n)
		}
	}
}

func TestSubdirAndRootAboveAllowlist(t *testing.T) {
	_, work, _, _ := setup(t)
	sub := filepath.Join(work, "x", "y")
	os.MkdirAll(sub, 0o755)
	out := mustOK(t, StatusFn, map[string]any{"repo_path": sub})
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "repo: ") || !strings.EqualFold(filepath.Base(strings.TrimPrefix(first, "repo: ")), "work") {
		t.Fatalf("subdir should resolve to repo root %s: %s", work, out)
	}
	// Allowlist only a subdirectory: the repo root lies above it -> refuse.
	t.Setenv(AllowedRootsEnv, filepath.Join(work, "x"))
	mustErr(t, StatusFn, map[string]any{"repo_path": sub}, "outside the allowed roots")
}
