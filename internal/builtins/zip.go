package builtins

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// ZIP archive builtins for AILANG
// These provide read-only ZIP archive access with capability-based security
// Part of M-STDLIB-ZIP (v0.7.3)

const (
	zipMaxEntries          = 10000
	zipMaxDecompressedSize = 100 * 1024 * 1024 // 100MB
)

func init() {
	registerZipListEntries()
	registerZipReadEntry()
	registerZipReadEntryBytes()
	registerZipCreateArchive()
}

// Result helpers for Ok/Err return values

func zipMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func zipMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// ============================================================================
// _zip_listEntries: string -> Result[[string], string] ! {FS}
// ============================================================================

func registerZipListEntries() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_listEntries",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeZipListEntriesType,
		Impl:    zipListEntriesImpl,
		Metadata: &BuiltinMetadata{
			Description: "List all file paths in a ZIP archive",
			LongDesc:    "Opens a ZIP archive and returns a list of all entry names (file paths within the archive). Respects AILANG_FS_SANDBOX. Rejects archives with more than 10,000 entries.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the ZIP archive"},
			},
			Returns: "Result[[string], string] - Ok(list of entry names) or Err(error message)",
			Examples: []Example{
				{Code: `_zip_listEntries("doc.docx")`, Description: `Returns Ok(["word/document.xml", "[Content_Types].xml", ...])`},
			},
			SeeAlso:   []string{"_zip_readEntry", "_zip_readEntryBytes"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "list", "fs"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_listEntries: %v", err))
	}
}

func makeZipListEntriesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.List(T.String()), T.String()),
	).Effects("FS")
}

func zipListEntriesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_listEntries: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
	}
	defer r.Close()

	if len(r.File) > zipMaxEntries {
		return zipMakeErr(fmt.Sprintf("too many entries: %d (max %d)", len(r.File), zipMaxEntries)), nil
	}

	entries := make([]eval.Value, len(r.File))
	for i, f := range r.File {
		entries[i] = &eval.StringValue{Value: f.Name}
	}

	return zipMakeOk(&eval.ListValue{Elements: entries}), nil
}

// ============================================================================
// _zip_readEntry: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerZipReadEntry() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_readEntry",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeZipReadEntryType,
		Impl:    zipReadEntryImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a text entry from a ZIP archive as a UTF-8 string",
			LongDesc:    "Opens a ZIP archive and reads the specified entry as a UTF-8 string. Rejects entry names containing path traversal (../). Limits decompressed size to 100MB. Respects AILANG_FS_SANDBOX.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the ZIP archive"},
				{Name: "entryName", Description: "Name of the entry within the archive"},
			},
			Returns: "Result[string, string] - Ok(entry contents) or Err(error message)",
			Examples: []Example{
				{Code: `_zip_readEntry("doc.docx", "word/document.xml")`, Description: `Returns Ok("<w:document>...")`},
			},
			SeeAlso:   []string{"_zip_listEntries", "_zip_readEntryBytes"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "read", "text", "fs"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_readEntry: %v", err))
	}
}

func makeZipReadEntryType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func zipReadEntryImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_readEntry: expected String for path, got %T", args[0])
	}
	entryVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_readEntry: expected String for entryName, got %T", args[1])
	}

	entryName := entryVal.Value
	if strings.Contains(entryName, "..") {
		return zipMakeErr(fmt.Sprintf("path traversal rejected: %s", entryName)), nil
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == entryName {
			data, err := readZipEntry(f)
			if err != nil {
				return zipMakeErr(err.Error()), nil
			}
			return zipMakeOk(&eval.StringValue{Value: string(data)}), nil
		}
	}

	return zipMakeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
}

// ============================================================================
// _zip_readEntryBytes: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerZipReadEntryBytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_readEntryBytes",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeZipReadEntryBytesType,
		Impl:    zipReadEntryBytesImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a binary entry from a ZIP archive as a base64-encoded string",
			LongDesc:    "Opens a ZIP archive and reads the specified entry, returning the content as a base64-encoded string. Use _bytes_from_base64 to decode. Rejects path traversal. Limits decompressed size to 100MB. Respects AILANG_FS_SANDBOX.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the ZIP archive"},
				{Name: "entryName", Description: "Name of the entry within the archive"},
			},
			Returns: "Result[string, string] - Ok(base64 encoded content) or Err(error message)",
			Examples: []Example{
				{Code: `_zip_readEntryBytes("doc.docx", "word/media/image1.png")`, Description: `Returns Ok("iVBORw0KGgo...")`},
			},
			SeeAlso:   []string{"_zip_listEntries", "_zip_readEntry", "_bytes_from_base64"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "read", "binary", "base64", "fs"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_readEntryBytes: %v", err))
	}
}

func makeZipReadEntryBytesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func zipReadEntryBytesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_readEntryBytes: expected String for path, got %T", args[0])
	}
	entryVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_readEntryBytes: expected String for entryName, got %T", args[1])
	}

	entryName := entryVal.Value
	if strings.Contains(entryName, "..") {
		return zipMakeErr(fmt.Sprintf("path traversal rejected: %s", entryName)), nil
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == entryName {
			data, err := readZipEntry(f)
			if err != nil {
				return zipMakeErr(err.Error()), nil
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			return zipMakeOk(&eval.StringValue{Value: encoded}), nil
		}
	}

	return zipMakeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
}

// ============================================================================
// _zip_createArchive: string -> [{name: string, content: string}] -> Result[(), string] ! {FS}
// ============================================================================
// M-DOCPARSE-DX M4: Functional batch ZIP write — creates entire archive atomically

func registerZipCreateArchive() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_createArchive",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeZipCreateArchiveType,
		Impl:    zipCreateArchiveImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create a ZIP archive from a list of entries",
			LongDesc:    "Creates a complete ZIP archive atomically. Each entry is a record with {name: string, content: string}. Archive is properly flushed and closed. Respects AILANG_FS_SANDBOX.",
			Params: []ParamDoc{
				{Name: "path", Description: "Output path for the ZIP archive"},
				{Name: "entries", Description: "List of {name: string, content: string} records"},
			},
			Returns: "Result[(), string] - Ok(()) on success or Err(error message)",
			Examples: []Example{
				{Code: `_zip_createArchive("out.zip", [{name: "hello.txt", content: "Hello!"}])`, Description: "Creates archive with one text entry"},
			},
			SeeAlso:   []string{"_zip_listEntries", "_zip_readEntry"},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "write", "create", "fs"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_createArchive: %v", err))
	}
}

func makeZipCreateArchiveType() types.Type {
	T := types.NewBuilder()
	entryType := T.Record(
		types.Field("name", T.String()),
		types.Field("content", T.String()),
	)
	return T.Func(T.String(), T.List(entryType)).Returns(
		T.App("Result", T.Unit(), T.String()),
	).Effects("FS")
}

func zipCreateArchiveImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_createArchive: expected String for path, got %T", args[0])
	}
	entriesVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_zip_createArchive: expected List for entries, got %T", args[1])
	}

	if len(entriesVal.Elements) > zipMaxEntries {
		return zipMakeErr(fmt.Sprintf("too many entries: %d (max %d)", len(entriesVal.Elements), zipMaxEntries)), nil
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
	}

	f, err := os.Create(path)
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot create file: %v", err)), nil
	}

	w := zip.NewWriter(f)
	var writeErr error

	for i, entry := range entriesVal.Elements {
		rec, ok := entry.(*eval.RecordValue)
		if !ok {
			writeErr = fmt.Errorf("entry %d: expected record, got %T", i, entry)
			break
		}
		nameVal, ok := rec.Fields["name"].(*eval.StringValue)
		if !ok {
			writeErr = fmt.Errorf("entry %d: 'name' field must be string", i)
			break
		}
		contentVal, ok := rec.Fields["content"].(*eval.StringValue)
		if !ok {
			writeErr = fmt.Errorf("entry %d: 'content' field must be string", i)
			break
		}

		if strings.Contains(nameVal.Value, "..") {
			writeErr = fmt.Errorf("entry %d: path traversal rejected: %s", i, nameVal.Value)
			break
		}

		ew, err := w.Create(nameVal.Value)
		if err != nil {
			writeErr = fmt.Errorf("entry %d: cannot create: %v", i, err)
			break
		}
		if _, err := io.WriteString(ew, contentVal.Value); err != nil {
			writeErr = fmt.Errorf("entry %d: write error: %v", i, err)
			break
		}
	}

	// Always close the writer and file
	if cerr := w.Close(); cerr != nil && writeErr == nil {
		writeErr = fmt.Errorf("close archive: %v", cerr)
	}
	if cerr := f.Close(); cerr != nil && writeErr == nil {
		writeErr = fmt.Errorf("close file: %v", cerr)
	}

	if writeErr != nil {
		// Clean up on error
		os.Remove(path)
		return zipMakeErr(writeErr.Error()), nil
	}

	return zipMakeOk(&eval.UnitValue{}), nil
}

// ============================================================================
// Shared helpers
// ============================================================================

// readZipEntry reads a zip file entry with size limits
func readZipEntry(f *zip.File) ([]byte, error) {
	if f.UncompressedSize64 > uint64(zipMaxDecompressedSize) {
		return nil, fmt.Errorf("entry too large: %d bytes (max %d)", f.UncompressedSize64, zipMaxDecompressedSize)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("cannot read entry: %v", err)
	}
	defer rc.Close()

	// Use LimitReader as defense-in-depth even if header says it's small
	limited := io.LimitReader(rc, int64(zipMaxDecompressedSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}
	if len(data) > zipMaxDecompressedSize {
		return nil, fmt.Errorf("entry too large: exceeded %d bytes", zipMaxDecompressedSize)
	}

	return data, nil
}
