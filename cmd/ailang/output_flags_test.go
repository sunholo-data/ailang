package main

import (
	"flag"
	"io"
	"testing"
)

// newQuietFlagSet builds a ContinueOnError FlagSet that does not print usage,
// so a deliberate parse failure below does not pollute the test log.
func newQuietFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// TestResolveTestFormat pins `ailang test`'s two output-format spellings.
//
// The load-bearing case is "legacy --format json": fifteen SHIPPED, FROZEN
// teaching prompts tell agents to type it, and check-prompt-freeze forbids
// editing them, so it must keep resolving to exactly "json" forever.
//
// Red mutation (measured): in resolveTestFormat, return formatFlag before the
// jsonFlag branch — i.e. drop the `if jsonFlag` clause. It compiles, and the
// two "--json" cases below fail with got="human", want="json".
func TestResolveTestFormat(t *testing.T) {
	cases := []struct {
		name   string
		json   bool
		format string
		want   string
	}{
		{"default is human", false, "human", "human"},
		{"legacy --format json still json", false, "json", "json"},
		{"canonical --json", true, "human", "json"},
		{"--json wins over an explicit --format human", true, "human", "json"},
		{"both spellings agree", true, "json", "json"},
		{"an unknown --format value is passed through untouched", false, "tap", "tap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveTestFormat(tc.json, tc.format); got != tc.want {
				t.Errorf("resolveTestFormat(%v, %q) = %q, want %q", tc.json, tc.format, got, tc.want)
			}
		})
	}
}

// TestJSONOnlyOutputFlags pins the eval-paired / eval-censored-pairs pair.
//
// The default MUST stay compact: tools/launchd/nightly-eval.sh reads
// eval-paired's stdout, and indenting it by default would change bytes a live
// caller parses.
//
// Red mutation (measured): in registerJSONOnlyOutputFlags, return
// `func() bool { return *pretty }` — dropping the `&& !*jsonOut` clause. It
// compiles, and "both: --json wins" fails with indent=true, want false.
func TestJSONOnlyOutputFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"default is compact", nil, false},
		{"--pretty indents", []string{"--pretty"}, true},
		{"--json is the explicit spelling of the default", []string{"--json"}, false},
		{"both: --json wins, so neither flag is ignored", []string{"--pretty", "--json"}, false},
		{"order does not matter", []string{"--json", "--pretty"}, false},
		{"--pretty=false", []string{"--pretty=false"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newQuietFlagSet("eval-paired")
			indent := registerJSONOnlyOutputFlags(fs)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("Parse(%v): %v", tc.args, err)
			}
			if got := indent(); got != tc.want {
				t.Errorf("indent() with %v = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

// TestJSONOnlyOutputFlagsNamesBothSpellings proves BOTH names are registered —
// a helper that registered only --pretty would pass the table above, because
// every --json case there expects the default.
func TestJSONOnlyOutputFlagsNamesBothSpellings(t *testing.T) {
	fs := newQuietFlagSet("eval-censored-pairs")
	registerJSONOnlyOutputFlags(fs)
	for _, name := range []string{"json", "pretty"} {
		if fs.Lookup(name) == nil {
			t.Errorf("flag --%s is not registered", name)
		}
	}
	// Control: a name the helper must NOT invent.
	if fs.Lookup("format") != nil {
		t.Error("--format must not be registered by the JSON-only helper")
	}
}

// TestRegisterJSONFlag pins the canonical spelling and its default. A command
// whose --json defaulted to true would emit machine output to a human.
func TestRegisterJSONFlag(t *testing.T) {
	fs := newQuietFlagSet("test")
	jsonFlag := registerJSONFlag(fs, "Machine-readable JSON output")
	if fs.Lookup(flagJSON) == nil {
		t.Fatalf("flag --%s is not registered", flagJSON)
	}
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("Parse(nil): %v", err)
	}
	if *jsonFlag {
		t.Error("--json must default to false")
	}
}

// TestAliasStringFlag covers `--model` / `--models` on eval-suite.
//
// Red mutation (measured): change aliasStringFlag's default from *target to ""
// — it compiles, and "neither passed keeps the canonical default" fails with
// got="" instead of "dev".
func TestAliasStringFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"neither passed keeps the canonical default", nil, "dev"},
		{"canonical spelling", []string{"--models", "a,b"}, "a,b"},
		{"alias spelling", []string{"--model", "a,b"}, "a,b"},
		{"alias with an equals sign", []string{"--model=a,b"}, "a,b"},
		{"single-dash alias", []string{"-model", "a,b"}, "a,b"},
		{"both passed: last wins, as a repeated flag already does", []string{"--models", "a", "--model", "b"}, "b"},
		{"both passed, other order", []string{"--model", "b", "--models", "a"}, "a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newQuietFlagSet("eval-suite")
			models := fs.String("models", "dev", "canonical")
			aliasStringFlag(fs, models, "model", "Alias of --models")
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("Parse(%v): %v", tc.args, err)
			}
			if *models != tc.want {
				t.Errorf("models with %v = %q, want %q", tc.args, *models, tc.want)
			}
		})
	}
}

// TestDryRunDefaultsAct is the M5 --dry-run audit, frozen as a test.
//
// The family rule is "--dry-run means the DEFAULT acts": every site registers
// it as a bool defaulting to false, so omitting it performs the action. A new
// command that registers `--dry-run` defaulting to true — or a `--no-dry-run`
// / `--apply` / `--execute` inversion, which would mean the default does NOT
// act — fails here rather than being discovered by an operator whose command
// silently did nothing.
//
// Red mutation (measured): flip any one site to `fs.Bool("dry-run", true, ...)`
// — it compiles, and this test fails naming that file and line.
func TestDryRunDefaultsAct(t *testing.T) {
	sites := dryRunRegistrations(t)
	if len(sites) < 14 {
		t.Fatalf("found %d --dry-run registrations, expected at least 14 — "+
			"the scanner has stopped seeing them, not the sites disappeared", len(sites))
	}
	for _, s := range sites {
		if s.defaultValue != "false" {
			t.Errorf("%s: --dry-run default is %s, want false — "+
				"a --dry-run whose default is true means the command does NOT act by default, "+
				"which is a behaviour change the owner must make deliberately", s.where, s.defaultValue)
		}
	}
}

// TestNoDryRunInversionFlags is the negative half of the audit: an inverted
// spelling is how "default acts" gets broken without touching any --dry-run
// default at all.
func TestNoDryRunInversionFlags(t *testing.T) {
	for _, name := range []string{"no-dry-run", "not-dry-run", "wet-run"} {
		if sites := flagRegistrations(t, name); len(sites) > 0 {
			t.Errorf("flag --%s is registered at %v — "+
				"the --dry-run family has exactly one spelling and the default acts", name, sites)
		}
	}
	// Control: the scanner finds a flag that IS registered, so a silent
	// zero above cannot be mistaken for a pass.
	if sites := flagRegistrations(t, "dry-run"); len(sites) == 0 {
		t.Fatal("scanner control failed: found no --dry-run registrations at all")
	}
}
