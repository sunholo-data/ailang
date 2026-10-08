package riglock

import (
	"os"
	"path/filepath"
	"time"
)

// Operator presence (2026-10-06).
//
// The rig's GPU also draws its desktop. While a local model generates, the GPU
// sits at ~100% and window focus lags by seconds — measured on the Mac Studio
// with the rotation filler running qwen3.8-27b. A watcher
// (tools/launchd/rig-operator-presence.sh) keeps rig.operator, beside the lock,
// in force while someone is using the keyboard or mouse. Batch jobs that opt in
// with AILANG_RIG_YIELD_TO_OPERATOR=1 lend the GPU at their next checkpoint and
// take it back once the operator has gone idle.
//
// It is deliberately NOT a handoff. A handoff is a short job's one-off ask and
// is first-come: an all-day presence marker there would refuse Daneel's intake
// for the whole working day. And it is opt-in, because an attended eval the
// operator started must not stall just because the operator is typing.

func operatorPath() string {
	return filepath.Join(filepath.Dir(lockDir()), "rig.operator")
}

// OperatorPresent reports whether the presence marker is in force. It uses the
// handoff's requester/pid/until format and expiry rules, so a watcher that dies
// stops pausing the rig within one marker lifetime instead of forever.
func OperatorPresent() bool {
	b, err := os.ReadFile(operatorPath())
	if err != nil {
		return false
	}
	y, err := parseYield(string(b))
	if err != nil || time.Now().After(y.Until) || (y.PID > 0 && !pidAlive(y.PID)) {
		return false
	}
	return true
}
