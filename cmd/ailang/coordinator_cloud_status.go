package main

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// contractCompletionStatus is the producer half of the executor status contract
// (M-TASK-STATUS-TRUTH S2). A status the coordinator does not understand is
// dropped there and the task later reads as a timeout — which is what happened
// to every `blocked` run between 2026-09-14 and 2026-09-25. Refuse here instead,
// loudly, and still report a terminal outcome that names the defect.
func contractCompletionStatus(status, errMsg string) (string, string) {
	if coordinator.IsExecutorCompletionStatus(status) {
		return status, errMsg
	}
	fmt.Fprintf(os.Stderr, "execute-job: refusing to publish unknown completion status %q; publishing failed instead\n", status)
	return string(coordinator.TaskStatusFailed),
		fmt.Sprintf("executor produced unknown completion status %q (not in coordinator.ExecutorCompletionStatuses): %s", status, errMsg)
}
