package main

import (
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"
)

// wantVisibleTopLevel is the visible top level Phase 3 item 2 specifies, and
// the gate this whole phase exists to close: `commands_top_level` 89 -> at most
// 20. It is written out in full rather than counted, because the count alone
// would pass while the wrong 17 commands were on screen.
var wantVisibleTopLevel = []string{
	"check", "docs", "eval", "examples", "fmt", "help", "iface", "init",
	"lsp", "pkg", "prompt", "repl", "run", "serve", "test", "verify", "version",
}

func TestGroups_VisibleTopLevelIsTheSpecifiedSet(t *testing.T) {
	var got []string
	for _, c := range visibleTopLevel() {
		got = append(got, c.Name)
	}
	want := append([]string(nil), wantVisibleTopLevel...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("visible top level drifted\n got: %v\nwant: %v", got, want)
	}
}

// TestGroups_VisibleTopLevelFitsTheGate states the gate itself. The set test
// above is stricter, but this one names the number the simplicity audit reads,
// so a failure says what it costs.
func TestGroups_VisibleTopLevelFitsTheGate(t *testing.T) {
	if n := len(visibleTopLevel()); n > 20 {
		t.Fatalf("%d visible top-level commands, gate is 20 (m-v1-simplification-program.md, commands_top_level)", n)
	}
}

// TestGroups_EveryRowIsFiledSomewhereReal catches the failure mode a typo
// produces: Group: "opss" would silently make a command unreachable through
// any group AND absent from the visible list.
func TestGroups_EveryRowIsFiledSomewhereReal(t *testing.T) {
	known := map[string]bool{"": true, groupPkg: true}
	for _, g := range dispatchGroups {
		known[g] = true
	}
	for i := range allCommands {
		c := &allCommands[i]
		if !known[c.Group] {
			t.Errorf("%s: unknown Group %q", c.Name, c.Group)
		}
	}
}

// TestGroups_HiddenRowsAreDeliberate pins the complete list of routes that
// appear in NO help at all. Hiding a command is how it stops being
// discoverable, so it should never happen by accident — and M1 found the
// opposite defect, `internal-dump-iface` showing up in `--help` for the first
// time because the generated list had no notion of "internal".
func TestGroups_HiddenRowsAreDeliberate(t *testing.T) {
	want := []string{
		// The two group drawers. They are reached by name, and the top-level
		// help points at them in its footer rather than spending two rows.
		"dev", "ops",
		// Internal: internal/pkg/iface_subprocess.go execs this to build a
		// package interface out-of-process. It was deliberately absent from
		// the pre-S5 hand-written help.
		"internal-dump-iface",
		// The bare registry verbs `ailang pkg <verb>` supersedes. Still
		// routed, because D1 says so; not listed, because there is now one
		// canonical spelling.
		"add", "install", "lock", "pkg-docs", "publish", "search", "tree", "unpublish",
		// M-V1-SIMPLIFY-S5 M3: folded into `chains`, whose namespaces
		// (`ailang chains trace|observatory|dashboard|eval`) are now the
		// canonical spelling. Each stays routed for one release — the
		// trace-debugger skill alone holds 58 references to `ailang trace ...`
		// — and stops being listed, which is what "deprecated for one release"
		// means in help. commands_folded_test.go asserts the same four rows
		// from the other direction, so removing one here is not enough to
		// unhide it silently.
		"dashboard", "eval-chains", "observatory", "trace",
	}
	var got []string
	for i := range allCommands {
		if allCommands[i].Hidden {
			got = append(got, allCommands[i].Name)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("hidden rows drifted\n got: %v\nwant: %v", got, want)
	}
}

// TestGroups_SubNameDerivation pins the one derived name in the design: an
// eval command's group spelling is its top-level name minus the `eval-`
// prefix, so `ailang eval suite` and `ailang eval-suite` are the same route
// and there is no second column to keep in step.
func TestGroups_SubNameDerivation(t *testing.T) {
	cases := []struct{ group, name, want string }{
		{groupEval, "eval-suite", "suite"},
		{groupEval, "eval-censored-pairs", "censored-pairs"},
		{groupEval, "browser-profile", "browser-profile"},
		{groupOps, "messages", "messages"},
		{groupDev, "micro-rag", "micro-rag"},
		// Not a prefix match: `develop` must not become `elop`.
		{groupDev, "development", "development"},
	}
	for _, tc := range cases {
		if got := groupSubName(tc.group, tc.name); got != tc.want {
			t.Errorf("groupSubName(%q, %q) = %q, want %q", tc.group, tc.name, got, tc.want)
		}
	}
}

// TestGroups_HelpListsEveryVisibleMember is the guard the hand-written help
// failed before M1 (15 commands the switch accepted were missing from it).
// Generated help cannot drift; this proves it for every group as well as the
// top level, and asserts hidden rows stay out.
func TestGroups_HelpListsEveryVisibleMember(t *testing.T) {
	var top bytes.Buffer
	renderCommandList(&top)
	topOut := top.String()
	for _, c := range visibleTopLevel() {
		if !strings.Contains(topOut, c.Name) {
			t.Errorf("visible command %q missing from `ailang --help`", c.Name)
		}
		for _, a := range c.Aliases {
			if !strings.Contains(topOut, a) {
				t.Errorf("alias %q of %q missing from `ailang --help`", a, c.Name)
			}
		}
	}
	for _, g := range []string{groupDev, groupOps} {
		if !strings.Contains(topOut, "ailang "+g+" --help") {
			t.Errorf("`ailang --help` does not point at the %q group, so it is undiscoverable", g)
		}
	}

	for _, group := range dispatchGroups {
		var buf bytes.Buffer
		printGroupHelp(&buf, group)
		out := buf.String()
		for _, c := range groupMembers(group) {
			sub := groupSubName(group, c.Name)
			if c.Hidden {
				if strings.Contains(out, "  "+sub+" ") {
					t.Errorf("hidden command %q is listed in `ailang %s --help`", c.Name, group)
				}
				continue
			}
			if !strings.Contains(out, sub) {
				t.Errorf("%q missing from `ailang %s --help`", sub, group)
			}
		}
		for _, extra := range groupExtraRows[group] {
			if !strings.Contains(out, extra[0]) {
				t.Errorf("extra row %q missing from `ailang %s --help`", extra[0], group)
			}
		}
	}
}

// TestGroups_TopLevelHelpIsOneScreen states the readable goal behind the gate:
// an agent reads `ailang --help` and sees the command list without paging. The
// bound is the command section only, which is what the gate counts.
func TestGroups_TopLevelHelpIsOneScreen(t *testing.T) {
	var buf bytes.Buffer
	renderCommandList(&buf)
	lines := strings.Count(strings.TrimSpace(buf.String()), "\n") + 1
	if lines > 30 {
		t.Errorf("the generated command section is %d lines; it was meant to fit one screen", lines)
	}
}

// TestGroups_GroupRouteIsByteIdenticalToTheTopLevelSpelling is the end-to-end
// form of the same property, and the one that actually caught bugs.
//
// The in-process test below proves os.Args is rewritten. It does NOT prove the
// group route behaves identically, because the binary has a SECOND argument
// source: the global flag.Args(), which `trace`, `chains`, `dashboard`,
// `axioms`, `editor`, `budget` and the rest of the noArgs family read instead.
// Measured before runAsName reset it too: `ailang ops trace status` exited 1
// where `ailang trace status` exited 0, and `ailang dev axioms --help` printed
// the entire axiom scorecard instead of its usage block. Both looked fine from
// the outside — wrong argument, not visible failure.
//
// Mutation: delete either rewrite from runAsName and this fails, naming the
// pair. The self-comparison and alias controls come first: without them a
// timestamp in the observatory retention log would make every pair "differ"
// (it did, on this harness's first run).
func TestGroups_GroupRouteIsByteIdenticalToTheTopLevelSpelling(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the binary ~80 times")
	}
	bin := cliTestBin(t)

	pairs := []struct {
		name     string
		topLevel []string
		grouped  []string
	}{
		// Controls FIRST. The same route twice, then a pre-existing alias:
		// if either reports a difference, the instrument is measuring noise
		// and every negative result below is meaningless.
		{"control/self", []string{"messages"}, []string{"messages"}},
		{"control/alias", []string{"messages"}, []string{"msg"}},

		// flag.Args() readers — the family the second rewrite exists for.
		{"ops/trace status", []string{"trace", "status"}, []string{"ops", "trace", "status"}},
		{"ops/dashboard", []string{"dashboard"}, []string{"ops", "dashboard"}},
		{"ops/budget", []string{"budget"}, []string{"ops", "budget"}},
		{"dev/axioms --help", []string{"axioms", "--help"}, []string{"dev", "axioms", "--help"}},
		{"dev/editor", []string{"editor"}, []string{"dev", "editor"}},

		// os.Args slicers.
		{"ops/messages", []string{"messages"}, []string{"ops", "messages"}},
		{"dev/micro-rag", []string{"micro-rag"}, []string{"dev", "micro-rag"}},
		{"dev/cache", []string{"cache"}, []string{"dev", "cache"}},

		// A language command reached through a group must stay quiet at
		// startup: no observatory line, no stale-binary warning. Measured
		// before invocationIsLanguage followed the group hop, `ailang dev
		// builtins` emitted two Observatory lines that `ailang builtins` did
		// not — so this pair is a probe-contract test as much as an argv one.
		{"dev/builtins", []string{"builtins"}, []string{"dev", "builtins"}},
		{"dev/policy-check", []string{"policy-check"}, []string{"dev", "policy-check"}},

		// eval, whose subcommand name is derived by stripping the prefix.
		{"eval/suite --help", []string{"eval-suite", "--help"}, []string{"eval", "suite", "--help"}},
		{"eval/summary", []string{"eval-summary"}, []string{"eval", "summary"}},
		{"eval/browser-profile", []string{"browser-profile"}, []string{"eval", "browser-profile"}},

		// The bare pkg verbs against their new canonical spelling.
		{"pkg/add", []string{"add"}, []string{"pkg", "add"}},
		{"pkg/docs", []string{"pkg-docs"}, []string{"pkg", "docs"}},
		{"pkg/lock", []string{"lock"}, []string{"pkg", "lock"}},
	}

	for _, p := range pairs {
		p := p
		t.Run(p.name, func(t *testing.T) {
			// ONE home and ONE working directory for both runs: `budget` and
			// `lock` print their CWD, so two different temp dirs would make
			// the same route differ from itself.
			home, work := t.TempDir(), t.TempDir()
			a := runCLIIn(t, bin, home, work, p.topLevel...)
			b := runCLIIn(t, bin, home, work, p.grouped...)
			if a.timedOut || b.timedOut {
				t.Fatalf("one of the invocations did not finish inside the bound")
			}
			if a.exitCode != b.exitCode {
				t.Errorf("exit code: `ailang %s` = %d, `ailang %s` = %d",
					strings.Join(p.topLevel, " "), a.exitCode,
					strings.Join(p.grouped, " "), b.exitCode)
			}
			if a.stdout != b.stdout {
				t.Errorf("stdout differs between `ailang %s` and `ailang %s`:\n--- top-level ---\n%s\n--- grouped ---\n%s",
					strings.Join(p.topLevel, " "), strings.Join(p.grouped, " "),
					firstLines(a.stdout, 8), firstLines(b.stdout, 8))
			}
			if a.stderr != b.stderr {
				t.Errorf("stderr differs between `ailang %s` and `ailang %s`:\n--- top-level ---\n%s\n--- grouped ---\n%s",
					strings.Join(p.topLevel, " "), strings.Join(p.grouped, " "),
					firstLines(a.stderr, 8), firstLines(b.stderr, 8))
			}
		})
	}
}

// TestGroups_LanguageCommandsStayQuietThroughTheirGroup states the probe half
// of that pair as a rule over the whole table, not a sample: for every
// language command, reaching it through its group must make the same startup
// decision as reaching it by name. Counted from the fake, so it cannot be
// satisfied by reading main.go.
func TestGroups_LanguageCommandsStayQuietThroughTheirGroup(t *testing.T) {
	realObs, realStale := probeObservatoryHealth, probeStaleBinary
	t.Cleanup(func() { probeObservatoryHealth, probeStaleBinary = realObs, realStale })

	var obsCalls, staleCalls int
	probeObservatoryHealth = func() { obsCalls++ }
	probeStaleBinary = func() { staleCalls++ }
	count := func(args []string) (int, int) {
		obsCalls, staleCalls = 0, 0
		platformStartup(args)()
		return obsCalls, staleCalls
	}

	var checked int
	for i := range allCommands {
		c := &allCommands[i]
		if c.Group == "" || !isGroupName(c.Group) {
			continue
		}
		sub := groupSubName(c.Group, c.Name)
		obs, stale := count([]string{c.Group, sub})
		want := 1
		if c.Language {
			want = 0
		}
		if obs != want || stale != want {
			t.Errorf("`ailang %s %s` ran observatory=%d stale=%d, want %d/%d (it is the same code as `ailang %s`)",
				c.Group, sub, obs, stale, want, want, c.Name)
		}
		if c.Language {
			checked++
		}
	}
	// Positive control: the rule is only meaningful if some language command
	// is actually reachable through a group.
	if checked == 0 {
		t.Fatal("no language command is reachable through a group — this test proved nothing")
	}

	// Negative control: a group with a PLATFORM member still probes, so the
	// hop did not simply silence every group route.
	if obs, stale := count([]string{groupOps, "messages"}); obs != 1 || stale != 1 {
		t.Errorf("`ailang ops messages` ran observatory=%d stale=%d, want 1/1", obs, stale)
	}
	// And an unknown subcommand under a group keeps probing, like any typo.
	if obs, stale := count([]string{groupOps, "not-a-command"}); obs != 1 || stale != 1 {
		t.Errorf("`ailang ops not-a-command` ran observatory=%d stale=%d, want 1/1", obs, stale)
	}
}

// TestGroups_RunAsTopLevelPresentsTheTopLevelArgv is the property that makes a
// group route identical to the spelling it mirrors.
//
// It is not cosmetic. 44 call sites in cmd/ailang parse os.Args[2:] or
// os.Args[3:] directly instead of the tail they are handed — messages.go reads
// os.Args[2] as its subcommand — so without the rewrite `ailang ops messages
// list` would read its subcommand as "messages" and no help text would show
// it. Mutation: delete the os.Args assignment in runAsName; this fails.
func TestGroups_RunAsTopLevelPresentsTheTopLevelArgv(t *testing.T) {
	savedArgs, savedInvoked := os.Args, invokedAs
	t.Cleanup(func() { os.Args, invokedAs = savedArgs, savedInvoked })

	var seenArgv []string
	var seenTail []string
	var seenInvoked string
	probe := &Command{Name: "messages", Run: func(args []string) error {
		seenArgv = append([]string(nil), os.Args...)
		seenTail = append([]string(nil), args...)
		seenInvoked = invokedAs
		return nil
	}}

	os.Args = []string{"ailang", "ops", "messages", "list", "--unread"}
	invokedAs = "ops"
	if err := runAsTopLevel(probe, []string{"list", "--unread"}); err != nil {
		t.Fatal(err)
	}

	wantArgv := []string{"ailang", "messages", "list", "--unread"}
	if strings.Join(seenArgv, " ") != strings.Join(wantArgv, " ") {
		t.Errorf("os.Args inside the command = %v, want %v — a command that reads os.Args[2] "+
			"would see the wrong subcommand", seenArgv, wantArgv)
	}
	if strings.Join(seenTail, " ") != "list --unread" {
		t.Errorf("tail = %v, want [list --unread]", seenTail)
	}
	if seenInvoked != "messages" {
		t.Errorf("invokedAs = %q, want %q", seenInvoked, "messages")
	}
	// And it is restored, so one group route cannot corrupt the next.
	if got := strings.Join(os.Args, " "); got != "ailang ops messages list --unread" {
		t.Errorf("os.Args not restored: %q", got)
	}
	if invokedAs != "ops" {
		t.Errorf("invokedAs not restored: %q", invokedAs)
	}
}
