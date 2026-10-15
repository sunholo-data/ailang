//go:build !js

package effects

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestProcessSpawnResultFailures(t *testing.T) {
	for _, tc := range []struct {
		name, ctor, detail string
		pc                 *ProcessContext
	}{
		{"missing", "NotFound", "absent", &ProcessContext{HasAllowlist: true, Allowlist: map[string]string{"absent": ""}}},
		{"denied", "NotAllowed", "absent", &ProcessContext{HasAllowlist: true}},
		{"confined", "NotAllowed", "confined Process", &ProcessContext{Confined: true}},
		{"subcommand", "NotAllowed", "absent denied", &ProcessContext{HasAllowlist: true, Allowlist: map[string]string{"absent": "unused"}, Subcommands: map[string][][]string{"absent": {{"allowed"}}}}},
		{"start", "SpawnFailed", "", &ProcessContext{HasAllowlist: true, Allowlist: map[string]string{"absent": filepath.Join(t.TempDir(), "missing")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewEffContext(nil)
			ctx.Grant(NewCapability("Process"))
			ctx.Process = tc.pc
			argv := spawnArgs("absent")
			if tc.name == "subcommand" {
				argv[1] = &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "denied"}}}
			}
			v, err := Call(ctx, "Process", "spawnProcessResult", argv)
			if err != nil {
				t.Fatal(err)
			}
			assertSpawnErr(t, v, tc.ctor, tc.detail)
			owner := ctx.executionWorkerOwner()
			owner.mu.Lock()
			defer owner.mu.Unlock()
			if len(owner.pending) != 0 || len(owner.workers) != 0 {
				t.Fatal("failed spawn retained admission")
			}
		})
	}
}

func assertSpawnErr(t *testing.T, v eval.Value, ctor, detail string) {
	t.Helper()
	r, ok := v.(*eval.TaggedValue)
	if !ok || r.CtorName != "Err" || len(r.Fields) != 1 {
		t.Fatalf("expected Err, got %v", v)
	}
	e, ok := r.Fields[0].(*eval.TaggedValue)
	if !ok || e.CtorName != ctor || e.TypeName != "ProcessError" || len(e.Fields) != 1 {
		t.Fatalf("expected %s, got %v", ctor, r)
	}
	if !strings.Contains(e.Fields[0].(*eval.StringValue).Value, detail) {
		t.Fatalf("missing %q: %v", detail, e)
	}
}

func TestProcessSpawnResultAdmissionAndAuthority(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Process = NewProcessContext()
	if _, err := Call(ctx, "Process", "spawnProcessResult", spawnArgs("absent")); err == nil {
		t.Fatal("missing capability accepted")
	}
	ctx.Grant(NewCapability("Process"))
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"worker": "unused"}
	owner := ctx.executionWorkerOwner()
	owner.mu.Lock()
	owner.closing = true
	owner.mu.Unlock()
	v, err := Call(ctx, "Process", "spawnProcessResult", spawnArgs("worker"))
	if err != nil {
		t.Fatal(err)
	}
	assertSpawnErr(t, v, "SpawnFailed", "execution owner is shutting down")
}

func TestProcessSpawnResultOSFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-executable")
	if err := os.WriteFile(path, []byte("invalid executable format"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.Process = NewProcessContext()
	ctx.Grant(NewCapability("Process"))
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"worker": path}
	v, err := Call(ctx, "Process", "spawnProcessResult", spawnArgs("worker"))
	if err != nil {
		t.Fatal(err)
	}
	assertSpawnErr(t, v, "SpawnFailed", "")
}

// The subprocess runs this Go test binary, so lifecycle coverage needs no shell
// or platform-specific executable. EOF finishes the worker normally.
func TestProcessSpawnResultWorker(t *testing.T) {
	if os.Getenv("AILANG_SPAWN_RESULT_TEST_WORKER") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func spawnResultWorker(t *testing.T) (*EffContext, []eval.Value) {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_SPAWN_RESULT_TEST_WORKER", "1")
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	ctx.Process = NewProcessContext()
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"worker": path}
	t.Cleanup(func() {
		if err := ctx.CloseWorkers(); err != nil {
			t.Error(err)
		}
	})
	args := []eval.Value{&eval.StringValue{Value: "worker"}, &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "-test.run=^TestProcessSpawnResultWorker$"}}}}
	return ctx, args
}

func TestProcessSpawnResultUsableHandle(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(fmt.Sprint(cancel), func(t *testing.T) {
			ctx, args := spawnResultWorker(t)
			v, err := Call(ctx, "Process", "spawnProcessResult", args)
			if err != nil {
				t.Fatal(err)
			}
			r := v.(*eval.TaggedValue)
			if r.CtorName != "Ok" {
				t.Fatalf("spawn: %v", v)
			}
			handle := r.Fields[0]
			id, err := extractProcessHandleID(handle)
			if err != nil {
				t.Fatal(err)
			}
			mp, ok := ctx.Process.GetManagedProcess(id)
			if !ok {
				t.Fatal("handle not registered")
			}
			w, err := ProcessWriteStdin(ctx, []eval.Value{handle, &eval.BytesValue{Value: []byte("hello\n")}})
			if err != nil || w.(*eval.TaggedValue).CtorName != "Ok" {
				t.Fatalf("write: %v %v", w, err)
			}
			if cancel && WorkerCancellationSupported() {
				v, err = Call(ctx, "Process", "cancelProcess", []eval.Value{handle})
				if err != nil || v.(*eval.TaggedValue).CtorName != "Ok" {
					t.Fatalf("cancel: %v %v", v, err)
				}
			} else {
				if _, err := ProcessCloseStdin(ctx, []eval.Value{handle}); err != nil {
					t.Fatal(err)
				}
			}
			bounded, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			if err := mp.Join(bounded); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProcessSpawnResultLateAdmissionCleanup(t *testing.T) {
	ctx, args := spawnResultWorker(t)
	admission, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	path := ctx.Process.Allowlist["worker"]
	mp, err := NewManagedProcess(admission.Context(), path, []string{args[1].(*eval.ListValue).Elements[0].(*eval.StringValue).Value})
	if err != nil {
		admission.Abort()
		t.Fatal(err)
	}
	t.Cleanup(mp.Close)
	// Deterministically cross the shutdown gate after start, before Complete.
	owner := ctx.executionWorkerOwner()
	owner.mu.Lock()
	owner.closing = true
	owner.mu.Unlock()
	handle, err := finishProcessSpawn(ctx, admission, mp)
	if handle != nil || err == nil || err.Error() != "execution owner is shutting down" {
		t.Fatalf("late admission: %v %v", handle, err)
	}
	assertSpawnErr(t, processSpawnResult(handle, err), "SpawnFailed", "execution owner is shutting down")
	select {
	case <-mp.Done():
	default:
		t.Fatal("late child not reaped")
	}
	select {
	case <-mp.writerDone:
	default:
		t.Fatal("writer not joined")
	}
	ctx.Process.mu.Lock()
	retained := len(ctx.Process.managed)
	ctx.Process.mu.Unlock()
	owner.mu.Lock()
	pending, workers := len(owner.pending), len(owner.workers)
	owner.mu.Unlock()
	if retained != 0 || pending != 0 || workers != 0 {
		t.Fatalf("retained handles=%d pending=%d workers=%d", retained, pending, workers)
	}
}

func TestProcessSpawnResultPATHAndCancelledAdmission(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	ctx.Process = NewProcessContext()
	t.Setenv("PATH", t.TempDir())
	v, err := Call(ctx, "Process", "spawnProcessResult", spawnArgs("crew-nonexistent-worker"))
	if err != nil {
		t.Fatal(err)
	}
	assertSpawnErr(t, v, "NotFound", "crew-nonexistent-worker")
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"worker": "unused"}
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	ctx.GoCtx = parent
	v, err = Call(ctx, "Process", "spawnProcessResult", spawnArgs("worker"))
	if err != nil {
		t.Fatal(err)
	}
	assertSpawnErr(t, v, "SpawnFailed", "context canceled")
}

func TestProcessSpawnResultMissingContextIsTyped(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	v, err := Call(ctx, "Process", "spawnProcessResult", spawnArgs("absent"))
	if err != nil {
		t.Fatal(err)
	}
	assertSpawnErr(t, v, "NotAllowed", "Process capability not granted")
}
