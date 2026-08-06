package to

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTextResult(t *testing.T) {
	result, err := TextResult(map[string]any{"name": "gitea"})
	if err != nil {
		t.Fatalf("TextResult() error = %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(result.Content))
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0] type = %T, want *mcp.TextContent", result.Content[0])
	}
	if content.Text != `{"name":"gitea"}` {
		t.Errorf("Text = %q, want JSON object", content.Text)
	}
}

func TestErrorResult(t *testing.T) {
	want := errors.New("failed")
	result, err := ErrorResult(want)
	if result != nil || !errors.Is(err, want) {
		t.Errorf("ErrorResult() = (%#v, %v), want (nil, %v)", result, err, want)
	}
}
