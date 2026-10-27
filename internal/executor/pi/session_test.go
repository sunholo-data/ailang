package pi

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/nativesession"
)

func TestNativeSessionArgs(t *testing.T) {
	task := &executor.Task{ID: "cloud-task"}
	t.Setenv("CLOUD_RUN_EXECUTION", "")
	args, err := buildPiArgsWithSession("model", task, "prompt", "generated-id")
	if err != nil || !slices.Contains(args, "--no-session") || slices.Contains(args, "--session-dir") {
		t.Fatalf("local args %v: %v", args, err)
	}
	t.Setenv("CLOUD_RUN_EXECUTION", "job")
	args, err = buildPiArgsWithSession("model", task, "prompt", "generated-id")
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(args, "--no-session") {
		t.Fatal(args)
	}
	for flag, want := range map[string]string{"--session-dir": nativesession.PiDir(task.ID), "--session-id": "generated-id"} {
		i := slices.Index(args, flag)
		if i < 0 || i+1 >= len(args) || args[i+1] != want {
			t.Fatalf("%s in %v", flag, args)
		}
	}
}

func TestNativeSessionBankedOnSuccessAndFailure(t *testing.T) {
	t.Setenv("CLOUD_RUN_EXECUTION", "job")
	t.Setenv("TMPDIR", t.TempDir())
	dir := nativesession.PiDir("d")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "stamp_native-id.jsonl")
	if err := os.WriteFile(path, []byte("native session"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, events := range [][]string{loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson"), {`{"type":"unknown_event"}`}} {
		result := runFake(t, events)
		if result.ProviderData["pi_session_jsonl"] != path {
			t.Fatal(result.ProviderData)
		}
		if _, ok := result.ProviderData["pi_events"]; !ok {
			t.Fatal("lost provider evidence")
		}
	}
}
