package motoko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestSystemPromptViaSystemRole_DefaultOn pins the DEFAULT system-prompt
// delivery mode (M-RIG-RELIABILITY). The AILANG teaching MUST be delivered as a
// persistent system-role message by default — folding it into the user message
// loses it to context compaction on long runs, which is the recurring "system
// prompt never reached the model" bug. If someone reverts the default back to
// gated (== "1"), the first case fails in CI. This guards the CALL-SITE, not
// just a helper, per the lesson from that 7×-recurring regression.
func TestSystemPromptViaSystemRole_DefaultOn(t *testing.T) {
	saved, had := os.LookupEnv("AILANG_MOTOKO_SYSTEM_ROLE")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("AILANG_MOTOKO_SYSTEM_ROLE", saved)
		} else {
			_ = os.Unsetenv("AILANG_MOTOKO_SYSTEM_ROLE")
		}
	})

	_ = os.Unsetenv("AILANG_MOTOKO_SYSTEM_ROLE")
	if !systemPromptViaSystemRole() {
		t.Fatal("DEFAULT (unset) must deliver the AILANG system prompt via the system role; " +
			"got fold-into-directive, which is stripped by context compaction on long runs " +
			"(reintroduces the recurring delivery-loss bug)")
	}

	_ = os.Setenv("AILANG_MOTOKO_SYSTEM_ROLE", "0")
	if systemPromptViaSystemRole() {
		t.Error("AILANG_MOTOKO_SYSTEM_ROLE=0 must opt OUT (fold into directive)")
	}

	_ = os.Setenv("AILANG_MOTOKO_SYSTEM_ROLE", "1")
	if !systemPromptViaSystemRole() {
		t.Error("AILANG_MOTOKO_SYSTEM_ROLE=1 must deliver via the system role")
	}
}

// TestParseSessionJSONL_CapturesSystemMD verifies the parser surfaces the
// end-to-end delivery signal from runtime_config_resolved into
// ProviderData["system_md"]. The executor asserts on this so ANY layer that
// breaks delivery (default flag, env-forward scrub, path rejection) fails
// loudly instead of silently recurring (M-RIG-RELIABILITY).
func TestParseSessionJSONL_CapturesSystemMD(t *testing.T) {
	for _, tc := range []struct{ name, state string }{
		{"delivered", "set"},
		{"not delivered", "unset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "session.jsonl")
			lines := []string{
				`{"schema_version":"1","session_id":"session_x","type":"session_start","task":"t","model":"m"}`,
				`{"schema_version":"1","session_id":"session_x","type":"runtime_config_resolved","model":"m","step_budget":60,"system_md":"` + tc.state + `"}`,
				`{"schema_version":"1","session_id":"session_x","type":"run_summary","model":"m","finish_reason":"stop","steps_executed":1,"duration_ms":1000}`,
			}
			if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			res, err := parseSessionJSONL(p)
			if err != nil {
				t.Fatalf("parseSessionJSONL: %v", err)
			}
			got, _ := res.ProviderData["system_md"].(string)
			if got != tc.state {
				t.Errorf("ProviderData[system_md] = %q, want %q", got, tc.state)
			}
		})
	}
}

// parseGuardFixture writes the given JSONL lines to a temp session file and
// runs them through parseSessionJSONL, so guard tests exercise the real
// parser-produced Result (system_md + motoko_run_summary_present) rather than
// hand-built ProviderData.
func parseGuardFixture(t *testing.T, lines []string) *executor.Result {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := parseSessionJSONL(p)
	if err != nil {
		t.Fatalf("parseSessionJSONL: %v", err)
	}
	return res
}

// TestGuardSystemPromptDelivery_StartupCrash: when the session JSONL is
// essentially empty (no runtime_config_resolved broadcast, no run_summary, no
// steps — e.g. the 2026-07-16 v0.30.0 std/ai Message `images` schema break
// that crash-looped the rig for 4 days), the guard must report a startup
// crash pointing at the stderr log — NOT the delivery-regression warning,
// which misdirected diagnosis toward the env-forward delivery class.
func TestGuardSystemPromptDelivery_StartupCrash(t *testing.T) {
	res := parseGuardFixture(t, []string{
		`{"schema_version":"1","session_id":"session_x","type":"session_start","task":"t","model":"m"}`,
	})

	verdict, msg := guardSystemPromptDelivery(res, "/ws/.motoko_system.md",
		"/tmp/motoko-stderr-x.log", "TypeError: Message has no field `images`")
	if verdict != sysPromptStartupCrash {
		t.Fatalf("verdict = %v, want sysPromptStartupCrash. msg: %s", verdict, msg)
	}
	if !strings.Contains(msg, "before step 0") {
		t.Errorf("message should name the startup-crash case; got: %s", msg)
	}
	if !strings.Contains(msg, "/tmp/motoko-stderr-x.log") {
		t.Errorf("message must include the stderr log path; got: %s", msg)
	}
	if !strings.Contains(msg, "Message has no field `images`") {
		t.Errorf("message must include the stderr tail; got: %s", msg)
	}
	if _, has := res.ProviderData["system_prompt_delivery_error"]; has {
		t.Error("startup crash must NOT set system_prompt_delivery_error (misdirects to the delivery class)")
	}
	if _, has := res.ProviderData["motoko_startup_crash"]; !has {
		t.Error("ProviderData[motoko_startup_crash] not set")
	}
	if res.Success {
		t.Error("Success = true after startup crash, want false")
	}
	if !strings.Contains(res.Error, "/tmp/motoko-stderr-x.log") {
		t.Errorf("result.Error must carry the stderr log path; got: %s", res.Error)
	}
}

// TestGuardSystemPromptDelivery_DeliveryRegression: the session actually ran
// (step-0 broadcast present, steps executed) but reports system_md != "set" —
// the recurring 7×-regressed delivery bug. The guard must keep the loud
// delivery warning and must NOT claim a startup crash.
func TestGuardSystemPromptDelivery_DeliveryRegression(t *testing.T) {
	res := parseGuardFixture(t, []string{
		`{"schema_version":"1","session_id":"session_x","type":"session_start","task":"t","model":"m"}`,
		`{"schema_version":"1","session_id":"session_x","type":"runtime_config_resolved","model":"m","step_budget":60,"system_md":"unset"}`,
		`{"schema_version":"1","session_id":"session_x","type":"thinking","step":1,"text":"...","finish_reason":"stop","input_tokens":10,"output_tokens":5}`,
		`{"schema_version":"1","session_id":"session_x","type":"run_summary","model":"m","finish_reason":"stop","steps_executed":1,"duration_ms":1000}`,
	})

	verdict, msg := guardSystemPromptDelivery(res, "/ws/.motoko_system.md",
		"/tmp/motoko-stderr-x.log", "")
	if verdict != sysPromptDeliveryRegression {
		t.Fatalf("verdict = %v, want sysPromptDeliveryRegression. msg: %s", verdict, msg)
	}
	if !strings.Contains(msg, "SYSTEM PROMPT NOT DELIVERED") {
		t.Errorf("delivery-regression warning lost; got: %s", msg)
	}
	if _, has := res.ProviderData["system_prompt_delivery_error"]; !has {
		t.Error("ProviderData[system_prompt_delivery_error] not set")
	}
	if _, has := res.ProviderData["motoko_startup_crash"]; has {
		t.Error("delivery regression must NOT be labeled a startup crash")
	}
}

// TestGuardSystemPromptDelivery_MidRunCrash_StillDeliveryClass: a crash AFTER
// steps ran but before run_summary (no broadcast either — e.g. pre-broadcast
// motoko) is NOT a startup crash: the model was actually running without the
// teaching, so the delivery warning is the right diagnosis.
func TestGuardSystemPromptDelivery_MidRunCrash_StillDeliveryClass(t *testing.T) {
	res := parseGuardFixture(t, []string{
		`{"schema_version":"1","session_id":"session_x","type":"session_start","task":"t","model":"m"}`,
		`{"schema_version":"1","session_id":"session_x","type":"thinking","step":1,"text":"...","finish_reason":"length","input_tokens":10,"output_tokens":5}`,
	})

	verdict, _ := guardSystemPromptDelivery(res, "/ws/.motoko_system.md",
		"/tmp/motoko-stderr-x.log", "")
	if verdict != sysPromptDeliveryRegression {
		t.Fatalf("verdict = %v, want sysPromptDeliveryRegression (steps ran → not a startup crash)", verdict)
	}
}

// TestGuardSystemPromptDelivery_Delivered: system_md="set" → no warning, no
// mutation.
func TestGuardSystemPromptDelivery_Delivered(t *testing.T) {
	res := parseGuardFixture(t, []string{
		`{"schema_version":"1","session_id":"session_x","type":"session_start","task":"t","model":"m"}`,
		`{"schema_version":"1","session_id":"session_x","type":"runtime_config_resolved","model":"m","step_budget":60,"system_md":"set"}`,
		`{"schema_version":"1","session_id":"session_x","type":"run_summary","model":"m","finish_reason":"stop","steps_executed":1,"duration_ms":1000}`,
	})

	verdict, msg := guardSystemPromptDelivery(res, "/ws/.motoko_system.md",
		"/tmp/motoko-stderr-x.log", "")
	if verdict != sysPromptDelivered {
		t.Fatalf("verdict = %v, want sysPromptDelivered. msg: %s", verdict, msg)
	}
	if msg != "" {
		t.Errorf("delivered case must return empty message; got: %s", msg)
	}
	if _, has := res.ProviderData["system_prompt_delivery_error"]; has {
		t.Error("delivered case must not set system_prompt_delivery_error")
	}
	if _, has := res.ProviderData["motoko_startup_crash"]; has {
		t.Error("delivered case must not set motoko_startup_crash")
	}
}
