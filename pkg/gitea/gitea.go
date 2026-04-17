package gitea

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	mcpContext "gitea.com/gitea/gitea-mcp/pkg/context"
	"gitea.com/gitea/gitea-mcp/pkg/flag"

	"code.gitea.io/sdk/gitea"
)

func NewClient(token string) (*gitea.Client, error) {
	httpClient := &http.Client{
		Transport:     http.DefaultTransport,
		CheckRedirect: checkRedirect,
	}

	opts := []gitea.ClientOption{
		gitea.SetToken(token),
	}
	if flag.Insecure {
		httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	opts = append(opts, gitea.SetHTTPClient(httpClient))
	if flag.Debug {
		opts = append(opts, gitea.SetDebugMode())
	}
	client, err := gitea.NewClient(flag.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gitea client err: %w", err)
	}

	// Set user agent for the client
	client.SetUserAgent("gitea-mcp-server/" + flag.Version)
	return client, nil
}

// checkRedirect prevents Go from silently changing mutating requests (POST, PATCH, etc.)
// to GET when following 301/302/303 redirects, which would drop the request body and
// make writes appear to succeed when they didn't.
func checkRedirect(_ *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if via[0].Method != http.MethodGet && via[0].Method != http.MethodHead {
		return http.ErrUseLastResponse
	}
	return nil
}

func ClientFromContext(ctx context.Context) (*gitea.Client, error) {
	token, ok := ctx.Value(mcpContext.TokenContextKey).(string)
	if !ok {
		token = flag.Token
	}
	return NewClient(token)
}
