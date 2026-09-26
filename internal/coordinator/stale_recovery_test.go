package coordinator

import "testing"

// M-COORDINATOR-EXECUTION-TRUST M7 (design doc V36).
//
// RecoverStaleTasks marks every running/queued task older than the threshold
// "cancelled" on daemon startup. Its premise — "the daemon restarted, so
// anything still running must be dead" — is TRUE in local mode, where the daemon
// owns the worker process, and FALSE in cloud mode, where the executor is a
// separate Cloud Run Job with a lifecycle entirely independent of the
// coordinator's.
//
// The coordinator scales to zero. So every cold start cancelled every in-flight
// task older than five minutes, while the job that was actually doing the work
// carried on, finished, opened its PR, and had its completion DISCARDED:
//
//	12:33:57  Cloud dispatch: task task-c8126248 → Cloud Run Job
//	12:49:05  stale task detector: stopped        (coordinator scaled to zero)
//	12:50:40  execute-job: opened PR #56; completion published (status=completed)
//	12:50:45  Recovered 1 stale task(s) from previous daemon run
//	12:50:46  CompletionHandler: task already in terminal state "cancelled", skipping
//
// The work succeeded and the requester was never told — which is exactly the
// failure that makes the plane untrustworthy for passing tasks between sessions.
//
// The intended split was already documented at stale_task_detector.go: "Only
// runs in cloud mode ... Local mode uses RecoverStaleTasks at startup instead."
// Nothing enforced it.

func TestStartupRecoveryIsLocalModeOnly(t *testing.T) {
	if shouldRecoverStaleTasksOnStartup(CoordinatorModeCloud) {
		t.Error("cloud mode must NOT cancel in-flight tasks on startup — the executor " +
			"is a separate Cloud Run Job and outlives the coordinator")
	}
	if !shouldRecoverStaleTasksOnStartup(CoordinatorModeLocal) {
		t.Error("local mode must still recover: there the daemon owned the worker process")
	}
}

// An unset or unrecognised mode must behave like local. Local is the documented
// default, and the cost of being wrong is asymmetric: recovering when you should
// not have merely re-runs a task, while NOT recovering a genuinely dead local
// task leaves it running forever.
func TestUnsetModeRecoversLikeLocal(t *testing.T) {
	for _, mode := range []string{"", "LOCAL", "something-else"} {
		if !shouldRecoverStaleTasksOnStartup(mode) {
			t.Errorf("mode %q must default to recovering, like local", mode)
		}
	}
}
