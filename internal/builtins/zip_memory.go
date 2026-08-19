package builtins

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// In-memory ZIP archive builders for AILANG.
//
// ailang#644: every archive constructor in std/zip wrote to a filesystem path
// and carried the FS effect, which made document GENERATION impossible in the
// browser — a WASM host has no filesystem, so the OOXML/ODF generators could
// not be loaded into a browser bundle at all even though every byte they need
// is already in memory. Extraction already had an escape hatch (the docs-site
// demo unzips in JS and hands entry content to AILANG); the write path had none.
//
// These two builders are the counterpart: they serialise to memory and RETURN
// the archive as base64 instead of writing it. That also makes the effect
// requirement honest — building a ZIP is pure, writing one is not. A caller who
// wants a file writes the result itself via std/fs.
//
// Purity is a real claim about this output, not a convenience: archive/zip
// stamps entries with the zero time.Time (MS-DOS epoch 1979-11-30) rather than
// the wall clock, so the same entry list serialises to the same bytes on every
// call. TestZipBuildArchive_Deterministic pins that.

// zipMaxArchiveContentSize caps the TOTAL payload bytes an in-memory builder
// will accept across all entries. The FS builders are unbounded in aggregate
// because their output streams to disk; these retain the whole archive in
// memory and then base64-expand it by 4/3, so an unbounded builder is an OOM
// primitive in a browser tab.
//
// A var rather than a const so the guard is reachable from a test at a sane
// size. A limit whose only exercise would be allocating 100MB is a limit
// nothing ever reds on, which is not a limit — it is a comment.
var zipMaxArchiveContentSize = 100 * 1024 * 1024 // 100MB

func init() {
	registerZipBuildArchive()
	registerZipBuildArchiveWithBytes()
}

// buildZipArchiveB64 serialises entries to an in-memory ZIP and returns the
// archive as a base64 string, mirroring the base64-at-the-boundary convention
// std/zip.readEntryBytes, std/gzip and std/deflate already use.
func buildZipArchiveB64(fn string, args []eval.Value, enc zipEntryEncoding) (eval.Value, error) {
	entriesVal, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected List for entries, got %T", fn, args[0])
	}

	if len(entriesVal.Elements) > zipMaxEntries {
		return zipMakeErr(fmt.Sprintf("too many entries: %d (max %d)", len(entriesVal.Elements), zipMaxEntries)), nil
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	writeErr := writeZipEntries(w, fn, entriesVal.Elements, enc, zipMaxArchiveContentSize)

	// Always close the writer: it is what flushes the central directory, so an
	// archive whose Close is skipped is silently truncated rather than absent.
	if cerr := w.Close(); cerr != nil && writeErr == nil {
		writeErr = fmt.Errorf("close archive: %v", cerr)
	}

	if writeErr != nil {
		return zipMakeErr(writeErr.Error()), nil
	}

	return zipMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}), nil
}

// ============================================================================
// _zip_buildArchive: [{name: string, content: string}] -> Result[string, string]   (pure)
// ============================================================================

func registerZipBuildArchive() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_buildArchive",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeZipBuildArchiveType,
		Impl:    zipBuildArchiveImpl,
		Metadata: &BuiltinMetadata{
			Description: "Build a ZIP archive in memory from text entries (base64 out, no FS)",
			LongDesc:    "Serialises a list of {name: string, content: string} records into a ZIP archive held in memory and returns it base64-encoded. The in-memory counterpart of _zip_createArchive: no filesystem, no FS effect, so it works in WASM/browser hosts where document GENERATION was previously impossible. Rejects path traversal and caps total content at 100MB. Output is deterministic — entries carry the MS-DOS epoch, not the wall clock.",
			Params: []ParamDoc{
				{Name: "entries", Description: "List of {name: string, content: string} records"},
			},
			Returns: "Result[string, string] - Ok(base64 of the ZIP archive) or Err(error message)",
			Examples: []Example{
				{Code: `_zip_buildArchive([{name: "hello.txt", content: "Hello!"}])`, Description: "Returns Ok(base64-encoded archive bytes)"},
			},
			SeeAlso:   []string{"_zip_buildArchiveWithBytes", "_zip_createArchive", "_zip_readEntry"},
			Since:     "v0.34.0",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "build", "memory", "base64", "pure", "wasm"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_buildArchive: %v", err))
	}
}

func makeZipBuildArchiveType() types.Type {
	T := types.NewBuilder()
	entryType := T.Record(
		types.Field("name", T.String()),
		types.Field("content", T.String()),
	)
	return T.Func(T.List(entryType)).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func zipBuildArchiveImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return buildZipArchiveB64("_zip_buildArchive", args, zipEntryText)
}

// ============================================================================
// _zip_buildArchiveWithBytes: [{name: string, data: string}] -> Result[string, string]   (pure)
// ============================================================================

func registerZipBuildArchiveWithBytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_buildArchiveWithBytes",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeZipBuildArchiveWithBytesType,
		Impl:    zipBuildArchiveWithBytesImpl,
		Metadata: &BuiltinMetadata{
			Description: "Build a ZIP archive in memory from base64 entries (base64 out, no FS)",
			LongDesc:    "Serialises a list of {name: string, data: string} records — where data is base64-encoded — into a ZIP archive held in memory and returns it base64-encoded. The in-memory counterpart of _zip_createArchiveWithBytes. Use for mixed text/binary documents (DOCX/PPTX/XLSX/ODT with embedded images or fonts) in WASM/browser hosts. Rejects path traversal and caps total decoded content at 100MB. Output is deterministic.",
			Params: []ParamDoc{
				{Name: "entries", Description: "List of {name: string, data: string} where data is base64-encoded"},
			},
			Returns: "Result[string, string] - Ok(base64 of the ZIP archive) or Err(error message)",
			Examples: []Example{
				{Code: `_zip_buildArchiveWithBytes([{name: "image.png", data: "iVBORw0KGgo..."}])`, Description: "Returns Ok(base64-encoded archive bytes)"},
			},
			SeeAlso:   []string{"_zip_buildArchive", "_zip_createArchiveWithBytes", "_zip_readEntryBytes"},
			Since:     "v0.34.0",
			Stability: StabilityStable,
			Tags:      []string{"zip", "archive", "build", "memory", "binary", "base64", "pure", "wasm"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_buildArchiveWithBytes: %v", err))
	}
}

func makeZipBuildArchiveWithBytesType() types.Type {
	T := types.NewBuilder()
	entryType := T.Record(
		types.Field("name", T.String()),
		types.Field("data", T.String()),
	)
	return T.Func(T.List(entryType)).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func zipBuildArchiveWithBytesImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return buildZipArchiveB64("_zip_buildArchiveWithBytes", args, zipEntryBase64)
}
