package main

import (
	"errors"
	"testing"
)

// "A pull request already exists" is the idempotency signal, not a failure.
//
// Measured 2026-09-15: task-1074a9cf pushed and opened PR #1223 at 07:16, was
// re-executed, and at 07:19 reported "no PR could be opened for it: github api
// returned 422: A pull request already exists". Two of fifteen parallel runs
// died that way with their work landed and a PR open. Re-execution is normal —
// the stale detector re-dispatches and MaxTaskExecutions allows two — so PR
// creation has to be replayable.
func TestIsPRAlreadyExists(t *testing.T) {
	dup := errors.New(`github api returned 422: {"message":"Validation Failed","errors":[{"message":"A pull request already exists for sunholo-data:coordinator/task-1074a9cf."}]}`)
	if !isPRAlreadyExists(422, dup) {
		t.Error("GitHub's duplicate-PR answer must be recognised as success")
	}

	// 422 is ALSO "No commits between ...", which is a real failure: the branch
	// carries nothing and there is genuinely no PR to open. Matching on status
	// alone would turn that into a silent success.
	noCommits := errors.New(`github api returned 422: {"message":"Validation Failed","errors":[{"message":"No commits between dev and coordinator/task-x"}]}`)
	if isPRAlreadyExists(422, noCommits) {
		t.Error(`422 "No commits between" must stay a hard failure`)
	}

	for _, tc := range []struct {
		status int
		err    error
	}{
		{500, errors.New("server error")},
		{403, errors.New("forbidden")},
		{422, nil},
		{0, errors.New("connection reset")},
	} {
		if isPRAlreadyExists(tc.status, tc.err) {
			t.Errorf("status %d / %v must not read as a duplicate", tc.status, tc.err)
		}
	}
}
