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

// tasks 70b77905 / 1063e5fd, design-doc-creator-daneel, 2026-09-13.
//
// The wrapper creates coordinator/task-<id>; the agent made its OWN branch
// (design-doc/...) and committed there, so the coordinator branch never moved.
// `git push origin coordinator/task-<id>` pushed that unmoved ref and GitHub
// refused the PR: 422 "No commits between main and coordinator/task-1063e5fd".
// The design docs existed the whole time on a branch nobody was told about.
//
// hasWork measures clonePoint..HEAD and DID find those commits — the push then
// sent a different ref. The check and the push must mean the same commits.
func TestPushRefspec_SendsHeadNotTheBranchName(t *testing.T) {
	got := pushRefspec("coordinator/task-1063e5fd")
	if got != "HEAD:refs/heads/coordinator/task-1063e5fd" {
		t.Fatalf("must push HEAD to the branch, got %q", got)
	}
	// The bare name is the bug: it pushes the local ref of that name, which is
	// only the agent's work if the agent stayed where the wrapper put it.
	if got == "coordinator/task-1063e5fd" {
		t.Fatal("pushing the bare branch name loses work done on any other branch")
	}
}

// task-70b77905: `pr create failed: github api returned 500`, best-effort, so
// the run reported success and the work simply had no PR.
func TestShouldRetryPRCreate_TransientOnly(t *testing.T) {
	if !shouldRetryPRCreate(500, errFake{}) {
		t.Error("a 500 is GitHub having a moment — retry it")
	}
	if !shouldRetryPRCreate(502, errFake{}) {
		t.Error("502 is transient")
	}
	if !shouldRetryPRCreate(0, errFake{}) {
		t.Error("never reached the API (DNS/TLS/reset) — retry it")
	}
	// 422 "No commits between ..." is a real answer. Retrying repeats the same
	// rejection more slowly and hides it behind a delay.
	if shouldRetryPRCreate(422, errFake{}) {
		t.Error("422 is a verdict, not a hiccup")
	}
	if shouldRetryPRCreate(403, errFake{}) {
		t.Error("403 is a verdict")
	}
	if shouldRetryPRCreate(201, nil) {
		t.Error("success is not retried")
	}
}
