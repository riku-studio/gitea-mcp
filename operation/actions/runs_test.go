package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitea.com/gitea/gitea-mcp/pkg/flag"
)

func TestListRepoActionRunsForwardsFilters(t *testing.T) {
	var gotRequest string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r.URL.RequestURI()
		_, _ = w.Write([]byte(`{"total_count":0,"workflow_runs":[]}`))
	}))
	defer server.Close()
	defer func(host string) { flag.Host = host }(flag.Host)
	flag.Host = server.URL

	result, err := listRepoActionRunsFn(t.Context(), map[string]any{"owner": "o", "repo": "r", "head_sha": "abc123", "branch": "main"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "/api/v1/repos/o/r/actions/runs?branch=main&head_sha=abc123&limit=30&page=1"; result.IsError || gotRequest != want {
		t.Errorf("request %q, isError %v, want %q", gotRequest, result.IsError, want)
	}
}
