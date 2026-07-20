package motoko

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// systemPromptGuardVerdict classifies the outcome of the end-to-end
// system-prompt delivery check (M-RIG-RELIABILITY).
type systemPromptGuardVerdict int

const (
	// sysPromptDelivered: session reports system_md="set" — teaching reached
	// the model.
	sysPromptDelivered systemPromptGuardVerdict = iota
	// sysPromptStartupCrash: motoko died BEFORE step 0 — no
	// runtime_config_resolved broadcast, no run_summary, no steps executed.
	// This is NOT the delivery-regression class; the answer is in stderr.
	sysPromptStartupCrash
	// sysPromptDeliveryRegression: the session actually ran but system_md !=
	// "set" — the recurring env-forward/flag/path delivery bug.
	sysPromptDeliveryRegression
)

// guardSystemPromptDelivery is the END-TO-END SYSTEM-PROMPT DELIVERY GUARD
// (M-RIG-RELIABILITY). The delivery bug has recurred 7×, each time in a
// DIFFERENT layer (default flag, env-forward scrub, path rejection) with the
// IDENTICAL symptom: the AILANG teaching never reaches the model → dialect
// confusion → step-budget exhaustion. No per-layer fix stops it; only
// asserting the OUTCOME does. If we wrote a SYSTEM_MD file (intended
// delivery) but the session reports system_md != "set", delivery failed —
// FAIL LOUDLY (CLAUDE.md: NO SILENT FALLBACKS).
//
// BUT: system_md comes from motoko's step-0 runtime_config_resolved
// broadcast, so a crash before step 0 (e.g. the 2026-07-16 v0.30.0 std/ai
// Message `images` schema break) leaves the session JSONL essentially empty
// and produces the same system_md != "set" — for 4 days that misdirected
// diagnosis toward the delivery class. When the JSONL has no broadcast, no
// run_summary, AND no steps, report a startup crash pointing at the stderr
// log instead.
//
// Mutates result (ProviderData + Error) and returns the verdict plus the
// human-readable message ("" when delivered). The Execute call-site owns
// stderr printing and span attributes.
func guardSystemPromptDelivery(result *executor.Result, systemPromptPath, stderrLogPath, stderrTail string) (systemPromptGuardVerdict, string) {
	sysMD, _ := result.ProviderData["system_md"].(string)
	if sysMD == "set" {
		return sysPromptDelivered, ""
	}

	runSummaryPresent, _ := result.ProviderData["motoko_run_summary_present"].(bool)
	if sysMD == "" && !runSummaryPresent && result.NumTurns == 0 {
		msg := fmt.Sprintf("motoko crashed at startup before step 0 — session JSONL has no "+
			"runtime_config_resolved broadcast, no run_summary, and no steps. This is NOT the "+
			"system-prompt delivery regression; see stderr log at %s", stderrLogPath)
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "\nstderr tail:\n" + tail
		}
		result.ProviderData["motoko_startup_crash"] = msg
		result.Success = false
		if result.Error == "" {
			result.Error = msg
		} else {
			result.Error += " — " + msg
		}
		return sysPromptStartupCrash, msg
	}

	msg := fmt.Sprintf("SYSTEM PROMPT NOT DELIVERED: wrote SYSTEM_MD=%s but session reports system_md=%q "+
		"(expected \"set\"). The AILANG teaching did not reach the model — recurring delivery regression.",
		systemPromptPath, sysMD)
	result.ProviderData["system_prompt_delivery_error"] = msg
	return sysPromptDeliveryRegression, msg
}
