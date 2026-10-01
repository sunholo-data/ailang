package motoko

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestIsLaneTask(t *testing.T) {
	lane, _ := executor.ProfileTools(executor.ToolProfileAILANGOnly)
	reversed := make([]string, len(lane))
	for i, v := range lane {
		reversed[len(lane)-1-i] = v
	}
	for _, tc := range []struct {
		name  string
		tools []string
		want  bool
	}{
		{"nil", nil, false},
		{"eval harness list", []string{"Bash", "Read", "Write", "Edit", "Grep"}, false},
		{"ailang_only", lane, true},
		{"ailang_only, any order", reversed, true},
		{"ailang_only minus one", lane[1:], false},
	} {
		if got := isLaneTask(&executor.Task{AllowedTools: tc.tools}); got != tc.want {
			t.Errorf("%s: isLaneTask = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// fakeRunner answers the resolver call with `config` and the probe with `probeExit`.
func fakeRunner(config string, probeExit int) laneRunner {
	return func(_ context.Context, _ string, _ []string, _ string, args ...string) (string, int, error) {
		for _, a := range args {
			if a == "print_config_json" {
				return "WARNING something\n" + config, 0, nil
			}
		}
		return "", probeExit, nil
	}
}

func TestCheckLaneProfile(t *testing.T) {
	good := `{"extensions":{"order":["ailang_policy","empty_stop_guard","compaction_ai"],"strict":true},"tools":{"hybrid":false}}`
	for _, tc := range []struct {
		name, config, wantErr string
	}{
		{"lane profile", good, ""},
		{"not first", `{"extensions":{"order":["empty_stop_guard","ailang_policy"],"strict":true},"tools":{"hybrid":false}}`, "needs \"ailang_policy\" first"},
		{"strict off", `{"extensions":{"order":["ailang_policy"],"strict":false},"tools":{"hybrid":false}}`, "extensions.strict is false"},
		{"hybrid on", `{"extensions":{"order":["ailang_policy"],"strict":true},"tools":{"hybrid":true}}`, "tools.hybrid is on"},
		{"bypassing extension", `{"extensions":{"order":["ailang_policy","compose"],"strict":true},"tools":{"hybrid":false}}`, "not on the lane allowlist"},
		{"profile not found", `{"extensions":{"order":[],"strict":false},"tools":{"hybrid":true}}`, "empty extensions.order"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkLaneProfile(context.Background(), fakeRunner(tc.config, 2), "/repo", "/ws", "ailang_only")
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected refusal: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("want refusal containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCheckStrictEnforced(t *testing.T) {
	repoWith := func() string {
		r := t.TempDir()
		if err := os.MkdirAll(filepath.Join(r, "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r, "scripts", "verify_strict_extensions.ail"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return r
	}
	if err := checkStrictEnforced(context.Background(), fakeRunner("", 2), repoWith()); err != nil {
		t.Fatalf("enforcing build refused: %v", err)
	}
	if err := checkStrictEnforced(context.Background(), fakeRunner("", 0), repoWith()); err == nil || !strings.Contains(err.Error(), "does not enforce") {
		t.Fatalf("a build whose probe starts was accepted: %v", err)
	}
	if err := checkStrictEnforced(context.Background(), fakeRunner("", 2), t.TempDir()); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("a build without the probe script was accepted: %v", err)
	}
}

func TestCheckLaneTask_Policy(t *testing.T) {
	e := &MotokoExecutor{motokoRepo: "/repo"}
	run := fakeRunner(`{"extensions":{"order":["ailang_policy"],"strict":true},"tools":{"hybrid":false}}`, 2)
	if err := e.checkLaneTask(context.Background(), run, &executor.Task{}, "ailang_only"); err == nil || !strings.Contains(err.Error(), "program policy") {
		t.Fatalf("no policy accepted: %v", err)
	}
	dir := t.TempDir()
	trusted := filepath.Join(dir, "trusted.toml")
	if err := os.WriteFile(trusted, []byte("security_mode = \"trusted_host\"\nallowed_caps = [\"IO\"]\nentry = \"main\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.checkLaneTask(context.Background(), run, &executor.Task{PolicyPath: trusted}, "ailang_only"); err == nil || !strings.Contains(err.Error(), "restricted") {
		t.Fatalf("trusted_host policy accepted: %v", err)
	}
}

func writeSession(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLaneSessionViolation(t *testing.T) {
	start := `{"type":"session_start","loaded_extensions":["ailang_policy","empty_stop_guard"]}`
	runStart := `{"type":"session_start","mode":"v2","run_id":"r0"}`
	ok := writeSession(t, start, runStart,
		`{"type":"native_tool_results","results":[{"tool":"ailang_write","exit_code":0}]}`,
		`{"type":"native_tool_results","results":[{"exit_code":1,"payload":{"denied_by_policy":true,"tool":"BashExec"}}]}`)
	if v := laneSessionViolation(ok); v != "" {
		t.Fatalf("clean lane session flagged: %s", v)
	}
	native := writeSession(t, start, `{"type":"native_tool_results","results":[{"tool":"BashExec","exit_code":0}]}`)
	if v := laneSessionViolation(native); !strings.Contains(v, "BashExec") {
		t.Fatalf("an executed native tool was not flagged: %q", v)
	}
	wrongFirst := writeSession(t, `{"type":"session_start","loaded_extensions":["empty_stop_guard","ailang_policy"]}`)
	if v := laneSessionViolation(wrongFirst); !strings.Contains(v, "first") {
		t.Fatalf("a session without ailang_policy first was not flagged: %q", v)
	}
	if v := laneSessionViolation(writeSession(t, runStart)); !strings.Contains(v, "session_start") {
		t.Fatalf("a session without session_start was not flagged: %q", v)
	}
}
