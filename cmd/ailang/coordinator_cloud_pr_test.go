package main

import "testing"

// task-389b7a51, design-doc-creator-daneel's first run, 2026-09-12.
//
// The agent pushed its own branch, so `git log origin/BRANCH..HEAD` was empty,
// the wrapper logged "no commits to push" and returned — and the PR call lived
// inside that push block. The design doc sat on a pushed branch nobody was told
// about, and a human opened the PR by hand six minutes later.
//
// Pushing and reviewing are separate questions, and these two predicates are
// the split.

func TestBranchNeedsPush_OnlyAnswersAboutPushing(t *testing.T) {
	if !branchNeedsPush(true, "") {
		t.Error("a branch absent from the remote must be pushed, even with no log output — git log exits 128 there")
	}
	if !branchNeedsPush(false, "abc123 do the thing\n") {
		t.Error("local commits ahead of origin must be pushed")
	}
	if branchNeedsPush(false, "   \n") {
		t.Error("nothing ahead of origin means nothing to push")
	}
}

// The regression: already-pushed must NOT mean no-PR.
func TestBranchWantsPR_IndependentOfWhoPushed(t *testing.T) {
	if !branchWantsPR("coordinator/task-389b7a51", "main") {
		t.Error("a task branch always wants a PR — including one the agent pushed itself")
	}
	// A direct-push agent (skip_approval) commits onto the base branch; GitHub
	// answers 422 for a PR from a branch to itself, and that is working as
	// designed, not a failure to log.
	if branchWantsPR("dev", "dev") {
		t.Error("no PR from a branch to itself")
	}
	if branchWantsPR("", "main") {
		t.Error("no branch, no PR")
	}
}
