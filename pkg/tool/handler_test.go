package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func callTool(handler Handler, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	serverTool := ServerTool{Tool: &mcp.Tool{Name: "example"}, Handler: handler}
	return serverTool.MCPHandler()(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: arguments},
	})
}

func captureArguments(into *map[string]any) Handler {
	return func(_ context.Context, arguments map[string]any) (*mcp.CallToolResult, error) {
		*into = arguments
		return &mcp.CallToolResult{}, nil
	}
}

func TestMCPHandler(t *testing.T) {
	var got map[string]any
	result, err := callTool(captureArguments(&got), json.RawMessage(`{"count":2,"nested":{"enabled":true}}`))
	if err != nil {
		t.Fatalf("MCPHandler() error = %v", err)
	}
	if result == nil {
		t.Fatal("MCPHandler() result is nil")
	}
	if got["count"] != float64(2) {
		t.Errorf("count type/value = %T(%v), want float64(2)", got["count"], got["count"])
	}
}

func TestMCPHandlerRejectsInvalidArguments(t *testing.T) {
	called := false
	handler := func(context.Context, map[string]any) (*mcp.CallToolResult, error) {
		called = true
		return &mcp.CallToolResult{}, nil
	}

	for _, arguments := range []json.RawMessage{json.RawMessage(`[]`), json.RawMessage(`"text"`), json.RawMessage(`{"broken"`)} {
		_, err := callTool(handler, arguments)
		assertProtocolErrorCode(t, err, jsonrpc.CodeInvalidParams)
	}
	if called {
		t.Fatal("handler was called with invalid arguments")
	}
}

// Tools without parameters are callable with an omitted or null "arguments",
// which is what clients send and what mcp-go accepted before the SDK migration.
func TestMCPHandlerAcceptsAbsentArguments(t *testing.T) {
	for _, arguments := range []json.RawMessage{nil, json.RawMessage(`null`)} {
		var got map[string]any
		if _, err := callTool(captureArguments(&got), arguments); err != nil {
			t.Fatalf("MCPHandler() with arguments %s error = %v", arguments, err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("arguments = %#v, want an empty map", got)
		}
	}
}

func TestMCPHandlerErrorClassification(t *testing.T) {
	for _, test := range []struct {
		name     string
		handler  Handler
		wantCode int64
	}{
		{
			name:     "server error",
			handler:  func(context.Context, map[string]any) (*mcp.CallToolResult, error) { return nil, errors.New("failed") },
			wantCode: jsonrpc.CodeInternalError,
		},
		{
			name: "protocol error",
			handler: func(context.Context, map[string]any) (*mcp.CallToolResult, error) {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "failed"}
			},
			wantCode: jsonrpc.CodeInvalidParams,
		},
		{
			name:     "panic",
			handler:  func(context.Context, map[string]any) (*mcp.CallToolResult, error) { panic("failed") },
			wantCode: jsonrpc.CodeInternalError,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := callTool(test.handler, nil)
			if result != nil {
				t.Errorf("result = %#v, want nil", result)
			}
			assertProtocolErrorCode(t, err, test.wantCode)
		})
	}
}

func assertProtocolErrorCode(t *testing.T, err error, want int64) {
	t.Helper()
	var protocolErr *jsonrpc.Error
	if !errors.As(err, &protocolErr) {
		t.Fatalf("error = %v, want *jsonrpc.Error", err)
	}
	if protocolErr.Code != want {
		t.Errorf("error code = %d, want %d", protocolErr.Code, want)
	}
}
