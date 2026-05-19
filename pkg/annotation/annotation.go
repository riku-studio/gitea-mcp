package annotation

import "github.com/mark3labs/mcp-go/mcp"

func ReadOnly(title string) mcp.ToolAnnotation {
	t := true
	return mcp.ToolAnnotation{Title: title, ReadOnlyHint: &t}
}

func Write(title string) mcp.ToolAnnotation {
	f := false
	return mcp.ToolAnnotation{Title: title, ReadOnlyHint: &f}
}

func Destructive(title string) mcp.ToolAnnotation {
	f, t := false, true
	return mcp.ToolAnnotation{Title: title, ReadOnlyHint: &f, DestructiveHint: &t}
}
