package operation

import (
	"gitea.com/gitea/gitea-mcp/operation/repo"
	"gitea.com/gitea/gitea-mcp/pkg/tool"
)

// riku-studio fork: tools removed from the upstream surface.
//
// create_or_update_file requires the full file contents in tool arguments,
// which is impractical for large changes; the local git tools (git_add,
// git_commit, git_push) replace it. Filtering here instead of editing
// operation/repo keeps the upstream package untouched.
var disabledTools = map[string]struct{}{
	repo.CreateOrUpdateFileToolName: {},
}

// fileTool is repo.FileTool without the disabled tools, under the same scope.
var fileTool = withoutDisabled(repo.FileTool)

func withoutDisabled(src *tool.Tool) *tool.Tool {
	dst := tool.New(src.Scope())
	for _, st := range src.ReadTools() {
		if _, off := disabledTools[st.Tool.Name]; !off {
			dst.RegisterRead(st)
		}
	}
	for _, st := range src.WriteTools() {
		if _, off := disabledTools[st.Tool.Name]; !off {
			dst.RegisterWrite(st)
		}
	}
	return dst
}
