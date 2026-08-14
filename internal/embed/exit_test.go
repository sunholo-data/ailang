package embed

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// newExitTestEngine returns an engine rooted at the repo, with the IO
// capability granted so that exit() is reachable at all.
//
// Without the grant the effect layer refuses before ioExit is entered
// ("effect 'IO' requires capability"), which would make every arm below pass
// for the wrong reason — the sentinel would never be raised.
func newExitTestEngine(t *testing.T) *Engine {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}
	t.Setenv("AILANG_STDLIB_PATH", root)

	engine := New(root)
	t.Cleanup(func() { _ = engine.Close() })

	effCtx := effects.NewEffContext(nil)
	effCtx.Grant(effects.NewCapability("IO"))
	engine.runtime.GetEvaluator().SetEffContext(effCtx)

	return engine
}

// TestCallReportsExitAsTypedError is the headline arm for #691: an embedded
// module calling exit(1) must not take the host down. Before the fix this did
// not fail — it panicked the test binary through embed.go's CallEntrypoint.
func TestCallReportsExitAsTypedError(t *testing.T) {
	engine := newExitTestEngine(t)

	val, err := engine.Call("internal/embed/testdata/exit_nonzero", "main")
	if err == nil {
		t.Fatal("Call on a module calling exit(1) returned nil error; want *ExitError")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Call error = %T (%v); want *ExitError", err, err)
	}
	if exitErr.Code != 1 {
		t.Errorf("ExitError.Code = %d, want 1", exitErr.Code)
	}
	if val != nil {
		t.Errorf("Call value = %v, want nil on exit", val)
	}
}

// TestCallPreserveFloatsReportsExitAsTypedError covers the SECOND call site.
// This repo's named recurring failure shape is guard-the-helper-miss-the-
// call-site, so the two entry points are pinned separately rather than
// assumed equivalent.
func TestCallPreserveFloatsReportsExitAsTypedError(t *testing.T) {
	engine := newExitTestEngine(t)

	val, err := engine.CallPreserveFloats("internal/embed/testdata/exit_nonzero", "main")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("CallPreserveFloats error = %T (%v); want *ExitError", err, err)
	}
	if exitErr.Code != 1 {
		t.Errorf("ExitError.Code = %d, want 1", exitErr.Code)
	}
	if val != nil {
		t.Errorf("CallPreserveFloats value = %v, want nil on exit", val)
	}
}

// TestCallJSONReportsExitAsTypedError pins the transitive consumer: CallJSON
// delegates to Call, so it inherits the contract rather than needing its own
// recover. Asserted rather than assumed.
func TestCallJSONReportsExitAsTypedError(t *testing.T) {
	engine := newExitTestEngine(t)

	out, err := engine.CallJSON("internal/embed/testdata/exit_nonzero", "main", nil)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("CallJSON error = %T (%v); want *ExitError", err, err)
	}
	if exitErr.Code != 1 {
		t.Errorf("ExitError.Code = %d, want 1", exitErr.Code)
	}
	if out != nil {
		t.Errorf("CallJSON output = %q, want nil on exit", out)
	}
}

// TestCallReportsExitZeroAsTypedError pins the contract DECISION for #691:
// exit(0) is an *ExitError too, not a nil error. The CLI's batch path maps
// exit(0) onto success because it owns a process; the embed layer does not,
// and a nil error here would be indistinguishable from a unit return.
func TestCallReportsExitZeroAsTypedError(t *testing.T) {
	engine := newExitTestEngine(t)

	_, err := engine.Call("internal/embed/testdata/exit_zero", "main")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Call error = %T (%v); want *ExitError", err, err)
	}
	if exitErr.Code != 0 {
		t.Errorf("ExitError.Code = %d, want 0", exitErr.Code)
	}
}

// TestCallWithoutExitIsUnaffected is the mechanism-removed control (rule 3d).
// Same engine, same call shape, exit() removed: a green here is what makes the
// arms above evidence about exit() rather than about the harness.
func TestCallWithoutExitIsUnaffected(t *testing.T) {
	engine := newExitTestEngine(t)

	val, err := engine.Call("internal/embed/testdata/no_exit", "main")
	if err != nil {
		t.Fatalf("Call on an exit-free module errored: %v", err)
	}
	got, err := ToInt(val)
	if err != nil {
		t.Fatalf("ToInt: %v", err)
	}
	if got != 42 {
		t.Errorf("Call value = %d, want 42", got)
	}
}

// --- recoverProgramExit branch arms (unit-level, no module runtime) ---

// TestRecoverProgramExitPassesThroughValueAndError covers the no-panic branch.
func TestRecoverProgramExitPassesThroughValueAndError(t *testing.T) {
	want := &eval.IntValue{Value: 7}
	wantErr := errors.New("ordinary failure")

	val, err := recoverProgramExit(func() (eval.Value, error) { return want, wantErr })
	if val != want {
		t.Errorf("value = %v, want %v", val, want)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// TestRecoverProgramExitMapsSentinel covers the exit branch in isolation.
func TestRecoverProgramExitMapsSentinel(t *testing.T) {
	val, err := recoverProgramExit(func() (eval.Value, error) {
		panic(&eval.EvalExitCode{Code: 3})
	})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T (%v); want *ExitError", err, err)
	}
	if exitErr.Code != 3 {
		t.Errorf("ExitError.Code = %d, want 3", exitErr.Code)
	}
	if val != nil {
		t.Errorf("value = %v, want nil", val)
	}
	if got, want := exitErr.Error(), "program called exit(3)"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestRecoverProgramExitReRaisesOtherPanics covers the third branch: a real
// crash must stay loud and arrive unchanged, not be swallowed into an error.
// Identity is asserted, not just non-nil-ness — a recover that re-panicked a
// wrapped or substituted value would still "panic" and pass a weaker check.
func TestRecoverProgramExitReRaisesOtherPanics(t *testing.T) {
	sentinel := fmt.Errorf("a real crash")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a non-exit panic was swallowed; want it re-raised")
		}
		if r != any(sentinel) {
			t.Fatalf("re-raised panic value = %#v, want the original %#v", r, sentinel)
		}
	}()

	_, _ = recoverProgramExit(func() (eval.Value, error) { panic(sentinel) })
	t.Fatal("unreachable: recoverProgramExit returned instead of re-panicking")
}
