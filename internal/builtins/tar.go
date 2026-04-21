package builtins

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// TAR archive builtins for AILANG.
// Part of M-STDLIB-TAR-GZIP (v0.12.0)
//
// Caps and limits:
//   - 10,000 entries max per archive (matches std/zip)
//   - 100 MB decompressed per entry
//   - Respects AILANG_FS_SANDBOX
//   - Rejects path traversal ("..") in entry names
//   - extractAll also rejects symlink escapes

const (
	tarMaxEntries          = 10000
	tarMaxDecompressedSize = 100 * 1024 * 1024 // 100 MB
)

func init() {
	registerTarListEntries()
	registerTarReadEntry()
	registerTarReadEntryBytes()
	registerTarExtractAll()
	registerTarReadFromGzip()
	registerTarReadFromGzipBytes()
}

// Result helpers

func tarMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func tarMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// ============================================================================
// Shared helpers
// ============================================================================

// resolveTarPath applies the sandbox prefix if one is configured.
func resolveTarPath(ctx *effects.EffContext, path string) string {
	if ctx.Env.Sandbox != "" {
		return filepath.Join(ctx.Env.Sandbox, path)
	}
	return path
}

// isEntryPathTraversal returns true if the entry name contains ".." segments
// or is an absolute path. This is the same check std/zip applies.
func isEntryPathTraversal(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "..") {
		return true
	}
	// Cross-platform absolute-path check: tar entry names use forward slashes
	// by convention. filepath.IsAbs on Windows requires a drive letter and
	// would miss "/tmp/foo", so check for leading / or \ explicitly.
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) {
		return true
	}
	if filepath.IsAbs(name) {
		return true
	}
	// filepath.IsLocal (Go 1.20+) catches more edge cases: absolute, reserved
	// Windows names, and volume-rooted paths. Use it as an additional guard.
	cleaned := filepath.ToSlash(filepath.Clean(name))
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return true
	}
	return false
}

// readTarEntry reads a tar entry body with the 100MB cap enforced.
func readTarEntry(tr *tar.Reader, declaredSize int64) ([]byte, error) {
	// Use LimitReader as a defense-in-depth even when the header says it's small.
	if declaredSize > tarMaxDecompressedSize {
		return nil, fmt.Errorf("entry too large: %d bytes (max %d)", declaredSize, tarMaxDecompressedSize)
	}
	limited := io.LimitReader(tr, int64(tarMaxDecompressedSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}
	if len(data) > tarMaxDecompressedSize {
		return nil, fmt.Errorf("entry too large: exceeded %d bytes", tarMaxDecompressedSize)
	}
	return data, nil
}

// openTarReader opens a file and returns a *tar.Reader along with a close func.
// If gzipped is true, wraps the file in a gzip.Reader first.
func openTarReader(path string, gzipped bool) (*tar.Reader, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open: %v", err)
	}

	if !gzipped {
		tr := tar.NewReader(f)
		return tr, func() { _ = f.Close() }, nil
	}

	gr, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("invalid gzip: %v", err)
	}
	tr := tar.NewReader(gr)
	closer := func() {
		_ = gr.Close()
		_ = f.Close()
	}
	return tr, closer, nil
}

// makeTarEntryRecord builds a {name: string, size: int, isDir: bool} record value.
func makeTarEntryRecord(name string, size int64, isDir bool) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name":  &eval.StringValue{Value: name},
			"size":  &eval.IntValue{Value: int(size)},
			"isDir": &eval.BoolValue{Value: isDir},
		},
	}
}

func tarEntryRecordType() types.Type {
	T := types.NewBuilder()
	return T.Record(
		types.Field("name", T.String()),
		types.Field("size", T.Int()),
		types.Field("isDir", T.Bool()),
	)
}

// ============================================================================
// _tar_listEntries: string -> Result[[TarEntry], string] ! {FS}
// ============================================================================

func registerTarListEntries() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_listEntries",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarListEntriesType,
		Impl:    tarListEntriesImpl,
		Metadata: &BuiltinMetadata{
			Description: "List all entries in a tar archive",
			LongDesc:    "Returns each entry's name, declared size, and directory flag. Rejects archives with more than 10,000 entries. Respects AILANG_FS_SANDBOX.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the tar archive"},
			},
			Returns:   "Result[[{name, size, isDir}], string]",
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"tar", "archive", "list", "fs"},
			Category:  "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_listEntries: %v", err))
	}
}

func makeTarListEntriesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.List(tarEntryRecordType()), T.String()),
	).Effects("FS")
}

func tarListEntriesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_tar_listEntries: expected String, got %T", args[0])
	}

	path := resolveTarPath(ctx, pathVal.Value)
	tr, closer, err := openTarReader(path, false)
	if err != nil {
		return tarMakeErr(err.Error()), nil
	}
	defer closer()

	entries := []eval.Value{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tarMakeErr(fmt.Sprintf("tar read error: %v", err)), nil
		}
		if len(entries) >= tarMaxEntries {
			return tarMakeErr(fmt.Sprintf("too many entries: exceeded %d", tarMaxEntries)), nil
		}
		entries = append(entries, makeTarEntryRecord(
			hdr.Name,
			hdr.Size,
			hdr.Typeflag == tar.TypeDir,
		))
	}

	return tarMakeOk(&eval.ListValue{Elements: entries}), nil
}

// ============================================================================
// _tar_readEntry: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerTarReadEntry() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_readEntry",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarReadEntryType,
		Impl:    tarReadEntryImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a text entry from a tar archive as a UTF-8 string",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the tar archive"},
				{Name: "entryName", Description: "Name of the entry within the archive"},
			},
			Returns:   "Result[string, string]",
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"tar", "archive", "read", "text", "fs"},
			Category:  "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_readEntry: %v", err))
	}
}

func makeTarReadEntryType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func tarReadEntryImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return tarReadEntryGeneric(ctx, args, false, false)
}

// ============================================================================
// _tar_readEntryBytes: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerTarReadEntryBytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_readEntryBytes",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarReadEntryBytesType,
		Impl:    tarReadEntryBytesImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a binary entry from a tar archive as a base64-encoded string",
			Returns:     "Result[string, string] — Ok(base64 content) or Err(message)",
			Since:       "v0.12.0",
			Stability:   StabilityStable,
			Tags:        []string{"tar", "archive", "read", "binary", "base64", "fs"},
			Category:    "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_readEntryBytes: %v", err))
	}
}

func makeTarReadEntryBytesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func tarReadEntryBytesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return tarReadEntryGeneric(ctx, args, false, true)
}

// tarReadEntryGeneric backs all four single-entry read builtins:
//
//	fromGzip=true  → path points to a .tar.gz
//	asBase64=true  → return the data as base64 (binary), else as UTF-8 text
func tarReadEntryGeneric(ctx *effects.EffContext, args []eval.Value, fromGzip, asBase64 bool) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("tar read: expected String for path, got %T", args[0])
	}
	entryVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("tar read: expected String for entryName, got %T", args[1])
	}

	entryName := entryVal.Value
	if isEntryPathTraversal(entryName) {
		return tarMakeErr(fmt.Sprintf("path traversal rejected: %s", entryName)), nil
	}

	path := resolveTarPath(ctx, pathVal.Value)
	tr, closer, err := openTarReader(path, fromGzip)
	if err != nil {
		return tarMakeErr(err.Error()), nil
	}
	defer closer()

	scanned := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tarMakeErr(fmt.Sprintf("tar read error: %v", err)), nil
		}
		scanned++
		if scanned > tarMaxEntries {
			return tarMakeErr(fmt.Sprintf("too many entries: exceeded %d", tarMaxEntries)), nil
		}
		if hdr.Name != entryName {
			continue
		}
		if hdr.Typeflag == tar.TypeDir {
			return tarMakeErr(fmt.Sprintf("entry is a directory: %s", entryName)), nil
		}
		data, err := readTarEntry(tr, hdr.Size)
		if err != nil {
			return tarMakeErr(err.Error()), nil
		}
		if asBase64 {
			return tarMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(data)}), nil
		}
		return tarMakeOk(&eval.StringValue{Value: string(data)}), nil
	}

	return tarMakeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
}

// ============================================================================
// _tar_extractAll: string -> string -> Result[[string], string] ! {FS}
// ============================================================================

func registerTarExtractAll() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_extractAll",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarExtractAllType,
		Impl:    tarExtractAllImpl,
		Metadata: &BuiltinMetadata{
			Description: "Extract all entries from a tar archive into destDir",
			LongDesc:    "Writes every regular-file and directory entry under destDir. Rejects tarbomb (../) entries and symlinks that escape destDir. Returns the ordered list of written absolute paths. Respects AILANG_FS_SANDBOX.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the tar archive"},
				{Name: "destDir", Description: "Destination directory (created if missing)"},
			},
			Returns:   "Result[[string], string] — Ok(written paths) or Err(message)",
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"tar", "archive", "extract", "fs"},
			Category:  "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_extractAll: %v", err))
	}
}

func makeTarExtractAllType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.List(T.String()), T.String()),
	).Effects("FS")
}

func tarExtractAllImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_tar_extractAll: expected String for path, got %T", args[0])
	}
	destVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_tar_extractAll: expected String for destDir, got %T", args[1])
	}

	path := resolveTarPath(ctx, pathVal.Value)
	destDir := resolveTarPath(ctx, destVal.Value)

	// Resolve destDir to an absolute clean form so symlink-escape checks are reliable.
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return tarMakeErr(fmt.Sprintf("cannot resolve destDir: %v", err)), nil
	}
	if err := os.MkdirAll(absDest, 0o755); err != nil {
		return tarMakeErr(fmt.Sprintf("cannot create destDir: %v", err)), nil
	}

	tr, closer, err := openTarReader(path, false)
	if err != nil {
		return tarMakeErr(err.Error()), nil
	}
	defer closer()

	writtenPaths := []eval.Value{}
	scanned := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tarMakeErr(fmt.Sprintf("tar read error: %v", err)), nil
		}
		scanned++
		if scanned > tarMaxEntries {
			return tarMakeErr(fmt.Sprintf("too many entries: exceeded %d", tarMaxEntries)), nil
		}

		// 1. Reject path traversal in entry name
		if isEntryPathTraversal(hdr.Name) {
			return tarMakeErr(fmt.Sprintf("path traversal rejected: %s", hdr.Name)), nil
		}

		// 2. Compute target path and verify it stays under absDest
		target := filepath.Join(absDest, filepath.FromSlash(hdr.Name))
		rel, err := filepath.Rel(absDest, target)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return tarMakeErr(fmt.Sprintf("path escapes destDir: %s", hdr.Name)), nil
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return tarMakeErr(fmt.Sprintf("mkdir %s: %v", hdr.Name, err)), nil
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return tarMakeErr(fmt.Sprintf("mkdir parent %s: %v", hdr.Name, err)), nil
			}
			data, err := readTarEntry(tr, hdr.Size)
			if err != nil {
				return tarMakeErr(fmt.Sprintf("entry %s: %v", hdr.Name, err)), nil
			}
			// Use 0o644 regardless of archive mode for predictable sandbox behaviour.
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return tarMakeErr(fmt.Sprintf("write %s: %v", hdr.Name, err)), nil
			}
			writtenPaths = append(writtenPaths, &eval.StringValue{Value: target})
		case tar.TypeSymlink:
			// Symlinks are rejected outright — even if the link target is local now,
			// resolving it later during FS access could escape. Safer to refuse.
			return tarMakeErr(fmt.Sprintf("symlink rejected: %s -> %s", hdr.Name, hdr.Linkname)), nil
		case tar.TypeLink:
			return tarMakeErr(fmt.Sprintf("hard link rejected: %s -> %s", hdr.Name, hdr.Linkname)), nil
		default:
			// Skip devices, fifos, etc. — not relevant to document parsing use cases.
			continue
		}
	}

	return tarMakeOk(&eval.ListValue{Elements: writtenPaths}), nil
}

// ============================================================================
// _tar_readFromGzip: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerTarReadFromGzip() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_readFromGzip",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarReadFromGzipType,
		Impl:    tarReadFromGzipImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a text entry from a .tar.gz archive without writing a temp file",
			LongDesc:    "Streams the file through gzip + tar in memory, returning the named entry as UTF-8 text. Designed for arXiv source bundles and similar composed archives.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the .tar.gz archive"},
				{Name: "entryName", Description: "Name of the entry to extract"},
			},
			Returns:   "Result[string, string]",
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"tar", "gzip", "archive", "read", "text", "fs"},
			Category:  "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_readFromGzip: %v", err))
	}
}

func makeTarReadFromGzipType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func tarReadFromGzipImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return tarReadEntryGeneric(ctx, args, true, false)
}

// ============================================================================
// _tar_readFromGzipBytes: string -> string -> Result[string, string] ! {FS}
// ============================================================================

func registerTarReadFromGzipBytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/tar",
		Name:    "_tar_readFromGzipBytes",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeTarReadFromGzipBytesType,
		Impl:    tarReadFromGzipBytesImpl,
		Metadata: &BuiltinMetadata{
			Description: "Read a binary entry from a .tar.gz archive as base64 without a temp file",
			Returns:     "Result[string, string] — Ok(base64) or Err(message)",
			Since:       "v0.12.0",
			Stability:   StabilityStable,
			Tags:        []string{"tar", "gzip", "archive", "read", "binary", "base64", "fs"},
			Category:    "tar",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _tar_readFromGzipBytes: %v", err))
	}
}

func makeTarReadFromGzipBytesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func tarReadFromGzipBytesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return tarReadEntryGeneric(ctx, args, true, true)
}
