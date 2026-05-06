package effects

// M-AILANG-FS-RESULT (v0.16.0): Tests for Result-returning fs builtins.
//
// Each handler is paired: success path returns Ok(value), failure path
// returns Err(message). The handlers must NEVER return a Go error for these
// failure modes — that would propagate as a runtime panic and defeat the
// whole purpose. The (eval.Value, error) return shape uses error only for
// arity / type-mismatch cases that the AILANG typechecker prevents at
// compile time.
//
// Split out of fs_test.go (which would otherwise exceed the 800-LOC
// AI-maintainability ceiling) for v0.16.0 release.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// assertResultOk asserts that v is a Result.Ok wrapping the expected payload.
// Returns the inner field for further inspection.
func assertResultOk(t *testing.T, v eval.Value) eval.Value {
	t.Helper()
	tagged, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected *eval.TaggedValue (Result), got %T", v)
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok constructor, got %s with fields %v", tagged.CtorName, tagged.Fields)
	}
	if len(tagged.Fields) != 1 {
		t.Fatalf("expected Ok to wrap 1 field, got %d", len(tagged.Fields))
	}
	return tagged.Fields[0]
}

// assertResultErrContains asserts that v is Result.Err with a message containing substring.
func assertResultErrContains(t *testing.T, v eval.Value, substring string) {
	t.Helper()
	tagged, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected *eval.TaggedValue (Result), got %T", v)
	}
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err constructor, got %s", tagged.CtorName)
	}
	if len(tagged.Fields) != 1 {
		t.Fatalf("expected Err to wrap 1 field, got %d", len(tagged.Fields))
	}
	msg, ok := tagged.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected Err message to be StringValue, got %T", tagged.Fields[0])
	}
	if !strings.Contains(msg.Value, substring) {
		t.Errorf("expected Err message to contain %q, got %q", substring, msg.Value)
	}
}

// --- readFileResult ---

func TestFSReadFileResult_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "ok.txt")
	want := "hello from result"
	if err := os.WriteFile(path, []byte(want), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Call(ctx, "FS", "readFileResult", []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	inner := assertResultOk(t, result)
	got, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue inside Ok, got %T", inner)
	}
	if got.Value != want {
		t.Errorf("expected %q, got %q", want, got.Value)
	}
}

func TestFSReadFileResult_NonexistentFileReturnsErr(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	result, err := Call(ctx, "FS", "readFileResult", []eval.Value{&eval.StringValue{Value: "/this/path/should/not/exist.txt"}})
	if err != nil {
		t.Fatalf("expected no Go error (failure should be wrapped as Err), got %v", err)
	}
	assertResultErrContains(t, result, "cannot read file")
}

// --- writeFileResult ---

func TestFSWriteFileResult_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	content := "result-write content"

	result, err := Call(ctx, "FS", "writeFileResult", []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: content},
	})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	inner := assertResultOk(t, result)
	if _, ok := inner.(*eval.UnitValue); !ok {
		t.Fatalf("expected UnitValue inside Ok, got %T", inner)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("expected %q on disk, got %q", content, string(got))
	}
}

func TestFSWriteFileResult_MissingParentReturnsErr(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "missing", "subdir", "out.txt")

	result, err := Call(ctx, "FS", "writeFileResult", []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "x"},
	})
	if err != nil {
		t.Fatalf("expected no Go error (missing parent should be Err), got %v", err)
	}
	assertResultErrContains(t, result, "cannot write file")
}

// --- appendFileResult ---

func TestFSAppendFileResult_SuccessCreates(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")

	result, err := Call(ctx, "FS", "appendFileResult", []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "first\n"},
	})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	if _, ok := assertResultOk(t, result).(*eval.UnitValue); !ok {
		t.Fatal("expected UnitValue inside Ok")
	}

	result2, _ := Call(ctx, "FS", "appendFileResult", []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "second\n"},
	})
	assertResultOk(t, result2)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\nsecond\n" {
		t.Errorf("unexpected file contents after appends: %q", string(got))
	}
}

func TestFSAppendFileResult_MissingParentReturnsErr(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "missing", "log.txt")

	result, err := Call(ctx, "FS", "appendFileResult", []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "x"},
	})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	assertResultErrContains(t, result, "cannot append to file")
}

// --- removeFileResult ---

func TestFSRemoveFileResult_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	path := filepath.Join(dir, "doomed.txt")
	if err := os.WriteFile(path, []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Call(ctx, "FS", "removeFileResult", []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	if _, ok := assertResultOk(t, result).(*eval.UnitValue); !ok {
		t.Fatal("expected UnitValue inside Ok")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err: %v", err)
	}
}

func TestFSRemoveFileResult_NonexistentReturnsErr(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	result, err := Call(ctx, "FS", "removeFileResult", []eval.Value{&eval.StringValue{Value: "/this/path/should/not/exist.txt"}})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	assertResultErrContains(t, result, "cannot remove file")
}

// --- mkdirAllResult ---

func TestFSMkdirAllResult_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")

	result, err := Call(ctx, "FS", "mkdirAllResult", []eval.Value{&eval.StringValue{Value: target}})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	if _, ok := assertResultOk(t, result).(*eval.UnitValue); !ok {
		t.Fatal("expected UnitValue inside Ok")
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected created path to be a directory")
	}
}

func TestFSMkdirAllResult_OverFileReturnsErr(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	dir := t.TempDir()
	conflict := filepath.Join(dir, "a-file")
	if err := os.WriteFile(conflict, []byte("not a dir"), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(conflict, "subdir")

	result, err := Call(ctx, "FS", "mkdirAllResult", []eval.Value{&eval.StringValue{Value: target}})
	if err != nil {
		t.Fatalf("expected no Go error, got %v", err)
	}
	assertResultErrContains(t, result, "cannot create directory")
}
