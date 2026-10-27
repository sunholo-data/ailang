package pi

import (
	"context"
	"errors"
	"github.com/sunholo-data/ailang/internal/executor"
	"strings"
	"testing"
)

func TestEvalContainmentNilPreservesPayload(t *testing.T) {
	task := &executor.Task{}
	args, env, err := evalContainmentCommand(task, "pi", []string{"--mode", "json"})
	if err != nil || len(env) != 0 || strings.Join(args, " ") != "pi --mode json" {
		t.Fatalf("nil changed payload: %v %v %v", args, env, err)
	}
	flags, err := evalContainmentArgs(task)
	if err != nil || len(flags) != 0 {
		t.Fatalf("nil changed args: %v %v", flags, err)
	}
}
func TestEvalContainmentRefusesIgnoredIsolation(t *testing.T) {
	task := &executor.Task{EvalContainment: &executor.EvalContainment{PolicyPath: "/missing", PolicyDigest: strings.Repeat("a", 64)}}
	for _, setup := range []func(){func() {}, func() { task.IsolateFromAmbientContext = true }, func() { task.IsolateFromProjectExtensions = true; task.AllowedTools = []string{"Read", "AilangRun"} }} {
		setup()
		if _, err := evalContainmentArgs(task); err == nil {
			t.Fatal("accepted unsupported/unbound containment request")
		}
	}
}

func TestEvalContainmentDispatchClosesBeforeModelAndCLI(t *testing.T) {
	e := &PiExecutor{piPath: "MUST-NOT-START", model: ""}
	task := &executor.Task{EvalContainment: &executor.EvalContainment{}}
	for _, call := range []func() (*executor.Result, error){func() (*executor.Result, error) { return e.Execute(context.Background(), task) }, func() (*executor.Result, error) { return e.ExecuteStreaming(context.Background(), task, nil) }} {
		result, err := call()
		if result != nil || !errors.Is(err, executor.ErrEvalContainmentAdmissionClosed) {
			t.Fatalf("expected admissionclosed before modelsetup/CLI probe: %v %v", result, err)
		}
		var typed *executor.EvalContainmentAdmissionError
		if !errors.As(err, &typed) || typed.Backend != "pi" {
			t.Fatalf("untyped admissionerror: %v", err)
		}
	}
}
