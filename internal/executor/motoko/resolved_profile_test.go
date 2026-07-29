package motoko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseSessionJSONL_CapturesResolvedProfile verifies the parser surfaces
// the profile the subject ACTUALLY loaded into ProviderData, so the harness can
// assert it against the models.yml claim (M-EVAL-MEASUREMENT-CONTRACT M4).
//
// Ten cloud motoko entries silently ran `dogfood` — no ailang_docs, no
// microrag, and a verify gate that cannot work in a benchmark workspace — for
// weeks while advertising the opposite. Nothing caught it because the claim was
// never compared against reality.
func TestParseSessionJSONL_CapturesResolvedProfile(t *testing.T) {
	tests := []struct {
		name  string
		field string // extra field on the broadcast
		want  string
	}{
		{name: "profile present", field: `,"profile":"cloud"`, want: "cloud"},
		{name: "ollama profile", field: `,"profile":"ollama"`, want: "ollama"},
		{
			// A motoko predating the profile field must not break, and must not
			// fabricate a value: absent means "cannot assert", not "mismatch".
			name: "absent on older motoko", field: "", want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "session.jsonl")
			lines := []string{
				`{"schema_version":"1","session_id":"s","type":"session_start","task":"t","model":"m"}`,
				`{"schema_version":"1","session_id":"s","type":"runtime_config_resolved","model":"m","step_budget":60,"system_md":"set"` + tt.field + `}`,
				`{"schema_version":"1","session_id":"s","type":"run_summary","model":"m","finish_reason":"stop","steps_executed":1,"duration_ms":1000}`,
			}
			if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			res, err := parseSessionJSONL(p)
			if err != nil {
				t.Fatalf("parseSessionJSONL: %v", err)
			}

			got, _ := res.ProviderData["resolved_profile"].(string)
			if got != tt.want {
				t.Errorf("ProviderData[resolved_profile] = %q, want %q", got, tt.want)
			}

			// The pre-existing delivery signal must keep working.
			if sys, _ := res.ProviderData["system_md"].(string); sys != "set" {
				t.Errorf("system_md = %q, want set (M4 must not disturb M-RIG-RELIABILITY)", sys)
			}
		})
	}
}
