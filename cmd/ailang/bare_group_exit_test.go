package main

import (
	"strings"
	"testing"
)

// M-V1-SIMPLIFY-S5 M4: a command that takes a subcommand and got none exits 1.
//
// Measured on bin/ailang-before (the merge-base binary, 2026-09-18):
//
//	models 0  workspaces 0  storage 0  access-control 0
//	chains 1  pkg 1  daemon 1  dev 1  ops 1
//
// Same shape, two answers. The three that returned 0 told a script that had
// forgotten its subcommand that the command had succeeded — `ailang models` and
// `ailang storage` print help and do nothing, which is not success. `budget` is
// excluded on purpose: a bare `ailang budget` defaults to `status` and really
// does run it, so it is not the no-subcommand shape.
//
// `trace`, `observatory` and `dashboard` also return 0 here. They are M3's
// files (the chains fold) and are deliberately NOT touched by this test, so
// that the two milestones do not collide on the same lines.
func TestBareGroup_NoSubcommandExitsOne(t *testing.T) {
	bin := cliTestBin(t)

	// The three this milestone changes, plus controls that were ALREADY 1.
	// The controls are what makes this a test of a rule rather than of three
	// special cases: if the dispatch table ever starts swallowing exit codes,
	// they fail too.
	for _, name := range []string{
		"models", "workspaces", "storage", // changed by M4
		"chains", "pkg", "daemon", // controls, unchanged
	} {
		t.Run(name, func(t *testing.T) {
			res := runCLIIsolated(t, bin, name)
			if res.timedOut {
				t.Fatalf("`ailang %s` with no subcommand did not exit", name)
			}
			if res.exitCode != 1 {
				t.Errorf("`ailang %s` exit code = %d, want 1 — no subcommand is a usage error\nstdout: %s\nstderr: %s",
					name, res.exitCode, firstLines(res.stdout, 5), firstLines(res.stderr, 5))
			}
			// It must still SAY something, or the non-zero exit is unexplained.
			if strings.TrimSpace(res.stdout+res.stderr) == "" {
				t.Errorf("`ailang %s` exited %d silently", name, res.exitCode)
			}
		})
	}
}

// TestBareGroup_HelpStillExitsZero is the other half, and the reason the change
// above is safe: asking for help is not the same as forgetting an argument.
// Without this, "make the bare case exit 1" could be satisfied by making the
// whole command exit 1, which would break `ailang <group> --help` — the M1
// acceptance criterion that every group answers --help with exit 0.
func TestBareGroup_HelpStillExitsZero(t *testing.T) {
	bin := cliTestBin(t)
	for _, name := range []string{"models", "workspaces", "storage"} {
		for _, flag := range []string{"--help", "-h", "help"} {
			t.Run(name+" "+flag, func(t *testing.T) {
				res := runCLIIsolated(t, bin, name, flag)
				if res.timedOut {
					t.Fatalf("`ailang %s %s` did not exit", name, flag)
				}
				if res.exitCode != 0 {
					t.Errorf("`ailang %s %s` exit code = %d, want 0\nstdout: %s\nstderr: %s",
						name, flag, res.exitCode, firstLines(res.stdout, 5), firstLines(res.stderr, 5))
				}
				// The help must reach stdout (not stderr) and name the command
				// it is about. NOT asserted: a literal "Usage:" line —
				// `workspaces` and `storage` open with one, `models` opens
				// with "ailang models — the model registry (models.yml)".
				// That difference predates this milestone (confirmed against
				// bin/ailang-before) and normalising help prose is M6's
				// generated-CLI-reference work, not a removal milestone's.
				if !strings.Contains(res.stdout, "ailang "+name) {
					t.Errorf("`ailang %s %s` printed no help naming the command on stdout:\n%s",
						name, flag, firstLines(res.stdout, 10))
				}
			})
		}
	}
}
