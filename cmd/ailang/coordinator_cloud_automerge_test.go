package main

import "testing"

// isDocsPath is the second of the two conditions that must both hold before
// GitHub is allowed to merge an agent's PR without a human. It has to agree
// EXACTLY with the docs-only lane in .github/workflows/ci.yml — if the two
// disagree, a PR can take the fast CI lane (compiler matrix skipped) and then be
// auto-merged on the strength of checks that never ran what it changed.

func TestIsDocsPath_OnlyDesignDocsAndChangelogs(t *testing.T) {
	safe := []string{
		"design_docs/planned/m-openrouter-eu-routing.md",
		"design_docs/implemented/v0_35_0/m-thing.md",
		"changelogs/v0.32-current.md",
	}
	for _, f := range safe {
		if !isDocsPath(f) {
			t.Errorf("%q should be docs-only", f)
		}
	}

	unsafe := []struct{ path, why string }{
		{"internal/coordinator/daemon.go", "Go source"},
		{"cmd/ailang/main.go", "Go source"},
		{".github/workflows/ci.yml", "CI config decides what the gate even is"},
		{"internal/modelreg/models.yml", "pricing and model data"},
		{"std/list.ail", "stdlib"},
		// The .md files that are NOT documentation in the relevant sense: these
		// change agent behaviour, so auto-merging them would let an agent widen
		// its own instructions without review.
		{"CLAUDE.md", "changes how every session behaves"},
		{".claude/rules/coding-standards.md", "changes how every session behaves"},
		{".claude/skills/mission-control/SKILL.md", "changes how the loop behaves"},
		// The website has its own build; not part of the fast lane.
		{"docs/docs/guides/evaluation.md", "website content, separate build"},
		{"README.md", "not under design_docs/ or changelogs/"},
		// Path-prefix confusion: a directory that merely starts with the same
		// letters must not pass.
		{"design_docs_archive/old.md", "not the design_docs/ directory"},
		{"changelogs_old/v1.md", "not the changelogs/ directory"},
	}
	for _, tt := range unsafe {
		t.Run(tt.path, func(t *testing.T) {
			if isDocsPath(tt.path) {
				t.Errorf("%q must NOT be treated as docs-only (%s)", tt.path, tt.why)
			}
		})
	}
}

func TestIsDocsPath_NonMarkdownUnderDesignDocsIsNotDocs(t *testing.T) {
	// A script or fixture parked under design_docs/ is still code. The extension
	// check runs first for exactly this reason.
	for _, f := range []string{
		"design_docs/planned/helper.sh",
		"design_docs/fixtures/input.json",
		"changelogs/generate.py",
	} {
		if isDocsPath(f) {
			t.Errorf("%q is not a markdown document and must not be docs-only", f)
		}
	}
}
