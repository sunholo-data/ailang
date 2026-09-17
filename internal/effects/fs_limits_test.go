package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-V1-MEMORY-FOOTPRINT M3 (D-C): FS reads honour EffEnv.FSMaxBytes. 0 keeps
// the CLI unbounded; serve-api sets it to its upload cap. The guard is the
// POST-READ length check, not the stat: a pseudo-file reports size 0 and a
// file can grow between stat and read, and a silently truncated string would
// replace host OOM with data corruption.

func fsCtx(t *testing.T, capBytes int64) (*EffContext, string) {
	t.Helper()
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))
	ctx.Env.FSMaxBytes = capBytes
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("d", 1000)), 0o644); err != nil {
		t.Fatal(err)
	}
	return ctx, path
}

func TestFSReadFileUnboundedByDefault(t *testing.T) {
	ctx, path := fsCtx(t, 0)
	v, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(*eval.StringValue).Value; len(got) != 1000 {
		t.Fatalf("len %d", len(got))
	}
}

func TestFSReadFileAboveCapIsATypedError(t *testing.T) {
	ctx, path := fsCtx(t, 999)
	_, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: path}})
	if err == nil || !strings.Contains(err.Error(), "E_FS_FILE_TOO_LARGE") {
		t.Fatalf("want E_FS_FILE_TOO_LARGE, got %v", err)
	}
	if !strings.Contains(err.Error(), "1000") || !strings.Contains(err.Error(), "999") {
		t.Fatalf("error must name the size and the cap: %v", err)
	}
	// Exactly at the cap is fine.
	ctx.Env.FSMaxBytes = 1000
	if _, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: path}}); err != nil {
		t.Fatalf("at-cap read failed: %v", err)
	}
}

func TestFSReadFileResultAndBytesAboveCapReturnErr(t *testing.T) {
	ctx, path := fsCtx(t, 10)
	for _, op := range []string{"readFileResult", "readFileBytes"} {
		v, err := Call(ctx, "FS", op, []eval.Value{&eval.StringValue{Value: path}})
		if err != nil {
			t.Fatalf("%s: Go error %v; want Err value", op, err)
		}
		assertResultErrContains(t, v, "E_FS_FILE_TOO_LARGE")
	}
}

// The stat is only an early exit. readCapped must reject on what it READ.
func TestReadCappedRejectsOnReadLengthNotStat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grow.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("g", 50)), 0o644); err != nil {
		t.Fatal(err)
	}
	// Simulate a file whose stat lies (size 0) or grows: open first, then
	// read through readCappedFrom with a stat size of 0.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := readCappedFrom(f, 0, 20); err == nil || !strings.Contains(err.Error(), "E_FS_FILE_TOO_LARGE") {
		t.Fatalf("50-byte read under a 20-byte cap with stat size 0 must fail: %v", err)
	}
	f2, _ := os.Open(path)
	defer f2.Close()
	data, err := readCappedFrom(f2, 0, 50)
	if err != nil || len(data) != 50 {
		t.Fatalf("at-cap read: %v len %d", err, len(data))
	}
}
