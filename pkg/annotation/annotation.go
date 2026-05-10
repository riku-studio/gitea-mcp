// Package annotation provides shared MCP tool annotation helpers.
package annotation

import "github.com/mark3labs/mcp-go/mcp"

// ReadOnly returns a ToolAnnotation for read-only tools.
func ReadOnly(title string) mcp.ToolAnnotation {
	t := true
	return mcp.ToolAnnotation{
		Title:        title,
		ReadOnlyHint: &t,
	}
}

// Write returns a ToolAnnotation for write tools.
func Write(title string) mcp.ToolAnnotation {
	f := false
	return mcp.ToolAnnotation{
		Title:        title,
		ReadOnlyHint: &f,
	}
}

// Destructive returns a ToolAnnotation for destructive write tools.
func Destructive(title string) mcp.ToolAnnotation {
	f, t := false, true
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    &f,
		DestructiveHint: &t,
	}
}
