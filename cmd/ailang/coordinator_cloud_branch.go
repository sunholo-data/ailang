package main

import "strings"

// Branch decisions for execute-job: what to push, and whether a PR makes sense.

// pushRefspec sends whatever HEAD is to the named branch.
//
// Deliberately not the bare branch name: that pushes the local ref of that name,
// which is only the agent's work if the agent stayed on the branch the wrapper
// created. Nothing makes it stay.
func pushRefspec(branchName string) string {
	return "HEAD:refs/heads/" + branchName
}

// branchNeedsPush reports whether the wrapper still has commits to send.
//
// It answers ONLY that. It used to be the same test that decided whether to
// open a PR, which meant an agent that pushed its own branch got no PR at all
// (task-389b7a51). Pushing and reviewing are separate questions.
func branchNeedsPush(newBranch bool, logOutput string) bool {
	return newBranch || len(strings.TrimSpace(logOutput)) > 0
}

// branchWantsPR reports whether there is a head/base pair to open a PR between.
//
// A direct-push agent commits onto the base branch itself; GitHub answers 422
// for a PR from a branch to itself, and that is not a failure worth logging.
func branchWantsPR(branchName, baseBranch string) bool {
	return branchName != "" && branchName != baseBranch
}
