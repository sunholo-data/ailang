package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/format"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// fmt.go implements `ailang fmt` — the canonical AILANG source formatter.
//
// Modes:
//   ailang fmt <file.ail>          write canonical source to stdout (exactly one file)
//   ailang fmt --write <files...>  atomically rewrite each file in place
//   ailang fmt --check <files...>  list drifted files; exit 1 if any drift
//
// Exit codes (fail-closed everywhere):
//   0  formatting succeeded, or every file is canonical in --check mode
//   1  --check found at least one non-canonical file (no operational error)
//   2  usage / read / comment-preflight / parse / print / round-trip / write error
//
// Phase 1 refuses commented input (the AST carries no trivia, so a reprint would
// silently delete comments). There is NO fallback to original source.

// fmtIgnorePos ignores positional metadata so the round-trip AST comparison is
// purely structural, matching the parser's own equivalence tests.
var fmtIgnorePos = cmpopts.IgnoreTypes(ast.Pos{}, ast.Span{})

// runFmtCommand is the entry point dispatched from main's `case "fmt"`. It parses
// the subcommand's own flags from the remaining args and exits with the
// appropriate code; it never returns on the exit paths.
func runFmtCommand(args []string) {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	writeFlag := fs.Bool("write", false, "Rewrite each file in place with canonical formatting")
	checkFlag := fs.Bool("check", false, "Report files that are not canonically formatted (no writes); exit 1 if any drift")
	helpFlag := fs.Bool("help", false, "Show help for the fmt command")
	fs.Usage = printFmtHelp
	if err := fs.Parse(args); err != nil {
		// flag already printed the error to stderr.
		os.Exit(2)
	}
	if *helpFlag {
		printFmtHelp()
		os.Exit(0)
	}

	files := fs.Args()

	// --write and --check are mutually exclusive.
	if *writeFlag && *checkFlag {
		fmtUsageError("--write and --check are mutually exclusive")
	}

	switch {
	case *writeFlag:
		if len(files) == 0 {
			fmtUsageError("--write requires at least one file")
		}
		os.Exit(fmtWrite(files))
	case *checkFlag:
		if len(files) == 0 {
			fmtUsageError("--check requires at least one file")
		}
		os.Exit(fmtCheck(files))
	default:
		// Default stdout mode: exactly one file.
		if len(files) == 0 {
			fmtUsageError("expected exactly one file (stdout mode); use --write or --check for multiple files")
		}
		if len(files) > 1 {
			fmtUsageError("stdout mode accepts exactly one file; use --write or --check for multiple files")
		}
		os.Exit(fmtStdout(files[0]))
	}
}

// fmtUsageError prints a usage error to stderr and exits with code 2.
func fmtUsageError(msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", red("Error"), msg)
	fmt.Fprintln(os.Stderr, "Usage: ailang fmt <file.ail> | --write <files...> | --check <files...>")
	os.Exit(2)
}

// formatOne reads, comment-preflights, parses, formats, and round-trip-verifies a
// single file entirely in memory. It returns the original bytes, the canonical
// bytes, and an error. All failures are fail-closed: the caller performs no write
// and does not fall back to the original source.
func formatOne(path string) (orig []byte, canonical []byte, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}

	// Comment safety gate (Phase 1): refuse any file containing real comments.
	hasComments, cerr := format.HasComments(src)
	if cerr != nil {
		return nil, nil, fmt.Errorf("%s: comment preflight failed: %w", path, cerr)
	}
	if hasComments {
		return nil, nil, fmt.Errorf("%s: comments are not yet supported by ailang fmt", path)
	}

	prog, perr := parseForFmt(string(src), path)
	if perr != nil {
		return nil, nil, perr
	}

	out, ferr := format.Source(prog, format.Options{})
	if ferr != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, ferr)
	}

	// Round-trip verification: re-parse the formatted output and require a
	// structurally identical AST (ignoring only positions/spans). A mismatch is a
	// formatter defect and must fail the file, never silently ship bad output.
	reprog, rperr := parseForFmt(string(out), path)
	if rperr != nil {
		return nil, nil, fmt.Errorf("%s: formatted output failed to re-parse: %w", path, rperr)
	}
	if diff := cmp.Diff(prog.File, reprog.File, fmtIgnorePos); diff != "" {
		return nil, nil, fmt.Errorf("%s: round-trip verification failed (formatter defect); AST changed", path)
	}

	return src, out, nil
}

// parseForFmt parses source into a program, converting parse errors into a
// single path-qualified error.
func parseForFmt(src, path string) (*ast.Program, error) {
	p := parser.New(lexer.New(src, path))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("%s: parse error: %v", path, errs[0])
	}
	if prog == nil || prog.File == nil {
		return nil, fmt.Errorf("%s: parser produced no file", path)
	}
	return prog, nil
}

// fmtStdout formats one file and writes canonical source to stdout, leaving the
// file unchanged. Returns the process exit code.
func fmtStdout(path string) int {
	_, canonical, err := formatOne(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		return 2
	}
	if _, err := os.Stdout.Write(canonical); err != nil {
		fmt.Fprintf(os.Stderr, "%s: writing to stdout: %v\n", red("Error"), err)
		return 2
	}
	return 0
}

// fmtCheck reports every non-canonical file to stdout and never writes. Exit 0 if
// all inputs are canonical, 1 if any drift, 2 on any operational error.
func fmtCheck(paths []string) int {
	drift := false
	for _, path := range paths {
		orig, canonical, err := formatOne(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			return 2
		}
		// Canonical means byte-equal to formatter output, including final newline.
		if string(orig) != string(canonical) {
			fmt.Println(path)
			drift = true
		}
	}
	if drift {
		return 1
	}
	return 0
}

// fmtWrite validates ALL inputs in memory first (preflight + parse + print +
// round-trip); if any fails, no file is touched. Only then is each file replaced,
// individually and atomically, via a same-directory temp + os.Rename. Cross-file
// crash atomicity is not claimed.
func fmtWrite(paths []string) int {
	type job struct {
		path      string
		orig      []byte
		canonical []byte
	}
	jobs := make([]job, 0, len(paths))
	for _, path := range paths {
		orig, canonical, err := formatOne(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			return 2
		}
		jobs = append(jobs, job{path: path, orig: orig, canonical: canonical})
	}

	// All inputs validated. Replace each file that actually changed.
	for _, j := range jobs {
		if string(j.orig) == string(j.canonical) {
			continue // already canonical; leave the file (and its mtime) untouched
		}
		if err := atomicWriteFile(j.path, j.canonical); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			return 2
		}
	}
	return 0
}

// atomicWriteFile replaces path with data by staging a temporary file in the same
// directory, preserving the original file mode, then calling os.Rename. This is a
// local unexported helper (no shared safe-write helper exists in the repo — V19);
// it matches the inlined temp-file + os.Rename convention used elsewhere.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Preserve the original file mode.
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".ailang-fmt-*")
	if err != nil {
		return fmt.Errorf("%s: create temp: %w", path, err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if we bail before the rename.
	defer func() {
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: write temp: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s: close temp: %w", path, err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("%s: chmod temp: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("%s: rename: %w", path, err)
	}
	return nil
}

// printFmtHelp prints usage for the fmt subcommand.
func printFmtHelp() {
	fmt.Println("ailang fmt — format AILANG source into canonical form")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ailang fmt <file.ail>          Write canonical source to stdout (exactly one file)")
	fmt.Println("  ailang fmt --write <files...>  Rewrite each file in place (atomic per file)")
	fmt.Println("  ailang fmt --check <files...>  List files that are not canonical; exit 1 if any drift")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --write   Rewrite files in place. Validates every input first; if any file")
	fmt.Println("            fails preflight/parse/format/round-trip, NO file is changed.")
	fmt.Println("  --check   Report drift without writing. CI-friendly.")
	fmt.Println("  --help    Show this help.")
	fmt.Println()
	fmt.Println("Exit codes:")
	fmt.Println("  0  Formatting succeeded, or every file is canonical (--check).")
	fmt.Println("  1  --check found at least one non-canonical file.")
	fmt.Println("  2  Usage, read, comment, parse, print, round-trip, or write error.")
	fmt.Println()
	fmt.Println("Phase 1 limitation: files containing comments are refused (exit 2) and left")
	fmt.Println("byte-identical — the formatter never deletes comments. Lossless comment")
	fmt.Println("preservation is a separately-scheduled Phase 2.")
}
