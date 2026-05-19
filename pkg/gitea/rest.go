package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	mcpContext "gitea.com/gitea/gitea-mcp/pkg/context"
	"gitea.com/gitea/gitea-mcp/pkg/flag"
)

const (
	httpClientTimeout  = 60 * time.Second
	errBodySnippetSize = 8192
)

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, e.Body)
}

func tokenFromContext(ctx context.Context) string {
	if ctx != nil {
		if token, ok := ctx.Value(mcpContext.TokenContextKey).(string); ok && token != "" {
			return token
		}
	}
	return flag.Token
}

var (
	restClientOnce sync.Once
	restClient     *http.Client
)

func restHTTPClient() *http.Client {
	restClientOnce.Do(func() {
		restClient = &http.Client{
			Transport:     sharedTransport(),
			Timeout:       httpClientTimeout,
			CheckRedirect: checkRedirect,
		}
	})
	return restClient
}

func buildAPIURL(path string, query url.Values) (string, error) {
	host := strings.TrimRight(flag.Host, "/")
	if host == "" {
		return "", errors.New("gitea host is empty")
	}
	p := strings.TrimLeft(path, "/")
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/%s", host, p))
	if err != nil {
		return "", err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

// DoJSON performs an API request and decodes a JSON response into respOut (if non-nil).
// It returns the HTTP status code.
func DoJSON(ctx context.Context, method, path string, query url.Values, body, respOut any) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	u, err := buildAPIURL(path, query)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	token := tokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := restHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, errBodySnippetSize))
		return resp.StatusCode, &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodySnippet))}
	}

	if respOut == nil {
		_, _ = io.Copy(io.Discard, resp.Body) // best-effort
		return resp.StatusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(respOut); err != nil {
		return resp.StatusCode, fmt.Errorf("decode response: %w", err)
	}
	return resp.StatusCode, nil
}

// DoBytes performs an API request and returns the raw response bytes.
// It returns the HTTP status code.
func DoBytes(ctx context.Context, method, path string, query url.Values, body any, accept string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	u, err := buildAPIURL(path, query)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	token := tokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := restHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodySnippet := respBytes
		if len(bodySnippet) > errBodySnippetSize {
			bodySnippet = bodySnippet[:errBodySnippetSize]
		}
		return nil, resp.StatusCode, &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodySnippet))}
	}

	return respBytes, resp.StatusCode, nil
}
