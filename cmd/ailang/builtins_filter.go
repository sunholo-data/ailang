package main

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/builtins"
)

// filterBuiltinSpecs narrows `ailang builtins list` (#1551): module keeps
// only builtins of exactly that module (std/fs); query keeps those whose
// name, module or description contains it, case-insensitively. Empty means
// no filter. The narrowed --json listing is what fits under a policy-tool
// output cap.
func filterBuiltinSpecs(specs map[string]*builtins.BuiltinSpec, module, query string) map[string]*builtins.BuiltinSpec {
	if module == "" && query == "" {
		return specs
	}
	q := strings.ToLower(query)
	out := make(map[string]*builtins.BuiltinSpec)
	for name, s := range specs {
		if module != "" && s.Module != module {
			continue
		}
		if q != "" {
			hay := name + " " + s.Module
			if s.Metadata != nil {
				hay += " " + s.Metadata.Description
			}
			if !strings.Contains(strings.ToLower(hay), q) {
				continue
			}
		}
		out[name] = s
	}
	return out
}

// restrictGroups drops from registry-wide groups (builtins.GroupByModule /
// GroupByEffect) every name the filter removed, and any group left empty, so
// grouped listings never index a spec that is not in specs.
func restrictGroups(grouped map[string][]string, specs map[string]*builtins.BuiltinSpec) map[string][]string {
	out := make(map[string][]string, len(grouped))
	for key, names := range grouped {
		var kept []string
		for _, n := range names {
			if _, ok := specs[n]; ok {
				kept = append(kept, n)
			}
		}
		if len(kept) > 0 {
			out[key] = kept
		}
	}
	return out
}
