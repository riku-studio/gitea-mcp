package gitea

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHTTPTransportPinsRequestsAndRedirectsToUnixSocket(t *testing.T) {
	t.Chdir(t.TempDir())
	listener, err := net.Listen("unix", "gitea.sock")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "unix" {
			http.Redirect(w, r, "http://example.com/api/v1/version", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, r.Host+r.URL.Path)
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	transport := newHTTPTransport(false, listener.Addr().String())
	if transport.Proxy != nil {
		t.Error("Proxy is set, want nil so proxy settings are ignored")
	}
	resp, err := (&http.Client{Transport: transport}).Get("http://unix/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "example.com/api/v1/version" {
		t.Errorf("body = %q, want the redirect served over the socket", body)
	}
}
