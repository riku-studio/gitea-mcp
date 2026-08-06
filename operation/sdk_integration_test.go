package operation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mcpContext "gitea.com/gitea/gitea-mcp/pkg/context"
	"gitea.com/gitea/gitea-mcp/pkg/flag"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Pin negotiated versions so SDK upgrades require compatibility review.
const (
	testServerVersion                   = "test-version"
	expectedProtocolVersion             = "2026-07-28"
	expectedStatefulHTTPProtocolVersion = "2025-11-25"
)

func exposeAllTools(t *testing.T) {
	t.Helper()
	originalReadOnly := flag.ReadOnly
	originalAllowedTools := flag.AllowedTools
	originalAllowedScopes := flag.AllowedScopes
	originalVersion := flag.Version
	t.Cleanup(func() {
		flag.ReadOnly = originalReadOnly
		flag.AllowedTools = originalAllowedTools
		flag.AllowedScopes = originalAllowedScopes
		flag.Version = originalVersion
	})
	flag.ReadOnly = false
	flag.AllowedTools = nil
	flag.AllowedScopes = nil
	flag.Version = testServerVersion
}

// registeredToolCount is what the registry exposes under the current flags, so
// the transport assertions track tool additions without being edited.
func registeredToolCount() int {
	count := 0
	for _, domain := range domainTools {
		count += len(domain.Tools())
	}
	return count
}

// stdioCommandEnvironment removes variables that override subprocess flags.
func stdioCommandEnvironment() []string {
	environment := os.Environ()
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "GITEA_READONLY", "GITEA_SCOPES", "GITEA_TOOLS", "MCP_MODE":
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func textContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return content.Text
}

// listAndCallVersion is the round trip every transport must support. wantText
// differs per transport: the stdio subprocess resolves its version from the VCS
// build info (main.go:14), so only the in-process servers have a known one.
func listAndCallVersion(ctx context.Context, t *testing.T, session *mcp.ClientSession, wantText string) {
	t.Helper()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if want := registeredToolCount(); len(result.Tools) != want {
		t.Fatalf("ListTools() count = %d, want %d", len(result.Tools), want)
	}
	callResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_gitea_mcp_server_version",
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if got := textContent(t, callResult); !strings.Contains(got, wantText) {
		t.Errorf("version tool result = %q, want it to contain %q", got, wantText)
	}
}

func TestOfficialSDKInMemory(t *testing.T) {
	exposeAllTools(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := newMCPServer(testServerVersion)
	RegisterTool(server)
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "gitea-mcp-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if got := session.InitializeResult().ProtocolVersion; got != expectedProtocolVersion {
		t.Errorf("protocol version = %q, want %q", got, expectedProtocolVersion)
	}
	listAndCallVersion(ctx, t, session, testServerVersion)
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-serverDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("server Run() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatal("server did not stop after the client session closed")
	}
}

func TestStreamableHTTPStateful(t *testing.T) {
	exposeAllTools(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := newMCPServer(testServerVersion)
	RegisterTool(server)
	httpTestServer := httptest.NewServer(newHTTPServer("", server).Handler)
	defer httpTestServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "gitea-mcp-http-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpTestServer.URL + "/mcp",
		HTTPClient:           httpTestServer.Client(),
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()
	// Stateful Streamable HTTP cannot negotiate the sessionless 2026 protocol.
	if got := session.InitializeResult().ProtocolVersion; got != expectedStatefulHTTPProtocolVersion {
		t.Errorf("protocol version = %q, want %q", got, expectedStatefulHTTPProtocolVersion)
	}
	listAndCallVersion(ctx, t, session, testServerVersion)

	response, err := httpTestServer.Client().Get(httpTestServer.URL + "/not-mcp")
	if err != nil {
		t.Fatalf("GET outside /mcp error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("GET outside /mcp status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

// spaceReader yields an endless run of spaces, so oversized bodies can be sent
// without allocating them.
type spaceReader struct{}

func (spaceReader) Read(p []byte) (int, error) {
	for index := range p {
		p[index] = ' '
	}
	return len(p), nil
}

func TestStreamableHTTPRequestBodyLimit(t *testing.T) {
	server := newMCPServer(testServerVersion)
	httpTestServer := httptest.NewServer(newHTTPServer("", server).Handler)
	defer httpTestServer.Close()

	for _, test := range []struct {
		name     string
		size     int64
		tooLarge bool
	}{
		{name: "above the SDK default", size: mcp.DefaultMaxRequestBodyBytes + 1},
		{name: "above our own limit", size: maxRequestBodyBytes + 1, tooLarge: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, httpTestServer.URL+"/mcp", io.LimitReader(spaceReader{}, test.size))
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			request.ContentLength = test.size
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			response, err := httpTestServer.Client().Do(request)
			if err != nil {
				t.Fatalf("POST %d bytes error = %v", test.size, err)
			}
			defer response.Body.Close()
			if gotTooLarge := response.StatusCode == http.StatusRequestEntityTooLarge; gotTooLarge != test.tooLarge {
				t.Errorf("POST %d bytes status = %d, want %d = %v", test.size, response.StatusCode, http.StatusRequestEntityTooLarge, test.tooLarge)
			}
		})
	}
}

type authorizationTransport struct {
	base  http.RoundTripper
	mu    sync.RWMutex
	value string
}

func (t *authorizationTransport) set(value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.value = value
}

func (t *authorizationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context()) // Clone already copies the header
	t.mu.RLock()
	value := t.value
	t.mu.RUnlock()
	if value != "" {
		clone.Header.Set("Authorization", value)
	}
	return t.base.RoundTrip(clone)
}

func authContextValue(ctx context.Context, session *mcp.ClientSession) (string, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "test_auth_context"})
	if err != nil {
		return "", err
	}
	if len(result.Content) != 1 {
		return "", fmt.Errorf("content count = %d, want 1", len(result.Content))
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return content.Text, nil
}

func TestHTTPAuthPerRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := newMCPServer(testServerVersion)
	server.AddTool(
		&mcp.Tool{
			Name:        "test_auth_context",
			Description: "Return the request-scoped authentication token.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			token, _ := ctx.Value(mcpContext.TokenContextKey).(string)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: token}},
			}, nil
		},
	)
	httpTestServer := httptest.NewServer(newHTTPServer("", server).Handler)
	defer httpTestServer.Close()

	baseTransport := httpTestServer.Client().Transport
	auth := &authorizationTransport{base: baseTransport}
	auth.set("Bearer first-token")
	baseClient := &http.Client{Transport: auth}
	client := mcp.NewClient(&mcp.Implementation{Name: "gitea-mcp-auth-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpTestServer.URL + "/mcp",
		HTTPClient:           baseClient,
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()

	for _, test := range []struct {
		header string
		want   string
	}{
		{header: "Bearer first-token", want: "first-token"},
		{header: "token second-token", want: "second-token"},
		{header: "Basic ignored", want: ""},
	} {
		auth.set(test.header)
		token, err := authContextValue(ctx, session)
		if err != nil {
			t.Fatalf("CallTool() with %q error = %v", test.header, err)
		}
		if token != test.want {
			t.Errorf("CallTool() token = %q, want %q", token, test.want)
		}
	}

	type authenticatedSession struct {
		session *mcp.ClientSession
		want    string
	}
	concurrentSessions := make([]authenticatedSession, 0, 2)
	for index, token := range []string{"parallel-one", "parallel-two"} {
		transport := &authorizationTransport{base: baseTransport}
		transport.set("Bearer " + token)
		httpClient := &http.Client{Transport: transport}
		parallelClient := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gitea-mcp-auth-parallel-%d", index),
			Version: "1",
		}, nil)
		parallelSession, err := parallelClient.Connect(ctx, &mcp.StreamableClientTransport{
			Endpoint:             httpTestServer.URL + "/mcp",
			HTTPClient:           httpClient,
			DisableStandaloneSSE: true,
			MaxRetries:           -1,
		}, nil)
		if err != nil {
			t.Fatalf("parallel Connect() error = %v", err)
		}
		defer parallelSession.Close()
		concurrentSessions = append(concurrentSessions, authenticatedSession{session: parallelSession, want: token})
	}

	var waitGroup sync.WaitGroup
	errorsCh := make(chan error, 20)
	for _, authenticated := range concurrentSessions {
		for range 10 {
			waitGroup.Go(func() {
				got, err := authContextValue(ctx, authenticated.session)
				if err != nil {
					errorsCh <- err
					return
				}
				if got != authenticated.want {
					errorsCh <- fmt.Errorf("parallel token = %q, want %q", got, authenticated.want)
				}
			})
		}
	}
	waitGroup.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
}

func TestStdioCommandTransport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess build in short mode")
	}
	for name, value := range map[string]string{
		"GITEA_READONLY": "true",
		"GITEA_SCOPES":   "user",
		"GITEA_TOOLS":    "get_me",
		"MCP_MODE":       "http",
	} {
		t.Setenv(name, value)
	}
	exposeAllTools(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "gitea-mcp")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build stdio test binary: %v\n%s", err, output)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "gitea-mcp-stdio-test", Version: "1"}, nil)
	command := exec.CommandContext(ctx, binary, "--transport", "stdio")
	command.Env = stdioCommandEnvironment()
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command:           command,
		TerminateDuration: 2 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()
	if got := session.InitializeResult().ProtocolVersion; got != expectedProtocolVersion {
		t.Errorf("protocol version = %q, want %q", got, expectedProtocolVersion)
	}
	listAndCallVersion(ctx, t, session, "Gitea MCP Server version:")
}
