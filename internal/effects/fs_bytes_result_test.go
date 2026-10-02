package effects

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// readFileRaw / writeFileBytesResult: raw bytes, failures as Err.

// binaryPayload is deliberately not valid UTF-8 and contains NUL, so a
// string round-trip or base64 detour would be caught.
var binaryPayload = []byte{0x00, 0xff, 0xfe, 0x80, 'a', '\n', 0xc3, 0x28, 0x00}

func TestFSReadFileRaw_ReturnsBytesUnchanged(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	path := filepath.Join(t.TempDir(), "blob.bin")
	if err := os.WriteFile(path, binaryPayload, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Call(ctx, "FS", "readFileRaw", []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	got, ok := assertResultOk(t, res).(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected BytesValue inside Ok, got %T", assertResultOk(t, res))
	}
	if !bytes.Equal(got.Value, binaryPayload) {
		t.Errorf("readFileRaw = %x, want %x", got.Value, binaryPayload)
	}
}

func TestFSReadFileRaw_MissingFileIsErr(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	res, err := Call(ctx, "FS", "readFileRaw", []eval.Value{&eval.StringValue{Value: filepath.Join(t.TempDir(), "nope.bin")}})
	if err != nil {
		t.Fatalf("missing file must be Err, not a Go error: %v", err)
	}
	assertResultErrContains(t, res, "cannot read file")
}

func TestFSReadFileRaw_RespectsMaxBytes(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	ctx.Env.FSMaxBytes = 4
	path := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(path, binaryPayload, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Call(ctx, "FS", "readFileRaw", []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("over-limit read must be Err, not a Go error: %v", err)
	}
	if tv := res.(*eval.TaggedValue); tv.CtorName != "Err" {
		t.Errorf("over-limit read = %s, want Err", tv.CtorName)
	}
}

func TestFSWriteFileBytesResult_RoundTrip(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	path := filepath.Join(t.TempDir(), "out.bin")
	res, err := Call(ctx, "FS", "writeFileBytesResult", []eval.Value{&eval.StringValue{Value: path}, &eval.BytesValue{Value: binaryPayload}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if _, ok := assertResultOk(t, res).(*eval.UnitValue); !ok {
		t.Fatalf("expected Ok(())")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binaryPayload) {
		t.Errorf("file = %x, want %x", got, binaryPayload)
	}
	// Truncates an existing file.
	if _, err := Call(ctx, "FS", "writeFileBytesResult", []eval.Value{&eval.StringValue{Value: path}, &eval.BytesValue{Value: []byte{1}}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, []byte{1}) {
		t.Errorf("second write did not truncate: %x", got)
	}
}

// The failure writeFileBytes panics on (missing parent directory) is an Err here.
func TestFSWriteFileBytesResult_MissingParentIsErr(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	path := filepath.Join(t.TempDir(), "no", "such", "dir", "out.bin")
	if _, err := Call(ctx, "FS", "writeFileBytes", []eval.Value{&eval.StringValue{Value: path}, &eval.BytesValue{Value: binaryPayload}}); err == nil {
		t.Fatal("precondition: writeFileBytes must fail on a missing parent")
	}
	res, err := Call(ctx, "FS", "writeFileBytesResult", []eval.Value{&eval.StringValue{Value: path}, &eval.BytesValue{Value: binaryPayload}})
	if err != nil {
		t.Fatalf("missing parent must be Err, not a Go error: %v", err)
	}
	assertResultErrContains(t, res, "cannot write file")
}

func TestFSWriteFileBytesResult_DenyWriteIsErr(t *testing.T) {
	sandbox := t.TempDir()
	if real, err := filepath.EvalSymlinks(sandbox); err == nil {
		sandbox = real
	}
	if err := os.WriteFile(filepath.Join(sandbox, "Makefile"), []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = sandbox
	ctx.Env.DenyWrite = []string{"Makefile"}
	ctx.Grant(NewCapability("FS"))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	res, err := Call(ctx, "FS", "writeFileBytesResult", []eval.Value{&eval.StringValue{Value: "Makefile"}, &eval.BytesValue{Value: []byte("x")}})
	if err != nil {
		t.Fatalf("protected path must be Err, not a Go error: %v", err)
	}
	assertResultErrContains(t, res, "E_FS_PROTECTED")
	if got, _ := os.ReadFile(filepath.Join(sandbox, "Makefile")); string(got) != "orig" {
		t.Errorf("protected file was modified: %q", got)
	}
}

func TestFSBytesResult_RequireFSCapability(t *testing.T) {
	ctx := NewEffContext(nil) // no FS grant
	path := filepath.Join(t.TempDir(), "x.bin")
	for _, c := range []struct {
		op   string
		args []eval.Value
	}{
		{"readFileRaw", []eval.Value{&eval.StringValue{Value: path}}},
		{"writeFileBytesResult", []eval.Value{&eval.StringValue{Value: path}, &eval.BytesValue{Value: []byte("x")}}},
	} {
		if _, err := Call(ctx, "FS", c.op, c.args); err == nil || !strings.Contains(err.Error(), "FS") {
			t.Errorf("%s without FS capability: want capability error, got %v", c.op, err)
		}
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("writeFileBytesResult wrote a file without the FS capability")
	}
}
