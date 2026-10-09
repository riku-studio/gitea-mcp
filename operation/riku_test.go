package operation

import "testing"

func TestDisabledToolsNotRegistered(t *testing.T) {
	for _, d := range domainTools {
		for _, st := range append(d.ReadTools(), d.WriteTools()...) {
			if _, off := disabledTools[st.Tool.Name]; off {
				t.Errorf("disabled tool %q is registered in scope %q", st.Tool.Name, d.Scope())
			}
		}
	}
}

func TestGitToolsRegistered(t *testing.T) {
	want := []string{"git_status", "git_diff", "git_add", "git_commit", "git_push",
		"git_pull", "git_fetch", "git_branch", "git_checkout", "git_log"}
	got := map[string]bool{}
	for _, d := range domainTools {
		for _, st := range append(d.ReadTools(), d.WriteTools()...) {
			got[st.Tool.Name] = true
		}
	}
	for _, n := range want {
		if !got[n] {
			t.Errorf("tool %q is not registered", n)
		}
	}
	if !got["get_file_contents"] || !got["delete_file"] {
		t.Error("other file-scope tools must remain registered")
	}
}
