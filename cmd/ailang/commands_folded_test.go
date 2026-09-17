package main

import (
	"bytes"
	"strings"
	"testing"
)

// M-V1-SIMPLIFY-S5 M3 — the fold of `trace`, `observatory`, `dashboard` and
// `eval-chains` into `chains`.
//
// The property that matters is NOT "the alias equals the canonical spelling".
// Measured while writing these tests: mutate the chains dispatcher to hand the
// namespace its own name back (`flag.Args()[1:]` instead of `[2:]`) and BOTH
// spellings break identically — `ailang trace status` and `ailang chains trace
// status` each print "Unknown trace subcommand: trace" and a pair-diff harness
// reports 31/31 identical. A twin comparison cannot see a fault the twins
// share. So every pair test below is anchored by a test that pins the route to
// OUTPUT it must actually produce, and the milestone's real instrument is the
// before/after surface diff against bin/ailang-before, not the pairs.

// TestFolded_TableRowsAreAliasesThatRouteThroughChains states the shape of the
// fold as table invariants: each namespace has a Legacy row, the row still
// resolves (D1 — 58 references to `ailang trace ...` live in the
// trace-debugger skill alone), and the row is Hidden because help should stop
// advertising a spelling that goes away after one release.
//
// Mutation: drop Hidden from the trace row, or delete the row, and this names
// which invariant broke.
func TestFolded_TableRowsAreAliasesThatRouteThroughChains(t *testing.T) {
	ns := foldedNamespaces()
	if len(ns) != 4 {
		t.Fatalf("expected 4 folded namespaces, got %d — if one was added or removed, "+
			"say so here and in the changelog", len(ns))
	}
	for _, n := range ns {
		c, ok := lookupCommand(n.Legacy)
		if !ok {
			t.Errorf("`ailang %s` no longer resolves — D1 says every pre-S5 spelling keeps working", n.Legacy)
			continue
		}
		if !c.Hidden {
			t.Errorf("the %q row is not Hidden; a folded alias is not advertised in help", n.Legacy)
		}
		if n.Run == nil || n.Help == nil {
			t.Errorf("namespace %q has a nil Run or Help", n.Sub)
		}
		if lookupFoldedNamespace(n.Sub) == nil {
			t.Errorf("`ailang chains %s` does not resolve to a namespace", n.Sub)
		}
	}
	// The namespace words must not shadow a chains subcommand. `chains stats`
	// and `chains view` already exist, which is exactly why the fold uses
	// namespaces and not a flattening of four colliding subcommand sets.
	for _, collide := range []string{"list", "view", "stats", "health", "tree", "chat", "find", "diff"} {
		if lookupFoldedNamespace(collide) != nil {
			t.Errorf("namespace %q shadows the chains subcommand of the same name", collide)
		}
	}
}

// TestFolded_ChainsHelpListsTheFoldedNamespaces is the milestone's help
// acceptance line: `ailang chains --help` must name what was folded in, or the
// fold is undiscoverable and the four aliases are the only way anyone finds
// these commands.
//
// It drives the BINARY rather than calling printFoldedNamespaces directly.
// Calling the renderer would pass even if printChainsHelp never called it, and
// even if `chains` were still answering --help from the dispatch table's
// generic block (which cannot name a namespace) — the two ways this line can
// actually be broken.
//
// Mutation: put "chains" back in helpFallbackCommands, or delete the
// printFoldedNamespaces call from printChainsHelp, and this fails naming the
// namespace that vanished.
func TestFolded_ChainsHelpListsTheFoldedNamespaces(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the binary")
	}
	bin := cliTestBin(t)
	res := runCLIIn(t, bin, t.TempDir(), t.TempDir(), "chains", "--help")
	if res.timedOut {
		t.Fatalf("`ailang chains --help` did not finish inside the bound")
	}
	if res.exitCode != 0 {
		t.Errorf("`ailang chains --help` exited %d, want 0", res.exitCode)
	}
	for _, n := range foldedNamespaces() {
		if !strings.Contains(res.stdout, "chains "+n.Sub) {
			t.Errorf("`ailang chains --help` does not list the %q namespace:\n%s",
				n.Sub, firstLines(res.stdout, 40))
		}
	}
	// And the renderer itself, so a failure says which half broke.
	var buf bytes.Buffer
	printFoldedNamespaces(&buf)
	for _, n := range foldedNamespaces() {
		if !strings.Contains(buf.String(), "chains "+n.Sub) {
			t.Errorf("printFoldedNamespaces omits %q", n.Sub)
		}
	}
}

// deletedObservatorySubcommands are the eight D7 removals.
//
// Each was verified at ZERO references over tracked files in Makefile, make/,
// tools/, scripts/, .github/, .claude/skills/, .agents/skills/ and
// .claude/rules/ (`git grep -F "ailang observatory <sub>"`), with the only
// citation anywhere a line in the frozen devtools prompt prompts/devtools/
// v0.8.0.md. Last substantive commit for all eight: January 2026 — `git log
// -S` puts the last content change at 7bfb1e834 (2026-01-22), itself a
// mechanical file split, over d58ac50dd / 6b826ff13 / 4242c4e5f / fa41f2223
// (2026-01-07 .. 2026-01-19). The 2026-04-21 date in the CLI audit is
// de4aa7ba8, the module-path rename.
var deletedObservatorySubcommands = []string{
	"seed", "heatmap", "evolution", "usage", "tokens", "outliers", "metrics", "hierarchy",
}

// TestFolded_DeletedObservatorySubcommandsAreGone pins the removal. It is the
// half of the milestone a build cannot check: the files are gone, so nothing
// references the functions, so nothing fails if a future edit quietly
// reintroduces a `case "heatmap"` arm that dispatches to something else.
//
// Mutation: add `case "heatmap": chainsHealthCommand()` to observatoryCommand
// and this fails naming heatmap.
func TestFolded_DeletedObservatorySubcommandsAreGone(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the binary 16 times")
	}
	bin := cliTestBin(t)
	home, work := t.TempDir(), t.TempDir()

	// Control first: a subcommand that MUST still work. Without it, a binary
	// that rejects everything would pass every assertion below.
	ctl := runCLIIn(t, bin, home, work, "observatory", "backfill", "--help")
	if ctl.timedOut {
		t.Fatalf("control `ailang observatory backfill --help` did not finish")
	}
	if strings.Contains(ctl.stderr, "Unknown observatory subcommand") {
		t.Fatalf("control failed: `observatory backfill` is rejected too, so the "+
			"assertions below measure nothing\nstderr: %s", ctl.stderr)
	}

	for _, sub := range deletedObservatorySubcommands {
		sub := sub
		t.Run(sub, func(t *testing.T) {
			for _, spelling := range [][]string{
				{"observatory", sub},
				{"chains", "observatory", sub},
			} {
				res := runCLIIn(t, bin, home, work, spelling...)
				if res.timedOut {
					t.Fatalf("`ailang %s` did not finish inside the bound", strings.Join(spelling, " "))
				}
				if res.exitCode == 0 {
					t.Errorf("`ailang %s` exited 0; it was deleted under D7 and must not succeed",
						strings.Join(spelling, " "))
				}
				if !strings.Contains(res.stderr, "Unknown observatory subcommand") {
					t.Errorf("`ailang %s` did not report an unknown subcommand\nstdout: %s\nstderr: %s",
						strings.Join(spelling, " "), firstLines(res.stdout, 5), firstLines(res.stderr, 5))
				}
			}
		})
	}
}

// TestFolded_AnchorTheRouteToRealOutput is the test the pair diff cannot be:
// it names output each route must actually produce. Without it, a dispatcher
// bug that breaks the alias AND the canonical spelling in the same way passes
// every equivalence check (measured — see the file comment).
//
// Mutation: change the chains namespace dispatch to pass flag.Args()[1:] and
// this fails on every row, because each prints "Unknown ... subcommand".
func TestFolded_AnchorTheRouteToRealOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the binary several times")
	}
	bin := cliTestBin(t)
	home, work := t.TempDir(), t.TempDir()

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"trace status via alias", []string{"trace", "status"}, "Telemetry Configuration Status"},
		{"trace status via chains", []string{"chains", "trace", "status"}, "Telemetry Configuration Status"},
		{"trace status via ops", []string{"ops", "trace", "status"}, "Telemetry Configuration Status"},
		{"observatory help via chains", []string{"chains", "observatory"}, "backfill"},
		{"dashboard help via chains", []string{"chains", "dashboard"}, "spans"},
		{"eval-chains help via chains", []string{"chains", "eval"}, "failures"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := runCLIIn(t, bin, home, work, tc.args...)
			if res.timedOut {
				t.Fatalf("`ailang %s` did not finish inside the bound", strings.Join(tc.args, " "))
			}
			if !strings.Contains(res.stdout, tc.want) {
				t.Errorf("`ailang %s` did not print %q — the route reaches the wrong code\nstdout: %s\nstderr: %s",
					strings.Join(tc.args, " "), tc.want, firstLines(res.stdout, 10), firstLines(res.stderr, 5))
			}
		})
	}
}

// TestFolded_AliasIsByteIdenticalToTheCanonicalSpelling is the D1 half: an
// existing invocation must not change. It is a twin comparison, so it is only
// meaningful alongside TestFolded_AnchorTheRouteToRealOutput above — read the
// file comment before trusting a green run of this test alone.
func TestFolded_AliasIsByteIdenticalToTheCanonicalSpelling(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the binary ~30 times")
	}
	bin := cliTestBin(t)

	pairs := []struct {
		name   string
		legacy []string
		canon  []string
	}{
		// Controls first: the same route twice, and a route outside the fold.
		// If either differs, the instrument is measuring noise.
		{"control/self", []string{"trace", "status"}, []string{"trace", "status"}},
		{"control/outside-the-fold", []string{"chains", "health"}, []string{"ops", "chains", "health"}},

		{"trace/status", []string{"trace", "status"}, []string{"chains", "trace", "status"}},
		{"trace/hierarchy", []string{"trace", "hierarchy"}, []string{"chains", "trace", "hierarchy"}},
		{"trace/hierarchy --limit", []string{"trace", "hierarchy", "--limit", "3"}, []string{"chains", "trace", "hierarchy", "--limit", "3"}},
		{"trace/bare", []string{"trace"}, []string{"chains", "trace"}},
		{"trace/unknown-sub", []string{"trace", "nope"}, []string{"chains", "trace", "nope"}},

		{"observatory/bare", []string{"observatory"}, []string{"chains", "observatory"}},
		{"observatory/cleanup --dry-run", []string{"observatory", "cleanup", "--dry-run"}, []string{"chains", "observatory", "cleanup", "--dry-run"}},
		{"observatory/backfill --help", []string{"observatory", "backfill", "--help"}, []string{"chains", "observatory", "backfill", "--help"}},

		{"dashboard/bare", []string{"dashboard"}, []string{"chains", "dashboard"}},
		{"dashboard/health", []string{"dashboard", "health"}, []string{"chains", "dashboard", "health"}},
		{"dashboard/unknown-sub", []string{"dashboard", "nope"}, []string{"chains", "dashboard", "nope"}},

		{"eval-chains/bare", []string{"eval-chains"}, []string{"chains", "eval"}},
		{"eval-chains/list", []string{"eval-chains", "list"}, []string{"chains", "eval", "list"}},
		{"eval-chains/list --limit", []string{"eval-chains", "list", "--limit", "3"}, []string{"chains", "eval", "list", "--limit", "3"}},

		// And through the groups, which is a third spelling of each.
		{"ops/trace status", []string{"trace", "status"}, []string{"ops", "trace", "status"}},
		{"eval/chains list", []string{"eval-chains", "list"}, []string{"eval", "chains", "list"}},
	}

	for _, p := range pairs {
		p := p
		t.Run(p.name, func(t *testing.T) {
			// ONE home and ONE working directory for both runs, for the same
			// reason commands_groups_test.go does it: a fresh temp dir per run
			// makes a route that prints its CWD differ from itself.
			home, work := t.TempDir(), t.TempDir()
			a := runCLIIn(t, bin, home, work, p.legacy...)
			b := runCLIIn(t, bin, home, work, p.canon...)
			if a.timedOut || b.timedOut {
				t.Fatalf("one of the invocations did not finish inside the bound")
			}
			if a.exitCode != b.exitCode {
				t.Errorf("exit code: `ailang %s` = %d, `ailang %s` = %d",
					strings.Join(p.legacy, " "), a.exitCode,
					strings.Join(p.canon, " "), b.exitCode)
			}
			if a.stdout != b.stdout {
				t.Errorf("stdout differs between `ailang %s` and `ailang %s`:\n--- alias ---\n%s\n--- canonical ---\n%s",
					strings.Join(p.legacy, " "), strings.Join(p.canon, " "),
					firstLines(a.stdout, 8), firstLines(b.stdout, 8))
			}
			if a.stderr != b.stderr {
				t.Errorf("stderr differs between `ailang %s` and `ailang %s`:\n--- alias ---\n%s\n--- canonical ---\n%s",
					strings.Join(p.legacy, " "), strings.Join(p.canon, " "),
					firstLines(a.stderr, 8), firstLines(b.stderr, 8))
			}
		})
	}
}

// TestFolded_BareNamespaceExitsOne is M1 finding 3, for the commands in M3's
// file set.
//
// Measured against bin/ailang-before: `trace`, `observatory`, `dashboard`,
// `models` and `workspaces` printed their subcommand list and exited **0**,
// while `chains`, `budget`, `pkg` and `daemon` printed theirs and exited **1**
// — the same "you named no subcommand" shape with two different answers, which
// makes `ailang <group> && next-step` do the wrong thing for five of them.
// M3 normalises the four it owns. `models`, `workspaces` and `budget` belong
// to M4/M6 and are deliberately NOT asserted here.
func TestFolded_BareNamespaceExitsOne(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the binary several times")
	}
	bin := cliTestBin(t)
	home, work := t.TempDir(), t.TempDir()

	for _, name := range []string{"trace", "observatory", "dashboard", "eval-chains", "chains"} {
		name := name
		t.Run(name, func(t *testing.T) {
			res := runCLIIn(t, bin, home, work, name)
			if res.timedOut {
				t.Fatalf("`ailang %s` did not finish inside the bound", name)
			}
			if res.exitCode != 1 {
				t.Errorf("`ailang %s` with no subcommand exited %d, want 1", name, res.exitCode)
			}
			// It must still SAY something: exiting 1 silently would satisfy
			// the line above and help nobody.
			if strings.TrimSpace(res.stdout) == "" {
				t.Errorf("`ailang %s` exited 1 with empty stdout; it must print its subcommand list", name)
			}
		})
	}
}
