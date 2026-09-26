package types

import (
	"strings"
	"testing"
)

// M-DX-SELF-DISCOVERY: builtinHint turns an unknown `_x` name into a one-step
// self-discovery (inventory + rebuild) instead of a bare error that invites
// "fixing" std/*.ail. See M-FS-RENAME downstream, 2026-08-28.
func TestBuiltinHint(t *testing.T) {
	tests := []struct {
		name      string
		wantSub   string
		wantEmpty bool
	}{
		{name: "_fs_rename", wantSub: "ailang builtins list"},
		{name: "_fs_renam", wantSub: "ailang builtins list"},
		{name: "_invented_builtin", wantSub: "make quick-install"},
		{name: "renameFile", wantEmpty: true}, // stdlib exports are unprefixed
		{name: "x", wantEmpty: true},
		{name: "_", wantSub: "builtin naming convention"},
	}
	for _, tt := range tests {
		got := builtinHint(tt.name)
		if tt.wantEmpty {
			if got != "" {
				t.Errorf("builtinHint(%q) = %q, want empty", tt.name, got)
			}
			continue
		}
		if !strings.Contains(got, tt.wantSub) {
			t.Errorf("builtinHint(%q) = %q, want substring %q", tt.name, got, tt.wantSub)
		}
	}
}
