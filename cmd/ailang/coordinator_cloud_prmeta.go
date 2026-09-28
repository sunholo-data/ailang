package main

import (
	"fmt"
	"strings"
)

// mergeWarning is the first line of a coordinator PR whose approval hands off to
// another agent. Merging the PR is how the approval is recorded, so merging it
// STARTS that agent — and nothing on the PR used to say so: a sprint plan read
// like inert paperwork while merging it would start sprint-executor writing code
// (feedback 2026-09-28, PR #1339). The targets come from the same registry edges
// the approval card shows ("APPROVING DISPATCHES: …"), so the two cannot differ.
// "" when merging starts nothing.
func mergeWarning(starts []string) string {
	if len(starts) == 0 {
		return ""
	}
	names := make([]string, len(starts))
	for i, s := range starts {
		names[i] = "**" + s + "**"
	}
	return fmt.Sprintf("> ⚠️ **Merging this approves the task and starts %s.** "+
		"Merge only when you want that next stage to run.\n\n", strings.Join(names, " and "))
}
