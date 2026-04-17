package issue

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

func Test_listRepoIssuesFn_filters(t *testing.T) {
	const (
		owner = "octo"
		repo  = "demo"
	)

	var (
		mu       sync.Mutex
		gotQuery string
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.12.0"}`))
		case r.URL.Path == fmt.Sprintf("/api/v1/repos/%s/%s", owner, repo):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"private":false}`))
		case r.URL.Path == fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner, repo):
			mu.Lock()
			gotQuery = r.URL.RawQuery
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
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
				"owner":  owner,
				"repo":   repo,
				"labels": []any{"bug", "enhancement"},
				"since":  "2026-01-01T00:00:00Z",
			},
		},
	}

	_, err := listRepoIssuesFn(context.Background(), req)
	if err != nil {
		t.Fatalf("listRepoIssuesFn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if !strings.Contains(gotQuery, "labels=bug%2Cenhancement") {
		t.Fatalf("expected labels query param, got %s", gotQuery)
	}
	if !strings.Contains(gotQuery, "since=2026-01-01") {
		t.Fatalf("expected since query param, got %s", gotQuery)
	}
}

func Test_createIssueFn_labels(t *testing.T) {
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
		case fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner, repo):
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
				"title":    "test issue",
				"body":     "body",
				"labels":   []any{float64(10), float64(20)},
				"deadline": "2026-06-01T00:00:00Z",
			},
		},
	}

	_, err := createIssueFn(context.Background(), req)
	if err != nil {
		t.Fatalf("createIssueFn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	labels, ok := gotBody["labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %v", gotBody["labels"])
	}
	if labels[0] != float64(10) || labels[1] != float64(20) {
		t.Fatalf("expected labels [10,20], got %v", labels)
	}
	if gotBody["due_date"] == nil {
		t.Fatalf("expected due_date to be set")
	}
}
