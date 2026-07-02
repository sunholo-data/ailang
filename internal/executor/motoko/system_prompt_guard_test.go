package motoko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
