package eval

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// Pins for the prelude capability gate added by 9504393d0 ("prelude println
// bypassed the capability system").
//
// Why these exist: the fix shipped `requireCap` and gated the prelude `println`
// builtin on IO, and NOTHING in the repo exercised either. Measured at
// 5a3a59126 by neutering the call site (`if err := requireCap(e, "IO"); false &&
// err != nil`): the mutant built and every package under ./internal/... and
// ./cmd/ailang/... stayed green (100 ok, rc=0). A capability refusal that no
// test can red is not a gate.
//
// The observable is deliberately narrow (rule 3i): each arm asserts the
// capability NAME requested and whether the effect actually reached stdout,
// not merely that some error came back. "An error was returned" is satisfied by
// any failure; "IO was requested and nothing was written" is satisfied only by
// this gate.

// errCapDenied is a sentinel so an arm can assert THIS refusal rather than any
// error (errors.Is, not a substring match).
var errCapDenied = errors.New("prelude_cap_gate_test: capability denied")

// fakeCapCtx is a minimal effect context implementing the locally-declared
// capRequirer interface. internal/eval cannot import internal/effects (import
// cycle) — which is exactly why capRequirer is declared locally, so the fake
// stands in for *effects.EffContext here.
type fakeCapCtx struct {
	grant     bool
	requested []string
}

func (f *fakeCapCtx) RequireCap(name string) error {
	f.requested = append(f.requested, name)
	if f.grant {
		return nil
	}
	return errCapDenied
}

// nonRequirerCtx is an effect context that does NOT implement capRequirer.
// requireCap documents this as "no capability system wired" — ungated.
type nonRequirerCtx struct{}

// captureStdout runs fn with os.Stdout redirected and returns what was written.
// The prelude println writes with fmt.Print directly, so stdout is the only
// place the effect is observable — and observing it is what discriminates
// "refused before the effect" from "errored after printing".
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	os.Stdout = saved
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return out
}

// preludePrintln pulls the prelude println builtin out of an environment.
// It is registered by registerBuiltins, so its ABSENCE is itself a failure
// worth naming rather than a nil-deref later.
func preludePrintln(t *testing.T, env *Environment) *BuiltinFunction {
	t.Helper()
	v, ok := env.Get("println")
	if !ok {
		t.Fatal("prelude println is not registered — registerBuiltins did not run")
	}
	fn, ok := v.(*BuiltinFunction)
	if !ok {
		t.Fatalf("prelude println is %T, want *BuiltinFunction", v)
	}
	return fn
}

// gatedConstructors is every production site that builds a CoreEvaluator and
// registers prelude builtins against it. Both must gate; pinning only one
// leaves the other free to regress (Principle 3 — sweep the sites, do not patch
// the one you happened to find). Fork is the third site and has its own test;
// NewTypedEvaluator is the fourth and passes nil, which is an ungated path.
var gatedConstructors = []struct {
	name string
	ctor func() *CoreEvaluator
}{
	{"NewCoreEvaluator", NewCoreEvaluator},
	{"NewCoreEvaluatorWithRegistry", func() *CoreEvaluator {
		return NewCoreEvaluatorWithRegistry(types.NewDictionaryRegistry())
	}},
}

func TestPreludePrintlnRequiresIOCapability(t *testing.T) {
	for _, tc := range gatedConstructors {
		t.Run(tc.name, func(t *testing.T) {
			deny := &fakeCapCtx{grant: false}
			e := tc.ctor()
			e.SetEffContext(deny)

			var got Value
			var err error
			out := captureStdout(t, func() {
				got, err = preludePrintln(t, e.Env()).Fn([]Value{&StringValue{Value: "leaked"}})
			})

			if !errors.Is(err, errCapDenied) {
				t.Fatalf("println with IO denied: err = %v, want errCapDenied", err)
			}
			if got != nil {
				t.Errorf("println with IO denied returned %v, want nil value", got)
			}
			// The refusal must precede the effect: a gate that returns the error
			// AFTER writing has not stopped anything.
			if out != "" {
				t.Errorf("println wrote %q to stdout despite refusing IO — the effect escaped the gate", out)
			}
			// Assert WHICH capability was demanded. Under a `"FS"` mutant the
			// assertions above still pass (the fake denies everything); this is
			// the assertion that pins IO.
			if len(deny.requested) != 1 || deny.requested[0] != "IO" {
				t.Errorf("capabilities requested = %v, want exactly [IO]", deny.requested)
			}
		})
	}
}

func TestPreludePrintlnAllowedWithIOCapability(t *testing.T) {
	for _, tc := range gatedConstructors {
		t.Run(tc.name, func(t *testing.T) {
			grant := &fakeCapCtx{grant: true}
			e := tc.ctor()
			e.SetEffContext(grant)

			var got Value
			var err error
			out := captureStdout(t, func() {
				got, err = preludePrintln(t, e.Env()).Fn([]Value{&StringValue{Value: "allowed"}})
			})

			if err != nil {
				t.Fatalf("println with IO granted: unexpected error %v", err)
			}
			if _, ok := got.(*UnitValue); !ok {
				t.Errorf("println returned %T, want *UnitValue", got)
			}
			if !strings.Contains(out, "allowed") {
				t.Errorf("println wrote %q, want it to contain %q", out, "allowed")
			}
			if len(grant.requested) != 1 || grant.requested[0] != "IO" {
				t.Errorf("capabilities requested = %v, want exactly [IO]", grant.requested)
			}
		})
	}
}

// The three ungated paths requireCap documents. They are pinned not because
// they enforce anything but because each is a `return nil` branch that a
// future "fail closed everywhere" change would silently flip — breaking the
// REPL / TypedEvaluator / SimpleEvaluator, which never granted caps at all.
func TestPreludePrintlnUngatedPaths(t *testing.T) {
	t.Run("no effect context", func(t *testing.T) {
		e := NewCoreEvaluator() // effContext stays nil
		out := captureStdout(t, func() {
			if _, err := preludePrintln(t, e.Env()).Fn([]Value{&StringValue{Value: "repl"}}); err != nil {
				t.Fatalf("unexpected error with no effect context: %v", err)
			}
		})
		if !strings.Contains(out, "repl") {
			t.Errorf("stdout = %q, want it to contain %q", out, "repl")
		}
	})

	t.Run("effect context is not a capRequirer", func(t *testing.T) {
		e := NewCoreEvaluator()
		e.SetEffContext(&nonRequirerCtx{})
		out := captureStdout(t, func() {
			if _, err := preludePrintln(t, e.Env()).Fn([]Value{&StringValue{Value: "noncap"}}); err != nil {
				t.Fatalf("unexpected error for a non-capRequirer context: %v", err)
			}
		})
		if !strings.Contains(out, "noncap") {
			t.Errorf("stdout = %q, want it to contain %q", out, "noncap")
		}
	})

	t.Run("nil evaluator (TypedEvaluator path)", func(t *testing.T) {
		// The real production site: NewTypedEvaluator calls
		// registerBuiltins(env, nil). Driving the constructor rather than
		// re-creating its binding is what makes this arm notice if that call
		// site ever starts passing a real evaluator.
		te := NewTypedEvaluator(false, 0, false)
		out := captureStdout(t, func() {
			if _, err := preludePrintln(t, te.env).Fn([]Value{&StringValue{Value: "typed"}}); err != nil {
				t.Fatalf("unexpected error for a nil evaluator: %v", err)
			}
		})
		if !strings.Contains(out, "typed") {
			t.Errorf("stdout = %q, want it to contain %q", out, "typed")
		}
	})
}

// Fork re-registers builtins so they close over the FORK, not the parent —
// the comment at eval_evaluator.go's registerBuiltins(env, forked) call. Each
// request goroutine gets its own Fork, so a fork whose context denies IO must
// refuse even though the parent granted it. Under `registerBuiltins(env, e)`
// the fork's println consults the parent and this arm reds.
func TestPreludePrintlnForkGatesOnForkContext(t *testing.T) {
	grant := &fakeCapCtx{grant: true}
	parent := NewCoreEvaluator()
	parent.SetEffContext(grant)

	forked := parent.Fork()
	deny := &fakeCapCtx{grant: false}
	forked.SetEffContext(deny)

	var forkErr error
	forkOut := captureStdout(t, func() {
		_, forkErr = preludePrintln(t, forked.Env()).Fn([]Value{&StringValue{Value: "fork"}})
	})
	if !errors.Is(forkErr, errCapDenied) {
		t.Fatalf("forked println: err = %v, want errCapDenied (the fork's own context must gate it)", forkErr)
	}
	if forkOut != "" {
		t.Errorf("forked println wrote %q despite refusing IO", forkOut)
	}
	if len(deny.requested) != 1 || deny.requested[0] != "IO" {
		t.Errorf("fork requested %v, want exactly [IO]", deny.requested)
	}

	// Control: the parent is unaffected, so the arm above measures the fork's
	// isolation rather than a global flip.
	parentOut := captureStdout(t, func() {
		if _, err := preludePrintln(t, parent.Env()).Fn([]Value{&StringValue{Value: "parent"}}); err != nil {
			t.Fatalf("parent println after fork: unexpected error %v", err)
		}
	})
	if !strings.Contains(parentOut, "parent") {
		t.Errorf("parent stdout = %q, want it to contain %q", parentOut, "parent")
	}
}
