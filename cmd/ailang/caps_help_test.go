package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/types"
)

// Every effect the runtime can execute must be discoverable from the --caps help
// text. Four hand-typed copies of this list drifted apart and none of them named
// Process, so agents grepping `ailang --help` concluded the effect did not exist.
func TestCapsList_CoversEveryRegisteredEffect(t *testing.T) {
	listed := map[string]bool{}
	for _, c := range strings.Split(CapsList, ",") {
		listed[c] = true
	}
	for name := range effects.Registry {
		if name == "Debug" {
			continue // ghost effect: always granted, never passed to --caps
		}
		if !listed[name] {
			t.Errorf("effect %q is in effects.Registry but missing from CapsList (help text) — add it", name)
		}
	}
	for _, must := range []string{"IO", "FS", "Net", "Env", "Process", "AI"} {
		if !listed[must] {
			t.Errorf("CapsList must name %q", must)
		}
	}
}

// Every documented capability is a known effect: help must not advertise a
// name the type system would reject.
func TestCapsList_EveryEntryIsAKnownEffect(t *testing.T) {
	for _, c := range strings.Split(CapsList, ",") {
		if !types.IsKnownEffect(c) {
			t.Errorf("CapsList names %q, which types.IsKnownEffect rejects", c)
		}
	}
}

// #1116: `--caps NOPE,IO` used to run silently. An unknown name is an error
// naming the entry, the closest valid name, and the full valid list.
func TestGrantCapabilities_RejectsUnknownNames(t *testing.T) {
	cases := []struct {
		caps     string
		wantErr  bool
		contains []string
	}{
		{"IO,FS", false, nil},
		{" IO , Net ", false, nil},
		{"", false, nil},
		{"Rand", false, nil}, // known effect even though --caps help does not list it (auto-grant passes it)
		{"NOPE,IO", true, []string{`"NOPE"`, CapsList}},
		{"IO,FS,Nett", true, []string{`"Nett"`, "did you mean Net"}},
		{"io", true, []string{`"io"`, "did you mean IO"}},
		{"Procss", true, []string{"did you mean Process"}}, // edit distance 1
	}
	for _, tc := range cases {
		effCtx := effects.NewEffContext(nil)
		err := grantCapabilities(effCtx, tc.caps)
		if !tc.wantErr {
			if err != nil {
				t.Errorf("grantCapabilities(%q): unexpected error %v", tc.caps, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("grantCapabilities(%q): want error, got nil", tc.caps)
			continue
		}
		for _, want := range tc.contains {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("grantCapabilities(%q): error %q lacks %q", tc.caps, err, want)
			}
		}
	}
	// An accepted list grants exactly what it names.
	effCtx := effects.NewEffContext(nil)
	if err := grantCapabilities(effCtx, "IO,Process"); err != nil {
		t.Fatal(err)
	}
	if !effCtx.HasCap("IO") || !effCtx.HasCap("Process") || effCtx.HasCap("Net") {
		t.Errorf("grants wrong: IO=%v Process=%v Net=%v", effCtx.HasCap("IO"), effCtx.HasCap("Process"), effCtx.HasCap("Net"))
	}
	// A rejected list grants NOTHING — no partial grant before the error.
	effCtx = effects.NewEffContext(nil)
	_ = grantCapabilities(effCtx, "IO,NOPE")
	if effCtx.HasCap("IO") {
		t.Error("IO was granted despite the list being rejected")
	}
}

// Through the real binary: the typo is a non-zero exit at invocation, not a
// program that runs with a capability quietly missing.
func TestRun_UnknownCapabilityIsFatal(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join(t.TempDir(), "p.ail")
	if err := os.WriteFile(src, []byte("module p\nimport std/io (println)\nexport func main() -> () ! {IO} { println(\"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runAilangBin(t, bin, "run", "--relax-modules", "--caps", "IO,FS,Nett", "--entry", "main", src)
	if code == 0 {
		t.Fatalf("--caps IO,FS,Nett exited 0\nstdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "\nx\n") || strings.HasPrefix(stdout, "x\n") {
		t.Errorf("program ran despite the rejected --caps\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, `unknown capability "Nett"`) || !strings.Contains(stderr, "did you mean Net") {
		t.Errorf("stderr should name the entry and the suggestion:\n%s", stderr)
	}
}
