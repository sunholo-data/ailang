package main

import (
	"sort"
	"strings"
	"testing"
)

// preS5LanguageCommands is the membership list of the isLanguageCommand switch
// as it stood before the dispatch table (cmd/ailang/main.go, M-V1-SIMPLIFY-S1
// M6). It is copied here verbatim so the table cannot quietly reclassify a
// command: moving a name between the two classes changes what runs at startup
// (an observatory DB stat and a git probe) for every invocation of it.
var preS5LanguageCommands = []string{
	"version", "run", "repl", "test", "watch", "check", "fmt", "ai-check",
	"iface", "internal-dump-iface", "verify", "compile", "disasm", "debug",
	"prompt", "devtools-prompt", "docs", "builtins", "examples", "lsp",
	"init", "select-best", "ast-edit", "replay", "export-training",
}

// s5AddedLanguageRoutes are the language routes that did not exist before S5,
// so preS5LanguageCommands cannot contain them. Each is listed with its reason,
// because adding a route here is how a command would quietly stop running the
// startup probes.
var s5AddedLanguageRoutes = []string{
	// M2: `ailang help [command]`. Printing help must not stat the
	// observatory DB or shell out to git, exactly as `version` must not.
	"help",
	// M2: the alias spelling of `internal-dump-iface`, named by the Phase 3
	// item-2 dev list. Same row, same contract.
	"dump-iface",
}

// TestTable_LanguageRoutesMatchPreS5List compares ROUTES, not names, because
// S5 M2 made `ai-check` an alias of `check` rather than a row of its own. The
// contract is about what runs at startup for a given spelling, and a spelling
// is a route — so comparing names would have silently dropped `ai-check` from
// the very list that exists to stop it being reclassified.
func TestTable_LanguageRoutesMatchPreS5List(t *testing.T) {
	var got []string
	for i := range allCommands {
		c := &allCommands[i]
		if !c.Language {
			continue
		}
		got = append(got, c.Name)
		got = append(got, c.Aliases...)
	}
	sort.Strings(got)

	want := append([]string(nil), preS5LanguageCommands...)
	want = append(want, s5AddedLanguageRoutes...)
	sort.Strings(want)

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("language routes drifted from the pre-S5 isLanguageCommand list\n got: %v\nwant: %v", got, want)
	}
}

// TestTable_GroupAndLanguageAreIndependent pins the separation S5 M2 made:
// where a command is FILED (Group) and whether it is quiet at startup
// (Language) are different questions. M1 had them on one field, which is why
// filing `disasm` under `dev` would otherwise have turned its probes on.
func TestTable_GroupAndLanguageAreIndependent(t *testing.T) {
	var langInDev, langVisible int
	for i := range allCommands {
		c := &allCommands[i]
		if !c.Language {
			continue
		}
		switch c.Group {
		case groupDev:
			langInDev++
		case "":
			langVisible++
		}
	}
	// Both populations must be non-empty, or the two fields are still
	// effectively one and this test proves nothing.
	if langInDev == 0 {
		t.Error("no language command is filed under dev — Group and Language have collapsed back into one field")
	}
	if langVisible == 0 {
		t.Error("no language command is visible at the top level")
	}
	for i := range allCommands {
		c := &allCommands[i]
		if c.Language && c.Group == groupOps {
			t.Errorf("%s is a language command filed under ops", c.Name)
		}
	}
}

func TestTable_RowsAreWellFormed(t *testing.T) {
	for _, c := range allCommands {
		if c.Name == "" {
			t.Errorf("command with empty Name: %+v", c)
			continue
		}
		if c.Run == nil {
			t.Errorf("%s: nil Run", c.Name)
		}
		if strings.TrimSpace(c.Summary) == "" {
			t.Errorf("%s: empty Summary — it is what `ailang --help` prints", c.Name)
		}
	}
}

func TestTable_EveryRouteResolvesToItsRow(t *testing.T) {
	for i := range allCommands {
		c := &allCommands[i]
		for _, route := range append([]string{c.Name}, c.Aliases...) {
			got, ok := lookupCommand(route)
			if !ok {
				t.Errorf("route %q does not resolve", route)
				continue
			}
			if got.Name != c.Name {
				t.Errorf("route %q resolves to %q, want %q", route, got.Name, c.Name)
			}
		}
	}
}

// TestTable_HelpFallbackNamesExist keeps the measured set honest: a rename that
// leaves a stale entry behind would silently stop answering --help for it.
func TestTable_HelpFallbackNamesExist(t *testing.T) {
	for name := range helpFallbackCommands {
		c, ok := lookupCommand(name)
		if !ok {
			t.Errorf("helpFallbackCommands names %q, which is not in the table", name)
			continue
		}
		if c.Name != name {
			t.Errorf("helpFallbackCommands names the alias %q; use the canonical name %q", name, c.Name)
		}
	}
}

func TestTable_PkgSubcommandsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, sub := range pkgSubcommands() {
		if sub.Run == nil {
			t.Errorf("pkg %s: nil Run", sub.Name)
		}
		if strings.TrimSpace(sub.Summary) == "" {
			t.Errorf("pkg %s: empty Summary", sub.Name)
		}
		if seen[sub.Name] {
			t.Errorf("pkg %s: duplicate verb", sub.Name)
		}
		seen[sub.Name] = true
	}
	for name := range pkgHelpFallback {
		if !seen[name] {
			t.Errorf("pkgHelpFallback names %q, which is not a pkg verb", name)
		}
	}
}

// TestProbes_LanguageCommandsRunNone counts the two startup probes for every
// route in the table. It is the contract S1 M6 established: a language command
// opens no observatory DB and runs no git probe. The count is taken from a
// FAKE, not from reading main.go, so re-adding a probe to the platform path
// keeps passing and moving one into the language path fails.
func TestProbes_LanguageCommandsRunNone(t *testing.T) {
	realObs, realStale := probeObservatoryHealth, probeStaleBinary
	t.Cleanup(func() { probeObservatoryHealth, probeStaleBinary = realObs, realStale })

	var obsCalls, staleCalls int
	probeObservatoryHealth = func() { obsCalls++ }
	probeStaleBinary = func() { staleCalls++ }

	count := func(args []string) (int, int) {
		obsCalls, staleCalls = 0, 0
		runStale := platformStartup(args)
		runStale()
		return obsCalls, staleCalls
	}

	for i := range allCommands {
		c := &allCommands[i]
		for _, route := range append([]string{c.Name}, c.Aliases...) {
			obs, stale := count([]string{route})
			if c.Language {
				if obs != 0 || stale != 0 {
					t.Errorf("language command %q ran probes: observatory=%d stale=%d, want 0/0", route, obs, stale)
				}
				continue
			}
			if obs < 1 || stale < 1 {
				t.Errorf("platform command %q ran probes: observatory=%d stale=%d, want >=1/>=1", route, obs, stale)
			}
		}
	}

	// Controls: no command at all runs neither; an unknown name keeps the
	// probes, exactly as the pre-table `!isLanguageCommand(...)` did.
	if obs, stale := count(nil); obs != 0 || stale != 0 {
		t.Errorf("no arguments ran probes: observatory=%d stale=%d, want 0/0", obs, stale)
	}
	if obs, stale := count([]string{"definitely-not-a-command"}); obs != 1 || stale != 1 {
		t.Errorf("unknown command ran probes: observatory=%d stale=%d, want 1/1", obs, stale)
	}
}

func TestSuggestCommand(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"chian", "chains"}, // a transposition (1) + insert (1) = 2; Levenshtein would tie it with "bin" and "check" at 3
		{"bim", "bin"},
		{"mesages", "messages"},
		{"evl-suite", "eval-suite"},
		{"cheque", "check"},  // 3 edits, still worth offering
		{"zzzzzzzzzzzz", ""}, // nothing is within three edits
		{"", ""},
	}
	for _, tc := range cases {
		if got := suggestCommand(tc.in); got != tc.want {
			t.Errorf("suggestCommand(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The generated-help coverage test that lived here moved to
// commands_groups_test.go (TestGroups_HelpListsEveryVisibleMember) when S5 M2
// introduced groups: "every non-hidden row appears in `ailang --help`" stopped
// being true and stopped being the goal. The replacement checks the top level
// AND each group's own help, which is the property that now matters.
