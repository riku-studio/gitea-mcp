package operation

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// toolTableRow matches a row of the "Available Tools" table in the README
// files, capturing the tool name and the access cell, e.g.
// "| get_me | User | Read | Get the current authenticated user |".
var toolTableRow = regexp.MustCompile(`^\|\s*([a-z_]+)\s*\|[^|]*\|\s*(\S+)\s*\|`)

// readmeAccessLabels maps each README to the access-column labels it uses.
var readmeAccessLabels = map[string]map[string]string{
	"../README.md":       {"Read": "read", "Write": "write"},
	"../README.zh-cn.md": {"读取": "read", "写入": "write"},
	"../README.zh-tw.md": {"讀取": "read", "寫入": "write"},
}

// TestReadmeToolTables ensures the tool tables in the README files stay in sync
// with the registered tools, in both directions and for every translation.
// The tables listed tools that no longer existed for several releases before
// anyone noticed.
func TestReadmeToolTables(t *testing.T) {
	registered := map[string]string{}
	for _, d := range domainTools {
		for _, st := range d.ReadTools() {
			registered[st.Tool.Name] = "read"
		}
		for _, st := range d.WriteTools() {
			registered[st.Tool.Name] = "write"
		}
	}

	for path, labels := range readmeAccessLabels {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			documented := map[string]string{}
			for line := range strings.SplitSeq(string(content), "\n") {
				if match := toolTableRow.FindStringSubmatch(line); match != nil {
					documented[match[1]] = labels[match[2]]
				}
			}

			for _, name := range slices.Sorted(maps.Keys(registered)) {
				access, ok := documented[name]
				switch {
				case !ok:
					t.Errorf("tool %q is registered but missing from the tool table", name)
				case access != registered[name]:
					t.Errorf("tool %q is documented with %q access, want %q", name, access, registered[name])
				}
			}
			for _, name := range slices.Sorted(maps.Keys(documented)) {
				if _, ok := registered[name]; !ok {
					t.Errorf("tool %q is in the tool table but is not registered", name)
				}
			}
		})
	}
}
