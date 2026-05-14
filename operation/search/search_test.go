package search

import (
	"slices"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestSearchToolsRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		tool     mcp.Tool
		required []string
	}{
		{
			name:     "search_users",
			tool:     SearchUsersTool,
			required: []string{"query"},
		},
		{
			name:     "search_org_teams",
			tool:     SearOrgTeamsTool,
			required: []string{"org", "query"},
		},
		{
			name:     "search_repos",
			tool:     SearchReposTool,
			required: []string{"query"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, field := range tt.required {
				if !slices.Contains(tt.tool.InputSchema.Required, field) {
					t.Errorf("tool %s: expected %q to be required, got required=%v", tt.name, field, tt.tool.InputSchema.Required)
				}
			}
		})
	}
}
