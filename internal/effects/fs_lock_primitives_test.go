package effects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// mkdirResult is the atomic try-lock: exactly one caller creates the directory,
// every later one gets Err. mkdirAllResult cannot express this (Ok on exists),
// which is why a lock client had to exec("mkdir") (Daneel, 2026-09-11).
func TestFS_MkdirResult_TryLock(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))
	lock := filepath.Join(t.TempDir(), "rig.lock.d")
	arg := []eval.Value{&eval.StringValue{Value: lock}}

	first, err := Call(ctx, "FS", "mkdirResult", arg)
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, first)

	second, err := Call(ctx, "FS", "mkdirResult", arg)
	if err != nil {
		t.Fatal(err)
	}
	assertResultErrContains(t, second, "exist")

	// Control: mkdirAllResult on the same existing directory is Ok — the two
	// primitives differ exactly where a lock needs them to.
	all, err := Call(ctx, "FS", "mkdirAllResult", arg)
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, all)

	// Parent missing: Err, not a silent mkdir -p.
	deep, err := Call(ctx, "FS", "mkdirResult", []eval.Value{&eval.StringValue{Value: filepath.Join(lock, "no", "parent")}})
	if err != nil {
		t.Fatal(err)
	}
	assertResultErrContains(t, deep, "cannot create directory")
}

func TestFS_RemoveDirResult_EmptyOnly(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))
	base := t.TempDir()
	lock := filepath.Join(base, "lock.d")
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	call := func(p string) eval.Value {
		v, err := Call(ctx, "FS", "removeDirResult", []eval.Value{&eval.StringValue{Value: p}})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	// Non-empty: Err and the contents survive — never recursive.
	inner := filepath.Join(lock, "owner")
	if err := os.WriteFile(inner, []byte("123"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertResultErrContains(t, call(lock), "cannot remove directory")
	if _, err := os.Stat(inner); err != nil {
		t.Fatalf("removeDirResult on a non-empty dir deleted its contents: %v", err)
	}

	// Not a directory: Err, and the file survives.
	assertResultErrContains(t, call(inner), "not a directory")
	if _, err := os.Stat(inner); err != nil {
		t.Fatalf("removeDirResult deleted a file: %v", err)
	}

	// Empty: Ok, gone; again: Err (missing).
	if err := os.Remove(inner); err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, call(lock))
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("lock dir still present after removeDirResult: %v", err)
	}
	assertResultErrContains(t, call(lock), "cannot remove directory")
}

func TestEnv_GetPid(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	v, err := Call(ctx, "Env", "getPid", []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(*eval.IntValue).Value; got != os.Getpid() {
		t.Errorf("getPid = %d, want %d", got, os.Getpid())
	}
	// Capability-gated like every Env op.
	if _, err := Call(NewEffContext(nil), "Env", "getPid", []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Error("getPid without Env capability should fail")
	}
}
