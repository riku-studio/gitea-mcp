package pull

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"gitea.com/gitea/gitea-mcp/pkg/flag"

	"github.com/mark3labs/mcp-go/mcp"
)

func Test_editPullRequestFn(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 7
	)

	indexInputs := []struct {
		name string
		val  any
	}{
		{"float64", float64(index)},
		{"string", "7"},
	}

	for _, ii := range indexInputs {
		t.Run(ii.name, func(t *testing.T) {
			var (
				mu        sync.Mutex
				gotMethod string
				gotPath   string
				gotBody   map[string]any
			)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/version":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"private":false}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
					mu.Lock()
					gotMethod = r.Method
					gotPath = r.URL.Path
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					gotBody = body
					mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(fmt.Appendf(nil, `{"number":%d,"title":"%s","state":"open"}`, index, body["title"]))
				default:
					http.NotFound(w, r)
				}
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			origHost := flag.Host
			origToken := flag.Token
			origVersion := flag.Version
			flag.Host = server.URL
			flag.Token = ""
			flag.Version = "test"
			defer func() {
				flag.Host = origHost
				flag.Token = origToken
				flag.Version = origVersion
			}()

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{
						"owner":       owner,
						"repo":        repo,
						"pull_number": ii.val,
						"title":       "WIP: my feature",
						"state":       "open",
					},
				},
			}

			result, err := editPullRequestFn(context.Background(), req)
			if err != nil {
				t.Fatalf("editPullRequestFn() error = %v", err)
			}

			mu.Lock()
			defer mu.Unlock()

			if gotMethod != http.MethodPatch {
				t.Fatalf("expected PATCH request, got %s", gotMethod)
			}
			if gotPath != fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index) {
				t.Fatalf("unexpected path: %s", gotPath)
			}
			if gotBody["title"] != "WIP: my feature" {
				t.Fatalf("expected title 'WIP: my feature', got %v", gotBody["title"])
			}
			if gotBody["state"] != "open" {
				t.Fatalf("expected state 'open', got %v", gotBody["state"])
			}

			if len(result.Content) == 0 {
				t.Fatalf("expected content in result")
			}
			textContent, ok := mcp.AsTextContent(result.Content[0])
			if !ok {
				t.Fatalf("expected text content, got %T", result.Content[0])
			}

			var parsed map[string]any
			if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
				t.Fatalf("unmarshal result text: %v", err)
			}
			if got := parsed["title"].(string); got != "WIP: my feature" {
				t.Fatalf("result title = %q, want %q", got, "WIP: my feature")
			}
		})
	}
}

func Test_mergePullRequestFn(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 5
	)

	indexInputs := []struct {
		name string
		val  any
	}{
		{"float64", float64(index)},
		{"string", "5"},
	}

	for _, ii := range indexInputs {
		t.Run(ii.name, func(t *testing.T) {
			var (
				mu        sync.Mutex
				gotMethod string
				gotPath   string
				gotBody   map[string]any
			)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/version":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"private":false}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner, repo, index):
					mu.Lock()
					gotMethod = r.Method
					gotPath = r.URL.Path
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					gotBody = body
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
				default:
					http.NotFound(w, r)
				}
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			origHost := flag.Host
			origToken := flag.Token
			origVersion := flag.Version
			flag.Host = server.URL
			flag.Token = ""
			flag.Version = "test"
			defer func() {
				flag.Host = origHost
				flag.Token = origToken
				flag.Version = origVersion
			}()

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{
						"owner":         owner,
						"repo":          repo,
						"pull_number":   ii.val,
						"merge_style":   "squash",
						"title":         "feat: my squashed commit",
						"message":       "Squash merge of PR #5",
						"delete_branch": true,
					},
				},
			}

			result, err := mergePullRequestFn(context.Background(), req)
			if err != nil {
				t.Fatalf("mergePullRequestFn() error = %v", err)
			}

			mu.Lock()
			defer mu.Unlock()

			if gotMethod != http.MethodPost {
				t.Fatalf("expected POST request, got %s", gotMethod)
			}
			if gotPath != fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner, repo, index) {
				t.Fatalf("unexpected path: %s", gotPath)
			}
			if gotBody["Do"] != "squash" {
				t.Fatalf("expected Do 'squash', got %v", gotBody["Do"])
			}
			if gotBody["MergeTitleField"] != "feat: my squashed commit" {
				t.Fatalf("expected MergeTitleField 'feat: my squashed commit', got %v", gotBody["MergeTitleField"])
			}
			if gotBody["MergeMessageField"] != "Squash merge of PR #5" {
				t.Fatalf("expected MergeMessageField 'Squash merge of PR #5', got %v", gotBody["MergeMessageField"])
			}
			if gotBody["delete_branch_after_merge"] != true {
				t.Fatalf("expected delete_branch_after_merge true, got %v", gotBody["delete_branch_after_merge"])
			}

			if len(result.Content) == 0 {
				t.Fatalf("expected content in result")
			}
			textContent, ok := mcp.AsTextContent(result.Content[0])
			if !ok {
				t.Fatalf("expected text content, got %T", result.Content[0])
			}

			var parsed map[string]any
			if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
				t.Fatalf("unmarshal result text: %v", err)
			}
			if parsed["merged"] != true {
				t.Fatalf("expected merged=true, got %v", parsed["merged"])
			}
			if parsed["merge_style"] != "squash" {
				t.Fatalf("expected merge_style 'squash', got %v", parsed["merge_style"])
			}
			if parsed["branch_deleted"] != true {
				t.Fatalf("expected branch_deleted=true, got %v", parsed["branch_deleted"])
			}
		})
	}
}

func Test_mergePullRequestFn_newParams(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 8
	)

	var (
		mu      sync.Mutex
		gotBody map[string]any
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner, repo, index):
			mu.Lock()
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotBody = body
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	origHost := flag.Host
	origToken := flag.Token
	origVersion := flag.Version
	flag.Host = server.URL
	flag.Token = ""
	flag.Version = "test"
	defer func() {
		flag.Host = origHost
		flag.Token = origToken
		flag.Version = origVersion
	}()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"owner":                     owner,
				"repo":                      repo,
				"pull_number":               float64(index),
				"merge_style":               "merge",
				"force_merge":               true,
				"merge_when_checks_succeed": true,
				"head_commit_id":            "abc123",
			},
		},
	}

	_, err := mergePullRequestFn(context.Background(), req)
	if err != nil {
		t.Fatalf("mergePullRequestFn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotBody["force_merge"] != true {
		t.Fatalf("expected force_merge true, got %v", gotBody["force_merge"])
	}
	if gotBody["merge_when_checks_succeed"] != true {
		t.Fatalf("expected merge_when_checks_succeed true, got %v", gotBody["merge_when_checks_succeed"])
	}
	if gotBody["head_commit_id"] != "abc123" {
		t.Fatalf("expected head_commit_id 'abc123', got %v", gotBody["head_commit_id"])
	}
}

func Test_createPullRequestFn_labels(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
	)

	var (
		mu      sync.Mutex
		gotBody map[string]any
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner, repo):
			mu.Lock()
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotBody = body
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":1,"title":"test","state":"open"}`))
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	origHost := flag.Host
	origToken := flag.Token
	origVersion := flag.Version
	flag.Host = server.URL
	flag.Token = ""
	flag.Version = "test"
	defer func() {
		flag.Host = origHost
		flag.Token = origToken
		flag.Version = origVersion
	}()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"owner":    owner,
				"repo":     repo,
				"title":    "test",
				"body":     "body",
				"head":     "feature",
				"base":     "main",
				"labels":   []any{float64(1), float64(2)},
				"deadline": "2026-06-01T00:00:00Z",
			},
		},
	}

	_, err := createPullRequestFn(context.Background(), req)
	if err != nil {
		t.Fatalf("createPullRequestFn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	labels, ok := gotBody["labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %v", gotBody["labels"])
	}
	if labels[0] != float64(1) || labels[1] != float64(2) {
		t.Fatalf("expected labels [1,2], got %v", labels)
	}
	if gotBody["due_date"] == nil {
		t.Fatalf("expected due_date to be set")
	}
}

func Test_applyDraftPrefix(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		isDraft bool
		want    string
	}{
		{"add prefix", "my feature", true, "WIP: my feature"},
		{"already prefixed WIP:", "WIP: my feature", true, "WIP: my feature"},
		{"already prefixed WIP: no space", "WIP:my feature", true, "WIP:my feature"},
		{"already prefixed [WIP]", "[WIP] my feature", true, "[WIP] my feature"},
		{"already prefixed case insensitive", "wip: my feature", true, "wip: my feature"},
		{"already prefixed [wip]", "[wip] my feature", true, "[wip] my feature"},
		{"remove WIP: prefix", "WIP: my feature", false, "my feature"},
		{"remove WIP: no space", "WIP:my feature", false, "my feature"},
		{"remove [WIP] prefix", "[WIP] my feature", false, "my feature"},
		{"remove [wip] prefix", "[wip] my feature", false, "my feature"},
		{"remove wip: lowercase", "wip: my feature", false, "my feature"},
		{"no prefix not draft", "my feature", false, "my feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyDraftPrefix(tt.title, tt.isDraft)
			if got != tt.want {
				t.Fatalf("applyDraftPrefix(%q, %v) = %q, want %q", tt.title, tt.isDraft, got, tt.want)
			}
		})
	}
}

func Test_createPullRequestFn_draft(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
	)

	tests := []struct {
		name      string
		title     string
		draft     any // bool or nil (omitted)
		wantTitle string
	}{
		{"draft true", "my feature", true, "WIP: my feature"},
		{"draft false strips WIP:", "WIP: my feature", false, "my feature"},
		{"draft false strips [WIP]", "[WIP] my feature", false, "my feature"},
		{"draft omitted preserves title", "WIP: my feature", nil, "WIP: my feature"},
		{"draft true already prefixed", "WIP: my feature", true, "WIP: my feature"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu      sync.Mutex
				gotBody map[string]any
			)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/version":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"private":false}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner, repo):
					mu.Lock()
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					gotBody = body
					mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(fmt.Appendf(nil, `{"number":1,"title":%q,"state":"open"}`, body["title"]))
				default:
					http.NotFound(w, r)
				}
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			origHost := flag.Host
			origToken := flag.Token
			origVersion := flag.Version
			flag.Host = server.URL
			flag.Token = ""
			flag.Version = "test"
			defer func() {
				flag.Host = origHost
				flag.Token = origToken
				flag.Version = origVersion
			}()

			args := map[string]any{
				"owner": owner,
				"repo":  repo,
				"title": tc.title,
				"body":  "test body",
				"head":  "feature",
				"base":  "main",
			}
			if tc.draft != nil {
				args["draft"] = tc.draft
			}

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: args,
				},
			}

			_, err := createPullRequestFn(context.Background(), req)
			if err != nil {
				t.Fatalf("createPullRequestFn() error = %v", err)
			}

			mu.Lock()
			defer mu.Unlock()

			if gotBody["title"] != tc.wantTitle {
				t.Fatalf("expected title %q, got %v", tc.wantTitle, gotBody["title"])
			}
		})
	}
}

func Test_editPullRequestFn_draft(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 7
	)

	tests := []struct {
		name      string
		title     string // title arg passed to the tool; empty means omitted
		draft     any
		wantTitle string
	}{
		{"set draft with title", "my feature", true, "WIP: my feature"},
		{"unset draft with title", "WIP: my feature", false, "my feature"},
		{"set draft without title fetches current", "", true, "WIP: existing title"},
		{"unset draft without title fetches current", "", false, "existing title"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu      sync.Mutex
				gotBody map[string]any
			)

			prPath := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/version":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"private":false}`))
				case prPath:
					w.Header().Set("Content-Type", "application/json")
					if r.Method == http.MethodGet {
						// Auto-fetch: return the existing PR with its current title
						_, _ = w.Write(fmt.Appendf(nil, `{"number":%d,"title":"existing title","state":"open"}`, index))
						return
					}
					mu.Lock()
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					gotBody = body
					mu.Unlock()
					title := "existing title"
					if s, ok := body["title"].(string); ok {
						title = s
					}
					_, _ = w.Write(fmt.Appendf(nil, `{"number":%d,"title":%q,"state":"open"}`, index, title))
				default:
					http.NotFound(w, r)
				}
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			origHost := flag.Host
			origToken := flag.Token
			origVersion := flag.Version
			flag.Host = server.URL
			flag.Token = ""
			flag.Version = "test"
			defer func() {
				flag.Host = origHost
				flag.Token = origToken
				flag.Version = origVersion
			}()

			args := map[string]any{
				"owner":       owner,
				"repo":        repo,
				"pull_number": float64(index),
			}
			if tc.title != "" {
				args["title"] = tc.title
			}
			if tc.draft != nil {
				args["draft"] = tc.draft
			}

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: args,
				},
			}

			_, err := editPullRequestFn(context.Background(), req)
			if err != nil {
				t.Fatalf("editPullRequestFn() error = %v", err)
			}

			mu.Lock()
			defer mu.Unlock()

			if gotBody["title"] != tc.wantTitle {
				t.Fatalf("expected title %q, got %v", tc.wantTitle, gotBody["title"])
			}
		})
	}
}

func Test_getPullRequestDiffFn(t *testing.T) {
	const (
		owner   = "octo"
		repo    = "demo"
		index   = 12
		diffRaw = "diff --git a/file.txt b/file.txt\n+line\n"
	)

	indexInputs := []struct {
		name string
		val  any
	}{
		{"float64", float64(index)},
		{"string", "12"},
	}

	for _, ii := range indexInputs {
		t.Run(ii.name, func(t *testing.T) {
			var (
				mu            sync.Mutex
				diffRequested bool
				binaryValue   string
			)
			errCh := make(chan error, 1)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/version":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
				case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"private":false}`))
				case fmt.Sprintf("/%s/%s/pulls/%d.diff", owner, repo, index):
					if r.Method != http.MethodGet {
						select {
						case errCh <- fmt.Errorf("unexpected method: %s", r.Method):
						default:
						}
					}
					mu.Lock()
					diffRequested = true
					binaryValue = r.URL.Query().Get("binary")
					mu.Unlock()
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write([]byte(diffRaw))
				default:
					select {
					case errCh <- fmt.Errorf("unexpected request path: %s", r.URL.Path):
					default:
					}
				}
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			origHost := flag.Host
			origToken := flag.Token
			origVersion := flag.Version
			flag.Host = server.URL
			flag.Token = ""
			flag.Version = "test"
			defer func() {
				flag.Host = origHost
				flag.Token = origToken
				flag.Version = origVersion
			}()

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{
						"owner":       owner,
						"repo":        repo,
						"pull_number": ii.val,
						"binary":      true,
					},
				},
			}

			result, err := getPullRequestDiffFn(context.Background(), req)
			if err != nil {
				t.Fatalf("getPullRequestDiffFn() error = %v", err)
			}

			select {
			case reqErr := <-errCh:
				t.Fatalf("handler error: %v", reqErr)
			default:
			}

			mu.Lock()
			requested := diffRequested
			gotBinary := binaryValue
			mu.Unlock()

			if !requested {
				t.Fatalf("expected diff request to be made")
			}
			if gotBinary != "true" {
				t.Fatalf("expected binary=true query param, got %q", gotBinary)
			}

			if len(result.Content) == 0 {
				t.Fatalf("expected content in result")
			}

			textContent, ok := mcp.AsTextContent(result.Content[0])
			if !ok {
				t.Fatalf("expected text content, got %T", result.Content[0])
			}

			// The diff response is now a plain string
			var parsed string
			if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
				t.Fatalf("unmarshal result text: %v", err)
			}
			if parsed != diffRaw {
				t.Fatalf("diff = %q, want %q", parsed, diffRaw)
			}
		})
	}
}

func Test_getPullRequestByIndexFn_includesAttachments(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 9
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":9,"title":"feat","body":"see screenshot","state":"open"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", owner, repo, index):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"name":"shot.png","browser_download_url":"https://example/shot.png"}]`))
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	origHost, origToken, origVersion := flag.Host, flag.Token, flag.Version
	flag.Host, flag.Token, flag.Version = server.URL, "", "test"
	defer func() { flag.Host, flag.Token, flag.Version = origHost, origToken, origVersion }()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"owner": owner, "repo": repo, "pull_number": float64(index),
	}}}
	res, err := getPullRequestByIndexFn(context.Background(), req)
	if err != nil {
		t.Fatalf("getPullRequestByIndexFn() error = %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %v", res.Content)
	}
	body := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(body, `[shot.png](https://example/shot.png)`) {
		t.Fatalf("expected attachment markdown inlined in body, got: %s", body)
	}
}

func Test_getPullRequestByIndexFn_emptyAssetsLeavesBody(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 9
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":9,"title":"feat","body":"plain body","state":"open"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", owner, repo, index):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	origHost, origToken, origVersion := flag.Host, flag.Token, flag.Version
	flag.Host, flag.Token, flag.Version = server.URL, "", "test"
	defer func() { flag.Host, flag.Token, flag.Version = origHost, origToken, origVersion }()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"owner": owner, "repo": repo, "pull_number": float64(index),
	}}}
	res, err := getPullRequestByIndexFn(context.Background(), req)
	if err != nil {
		t.Fatalf("getPullRequestByIndexFn() error = %v", err)
	}
	body := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(body, `"body":"plain body"`) {
		t.Fatalf("expected body unchanged when assets are empty, got: %s", body)
	}
}

func Test_getPullRequestByIndexFn_assetsFailureNonFatal(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 9
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":9,"title":"feat","body":"plain body","state":"open"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", owner, repo, index):
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	origHost, origToken, origVersion := flag.Host, flag.Token, flag.Version
	flag.Host, flag.Token, flag.Version = server.URL, "", "test"
	defer func() { flag.Host, flag.Token, flag.Version = origHost, origToken, origVersion }()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"owner": owner, "repo": repo, "pull_number": float64(index),
	}}}
	res, err := getPullRequestByIndexFn(context.Background(), req)
	if err != nil {
		t.Fatalf("getPullRequestByIndexFn() error = %v", err)
	}
	if res.IsError {
		t.Fatalf("assets fetch failure should not fail the PR fetch: %v", res.Content)
	}
	body := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(body, `"plain body"`) {
		t.Fatalf("expected PR body preserved when assets fail, got: %s", body)
	}
}

func Test_closePullRequestFn(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 7
	)

	var gotBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH method, got %s", r.Method)
			}
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fmt.Appendf(nil, `{"index":%d,"title":"Fix bug","state":"closed","head":{"ref":"fix-branch"},"base":{"ref":"main"}}`, index))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	origHost := flag.Host
	origToken := flag.Token
	flag.Host = server.URL
	flag.Token = "test-token"
	t.Cleanup(func() { flag.Host = origHost; flag.Token = origToken })

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"method":      "close",
				"owner":       owner,
				"repo":        repo,
				"pull_number": float64(index),
			},
		},
	}

	result, err := closePullRequestFn(context.Background(), req)
	if err != nil {
		t.Fatalf("closePullRequestFn() error = %v", err)
	}

	if gotBody["state"] != "closed" {
		t.Errorf("expected state=closed, got %v", gotBody["state"])
	}

	if len(result.Content) == 0 {
		t.Fatalf("expected content in result")
	}
}

func Test_reopenPullRequestFn(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
		index = 7
	)

	var gotBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner, repo, index):
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH method, got %s", r.Method)
			}
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fmt.Appendf(nil, `{"index":%d,"title":"Fix bug","state":"open","head":{"ref":"fix-branch"},"base":{"ref":"main"}}`, index))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	origHost := flag.Host
	origToken := flag.Token
	flag.Host = server.URL
	flag.Token = "test-token"
	t.Cleanup(func() { flag.Host = origHost; flag.Token = origToken })

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"method":      "reopen",
				"owner":       owner,
				"repo":        repo,
				"pull_number": float64(index),
			},
		},
	}

	result, err := reopenPullRequestFn(context.Background(), req)
	if err != nil {
		t.Fatalf("reopenPullRequestFn() error = %v", err)
	}

	if gotBody["state"] != "open" {
		t.Errorf("expected state=open, got %v", gotBody["state"])
	}

	if len(result.Content) == 0 {
		t.Fatalf("expected content in result")
	}
}
