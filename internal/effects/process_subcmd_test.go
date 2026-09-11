//go:build !js

package effects

import (
	"reflect"
	"strings"
	"testing"
)

// M-PROCESS-SUBCMD M1: `cmd:sub[:sub…]` entries narrow a binary to subcommand chains.
func TestProcessContext_ResolveAllowlist_Subcommands(t *testing.T) {
	cases := []struct {
		name    string
		list    string
		wantCmd []string              // keys expected in Allowlist
		wantSub map[string][][]string // expected Subcommands (nil entry = unrestricted)
	}{
		{"single chain", "echo:status",
			[]string{"echo"}, map[string][][]string{"echo": {{"status"}}}},
		{"two chains same binary", "echo:pr:list,echo:pr:view",
			[]string{"echo"}, map[string][][]string{"echo": {{"pr", "list"}, {"pr", "view"}}}},
		{"bare wins over narrowed", "echo,echo:status",
			[]string{"echo"}, map[string][][]string{}},
		{"narrowed then bare also bare wins", "echo:status,echo",
			[]string{"echo"}, map[string][][]string{}},
		{"star is bare", "echo:*",
			[]string{"echo"}, map[string][][]string{}},
		{"absolute path keeps path as key", "/bin/echo:status",
			[]string{"/bin/echo"}, map[string][][]string{"/bin/echo": {{"status"}}}},
		{"mixed bare and narrowed", "date,echo:hello",
			[]string{"date", "echo"}, map[string][][]string{"echo": {{"hello"}}}},
		{"legacy list unchanged", "echo,date",
			[]string{"echo", "date"}, map[string][][]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := NewProcessContext()
			if err := pc.ResolveAllowlist(tc.list); err != nil {
				t.Fatalf("ResolveAllowlist(%q): %v", tc.list, err)
			}
			for _, k := range tc.wantCmd {
				if _, ok := pc.Allowlist[k]; !ok {
					t.Errorf("Allowlist missing %q (have %v)", k, pc.Allowlist)
				}
			}
			if len(pc.Allowlist) != len(tc.wantCmd) {
				t.Errorf("Allowlist has %d keys, want %d: %v", len(pc.Allowlist), len(tc.wantCmd), pc.Allowlist)
			}
			got := pc.Subcommands
			if got == nil {
				got = map[string][][]string{}
			}
			if !reflect.DeepEqual(got, tc.wantSub) {
				t.Errorf("Subcommands = %v, want %v", got, tc.wantSub)
			}
		})
	}
}

// An empty subcommand segment is a startup error naming the entry — never a silent allow.
func TestProcessContext_ResolveAllowlist_MalformedSubcommand(t *testing.T) {
	for _, bad := range []string{"echo:", "echo::status", ":status", "echo:status:"} {
		pc := NewProcessContext()
		err := pc.ResolveAllowlist("date," + bad)
		if err == nil {
			t.Errorf("ResolveAllowlist(%q): want error, got nil (Allowlist=%v Subcommands=%v)", bad, pc.Allowlist, pc.Subcommands)
			continue
		}
		if !strings.Contains(err.Error(), bad) {
			t.Errorf("ResolveAllowlist(%q): error %q does not name the entry", bad, err)
		}
	}
}
