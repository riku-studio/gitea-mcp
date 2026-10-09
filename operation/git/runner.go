// Package git exposes local git working-copy operations (status, diff, add,
// commit, push, pull, fetch, branch, checkout, log) as MCP tools.
//
// Unlike the Gitea API tools, these run the local `git` executable against a
// repository on the machine where this MCP server runs, so file contents never
// have to be transferred through tool arguments.
//
// Safety:
//   - Every call must name a repo_path inside one of the roots listed in
//     GITEA_GIT_ALLOWED_ROOTS (os.PathListSeparator separated, ";" on
//     Windows). If the variable is empty, all git tools refuse to run.
//   - No force push, no hard reset, no file-discarding checkout.
//   - User-supplied refs/remotes may not start with "-" and paths are always
//     passed after "--", so arguments cannot be turned into git options.
//   - When the remote is an https URL on the configured Gitea host, the
//     server's Gitea token is injected via GIT_CONFIG_* environment variables
//     (never on the command line, never echoed back).
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gitea.com/gitea/gitea-mcp/pkg/flag"
)

const (
	// AllowedRootsEnv lists directories git tools may operate in.
	AllowedRootsEnv = "GITEA_GIT_ALLOWED_ROOTS"
	// GitBinaryEnv optionally overrides the git executable path.
	GitBinaryEnv = "GITEA_GIT_BINARY"

	defaultTimeout = 2 * time.Minute
	networkTimeout = 10 * time.Minute
	maxOutputBytes = 256 * 1024
)

var errNoRoots = fmt.Errorf("git tools are disabled: set %s to the directories that contain your repositories", AllowedRootsEnv)

// allowedRoots returns cleaned absolute roots from the environment.
func allowedRoots() []string {
	raw := os.Getenv(AllowedRootsEnv)
	var roots []string
	for _, r := range filepath.SplitList(raw) {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		abs, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		roots = append(roots, filepath.Clean(abs))
	}
	return roots
}

// within reports whether p is root or a descendant of root.
func within(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func withinAny(p string, roots []string) bool {
	for _, r := range roots {
		if within(p, r) {
			return true
		}
		// filepath.Rel is case-sensitive; Windows/macOS file systems are not.
		if (runtime.GOOS == "windows" || runtime.GOOS == "darwin") && within(strings.ToLower(p), strings.ToLower(r)) {
			return true
		}
	}
	return false
}

// resolveRepo validates repoPath and returns the work-tree top level.
func resolveRepo(ctx context.Context, repoPath string) (string, error) {
	roots := allowedRoots()
	if len(roots) == 0 {
		return "", errNoRoots
	}
	if strings.TrimSpace(repoPath) == "" {
		return "", errors.New("repo_path is required (absolute path of a local git working copy)")
	}
	if !filepath.IsAbs(repoPath) {
		return "", fmt.Errorf("repo_path must be absolute, got %q", repoPath)
	}
	p := filepath.Clean(repoPath)
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("repo_path %q: %v", repoPath, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("repo_path %q is not a directory", repoPath)
	}
	if !withinAny(p, roots) {
		return "", fmt.Errorf("repo_path %q is outside the allowed roots (%s=%s)", repoPath, AllowedRootsEnv, strings.Join(roots, string(os.PathListSeparator)))
	}
	// Derive the work-tree root from p itself (p minus git's prefix) instead
	// of trusting --show-toplevel's spelling, which on Windows may differ
	// (8.3 short names, drive-letter case, forward slashes).
	res := runGit(ctx, p, defaultTimeout, nil, "", "rev-parse", "--is-inside-work-tree", "--show-prefix")
	lines := strings.Split(strings.ReplaceAll(res.stdout, "\r", ""), "\n")
	if res.err != nil || len(lines) == 0 || strings.TrimSpace(lines[0]) != "true" {
		return "", fmt.Errorf("%q is not inside a git working tree: %s", repoPath, res.combined())
	}
	prefix := ""
	if len(lines) > 1 {
		prefix = strings.Trim(strings.TrimSpace(lines[1]), "/")
	}
	top := p
	if prefix != "" {
		for range strings.Split(prefix, "/") {
			top = filepath.Dir(top)
		}
	}
	if !withinAny(top, roots) {
		return "", fmt.Errorf("repository root %q is outside the allowed roots", top)
	}
	return top, nil
}

type result struct {
	stdout string
	stderr string
	err    error
}

func (r result) combined() string {
	out := strings.TrimSpace(r.stdout)
	errOut := strings.TrimSpace(r.stderr)
	switch {
	case out == "" && errOut == "" && r.err != nil:
		return r.err.Error()
	case out == "":
		return errOut
	case errOut == "":
		return out
	default:
		return out + "\n" + errOut
	}
}

func gitBinary() string {
	if b := strings.TrimSpace(os.Getenv(GitBinaryEnv)); b != "" {
		return b
	}
	return "git"
}

func truncate(s string) string {
	if len(s) <= maxOutputBytes {
		return s
	}
	return s[:maxOutputBytes] + fmt.Sprintf("\n... [output truncated, %d bytes total]", len(s))
}

// runGit executes git in dir. extraEnv is appended to the environment; stdin
// is fed to the process when non-empty.
func runGit(ctx context.Context, dir string, timeout time.Duration, extraEnv []string, stdin string, args ...string) result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBinary(), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // never block waiting for a password
		"GCM_INTERACTIVE=never", // Git Credential Manager: no GUI prompts
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"LC_ALL=C", // stable, English output
		"GIT_PAGER=cat",
		"GIT_EDITOR=true", // never open an editor
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("git %s timed out after %s", firstArg(args), timeout)
	}
	return result{stdout: truncate(stdout.String()), stderr: truncate(stderr.String()), err: err}
}

func firstArg(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// checkRef rejects values that git could interpret as an option.
func checkRef(kind, v string) error {
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "-") {
		return fmt.Errorf("%s %q must not start with '-'", kind, v)
	}
	if strings.ContainsAny(v, "\x00\r\n") {
		return fmt.Errorf("%s contains invalid characters", kind)
	}
	return nil
}

// authEnv returns GIT_CONFIG_* variables that add the Gitea token as an
// Authorization header, but only when remoteURL is https on the Gitea host.
func authEnv(remoteURL string) []string {
	if flag.Token == "" || flag.Host == "" {
		return nil
	}
	ru, err := url.Parse(strings.TrimSpace(remoteURL))
	if err != nil || ru.Scheme != "https" && ru.Scheme != "http" {
		return nil
	}
	hu, err := url.Parse(flag.Host)
	if err != nil || hu.Host == "" {
		return nil
	}
	if !strings.EqualFold(ru.Host, hu.Host) || ru.Scheme != hu.Scheme {
		return nil
	}
	base := hu.Scheme + "://" + hu.Host + "/"
	env := []string{
		"GIT_CONFIG_KEY_0=http." + base + ".extraHeader",
		"GIT_CONFIG_VALUE_0=Authorization: token " + flag.Token,
	}
	n := 1
	if flag.Insecure {
		env = append(env,
			"GIT_CONFIG_KEY_1=http."+base+".sslVerify",
			"GIT_CONFIG_VALUE_1=false",
		)
		n = 2
	}
	return append(env, fmt.Sprintf("GIT_CONFIG_COUNT=%d", n))
}

// remoteURL returns the push (or fetch) URL of remote.
func remoteURL(ctx context.Context, dir, remote string, push bool) string {
	args := []string{"remote", "get-url"}
	if push {
		args = append(args, "--push")
	}
	args = append(args, remote)
	res := runGit(ctx, dir, defaultTimeout, nil, "", args...)
	if res.err != nil {
		return ""
	}
	return strings.TrimSpace(res.stdout)
}

func currentBranch(ctx context.Context, dir string) (string, error) {
	res := runGit(ctx, dir, defaultTimeout, nil, "", "symbolic-ref", "--short", "-q", "HEAD")
	b := strings.TrimSpace(res.stdout)
	if res.err != nil || b == "" {
		return "", errors.New("HEAD is detached; specify a branch explicitly")
	}
	return b, nil
}
