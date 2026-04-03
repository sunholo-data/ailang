package apiserver

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestWriteTempFile(t *testing.T) {
	data := []byte("hello world")

	t.Run("preserves original filename", func(t *testing.T) {
		path, err := writeTempFile(data, "report.docx")
		if err != nil {
			t.Fatalf("writeTempFile: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path))

		if filepath.Base(path) != "report.docx" {
			t.Errorf("expected filename report.docx, got %q", filepath.Base(path))
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(contents) != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", string(contents))
		}
	})

	t.Run("no filename fallback", func(t *testing.T) {
		path, err := writeTempFile(data, "")
		if err != nil {
			t.Fatalf("writeTempFile: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path))

		if filepath.Base(path) != "upload" {
			t.Errorf("expected fallback filename 'upload', got %q", filepath.Base(path))
		}
	})
}

// makeMultipartRequest builds an *http.Request with multipart/form-data fields.
func makeMultipartRequest(t *testing.T, files map[string][]byte, fields map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for name, data := range files {
		part, err := w.CreateFormFile(name, name+".bin")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		part.Write(data)
	}
	for name, val := range fields {
		if err := w.WriteField(name, val); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
	}
	w.Close()

	req, err := http.NewRequest("POST", "/test", &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	return req
}

func TestParseMultipartArgsWithNames_FileToString(t *testing.T) {
	req := makeMultipartRequest(t,
		map[string][]byte{"filepath": []byte("PDF content")},
		map[string]string{"format": "markdown"},
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"filepath", "format"},
		[]string{"string", "string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	// File field + string param → temp file path with original filename
	path, ok := args[0].(string)
	if !ok {
		t.Fatalf("expected string for filepath arg, got %T", args[0])
	}
	if filepath.Base(path) != "filepath.bin" {
		t.Errorf("expected original filename filepath.bin, got %q", filepath.Base(path))
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(contents) != "PDF content" {
		t.Errorf("temp file content = %q, want %q", string(contents), "PDF content")
	}

	// Non-file field → string
	if args[1] != "markdown" {
		t.Errorf("format = %q, want %q", args[1], "markdown")
	}
}

func TestParseMultipartArgsWithNames_FileToBytes(t *testing.T) {
	req := makeMultipartRequest(t,
		map[string][]byte{"data": []byte("raw binary")},
		map[string]string{"format": "png"},
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"data", "format"},
		[]string{"bytes", "string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	// File field + bytes param → BytesValue
	bv, ok := args[0].(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected *eval.BytesValue for data arg, got %T", args[0])
	}
	if string(bv.Value) != "raw binary" {
		t.Errorf("BytesValue.Value = %q, want %q", string(bv.Value), "raw binary")
	}

	if args[1] != "png" {
		t.Errorf("format = %q, want %q", args[1], "png")
	}
}

func TestParseMultipartArgsWithNames_UnmatchedParams(t *testing.T) {
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("data")},
		nil,
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"file", "apiKey"},
		[]string{"bytes", "string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	// Matched file field
	if _, ok := args[0].(*eval.BytesValue); !ok {
		t.Fatalf("expected *eval.BytesValue, got %T", args[0])
	}

	// Unmatched string param → zero-value ""
	if args[1] != "" {
		t.Errorf("unmatched apiKey = %v, want empty string", args[1])
	}
}

func TestParseMultipartArgsWithNames_NoParamNames_Fallback(t *testing.T) {
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("data")},
		map[string]string{"key": "val"},
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20, nil, nil)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	// Falls back to positional — should have at least 1 arg
	if len(args) == 0 {
		t.Fatal("expected positional fallback to return args")
	}

	// One of the args should be a BytesValue (from the file field)
	foundBytes := false
	for _, a := range args {
		if _, ok := a.(*eval.BytesValue); ok {
			foundBytes = true
		}
	}
	if !foundBytes {
		t.Error("positional fallback should return BytesValue for file fields")
	}
}

func TestParseMultipartArgsWithNames_CleanupRemovesTempFiles(t *testing.T) {
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("temp data")},
		nil,
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"file"},
		[]string{"string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}

	path := args[0].(string)
	// File should exist before cleanup
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temp file should exist before cleanup: %v", err)
	}

	cleanup()

	// File should be gone after cleanup
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file should be removed after cleanup, got err: %v", err)
	}
}

func TestParseMultipartArgsWithNames_MismatchedFieldName(t *testing.T) {
	// Reproduces the P0 bug: curl -F 'file=@doc.docx' with param named 'filepath'
	// Field name "file" doesn't match param name "filepath", but the file should
	// still be assigned to the unmatched string param via pass 2 fallback.
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("document content")},
		nil,
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"filepath"},
		[]string{"string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	// File should be saved to temp and path returned (not empty string)
	path, ok := args[0].(string)
	if !ok {
		t.Fatalf("expected string for filepath arg, got %T", args[0])
	}
	if path == "" {
		t.Fatal("filepath arg is empty — file upload was silently dropped")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(contents) != "document content" {
		t.Errorf("temp file content = %q, want %q", string(contents), "document content")
	}
}

func TestParseMultipartArgsWithNames_MismatchedFieldName_Bytes(t *testing.T) {
	// Same mismatch but with a bytes-typed param
	req := makeMultipartRequest(t,
		map[string][]byte{"upload": []byte("binary data")},
		nil,
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"data"},
		[]string{"bytes"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	bv, ok := args[0].(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected *eval.BytesValue, got %T", args[0])
	}
	if string(bv.Value) != "binary data" {
		t.Errorf("BytesValue.Value = %q, want %q", string(bv.Value), "binary data")
	}
}

func TestParseMultipartArgsWithNames_MismatchedWithExtraFields(t *testing.T) {
	// File with mismatched name + a matched non-file field
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("doc content")},
		map[string]string{"format": "markdown"},
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"filepath", "format"},
		[]string{"string", "string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	// filepath should get the unmatched file
	path, ok := args[0].(string)
	if !ok || path == "" {
		t.Fatalf("filepath arg should be a non-empty temp path, got %v (%T)", args[0], args[0])
	}

	// format should get the matched form field
	if args[1] != "markdown" {
		t.Errorf("format = %v, want %q", args[1], "markdown")
	}
}

func TestParseMultipartArgsWithNames_ExactMatchTakesPriority(t *testing.T) {
	// When field name matches exactly, it should be used even if there are other unmatched files
	req := makeMultipartRequest(t,
		map[string][]byte{"filepath": []byte("exact match")},
		nil,
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"filepath"},
		[]string{"string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	path, ok := args[0].(string)
	if !ok || path == "" {
		t.Fatalf("expected non-empty string, got %v (%T)", args[0], args[0])
	}
	contents, _ := os.ReadFile(path)
	if string(contents) != "exact match" {
		t.Errorf("content = %q, want %q", string(contents), "exact match")
	}
}

func TestParseMultipartArgsWithNames_NamedOrdering(t *testing.T) {
	// Verify args are in param declaration order, not map iteration order
	req := makeMultipartRequest(t,
		map[string][]byte{"file": []byte("content")},
		map[string]string{"apiKey": "key123", "format": "md"},
	)

	args, cleanup, err := parseMultipartArgsWithNames(req, 32<<20,
		[]string{"format", "file", "apiKey"},
		[]string{"string", "bytes", "string"},
	)
	if err != nil {
		t.Fatalf("parseMultipartArgsWithNames: %v", err)
	}
	defer cleanup()

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	// args[0] = format (non-file field, string)
	if args[0] != "md" {
		t.Errorf("args[0] (format) = %v, want %q", args[0], "md")
	}

	// args[1] = file (file field, bytes type → BytesValue)
	if _, ok := args[1].(*eval.BytesValue); !ok {
		t.Errorf("args[1] (file) = %T, want *eval.BytesValue", args[1])
	}

	// args[2] = apiKey (non-file field, string)
	if args[2] != "key123" {
		t.Errorf("args[2] (apiKey) = %v, want %q", args[2], "key123")
	}
}
