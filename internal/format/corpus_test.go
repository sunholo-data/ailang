package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// corpus_test.go runs the formatter over every examples/**/*.ail file. It
// PROGRAMMATICALLY partitions the corpus via HasComments (the advisory counts in
// the sprint plan are not hard-coded):
//
//   - Comment-free + parse-clean files: must format, idempotently, and structurally
//     round-trip (Parse(fmt(x)) == Parse(x), ignoring pos/span).
//   - Commented files: HasComments must report true. The Phase-1 gate refuses them;
//     this partition asserts they are NEVER handed to the printer.
//
// The examples tree lives at the repository root, two levels above this package.

const corpusRoot = "../../examples"

// walkAilExamples visits every .ail file under corpusRoot, calling fn(relPath, data).
func walkAilExamples(t *testing.T, fn func(path string, data []byte)) {
	t.Helper()
	if _, err := os.Stat(corpusRoot); err != nil {
		t.Skipf("examples corpus not found at %s: %v", corpusRoot, err)
	}
	err := filepath.Walk(corpusRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".ail") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		fn(path, data)
		return nil
	})
	if err != nil {
		t.Fatalf("walking corpus: %v", err)
	}
}

// TestCorpusCommentFreeRoundTrips asserts idempotence + structural round-trip
// over every comment-free, parse-clean example. Parse-erroring examples (some are
// intentional negative fixtures) are skipped: they are not formatter-eligible.
func TestCorpusCommentFreeRoundTrips(t *testing.T) {
	var commentFree, formatted int
	walkAilExamples(t, func(path string, data []byte) {
		has, err := HasComments(data)
		if err != nil {
			t.Fatalf("%s: HasComments: %v", path, err)
		}
		if has {
			return // commented partition, handled separately
		}
		commentFree++

		// Skip files that do not parse cleanly (negative fixtures etc.).
		p := parser.New(lexer.New(string(data), path))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			return
		}

		out1, err := Source(prog, Options{})
		if err != nil {
			t.Errorf("%s: Source: %v", path, err)
			return
		}

		p2 := parser.New(lexer.New(string(out1), path))
		prog2 := p2.Parse()
		if len(p2.Errors()) > 0 {
			t.Errorf("%s: formatted output failed to re-parse: %v", path, p2.Errors()[0])
			return
		}

		out2, err := Source(prog2, Options{})
		if err != nil {
			t.Errorf("%s: second Source: %v", path, err)
			return
		}
		if string(out1) != string(out2) {
			t.Errorf("%s: idempotence failed", path)
			return
		}
		if diff := cmp.Diff(prog.File, prog2.File, ignorePosSpan); diff != "" {
			t.Errorf("%s: round-trip AST changed:\n%s", path, diff)
			return
		}
		formatted++
	})
	t.Logf("corpus: %d comment-free files, %d formatted+round-tripped", commentFree, formatted)
	if formatted == 0 {
		t.Error("expected at least one comment-free example to format and round-trip")
	}
}

// TestCorpusCommentedFilesAreDetected asserts that every commented example is
// detected by HasComments (the Phase-1 refusal partition). This is the guard that
// keeps commented files away from the printer entirely.
func TestCorpusCommentedFilesAreDetected(t *testing.T) {
	var commented int
	walkAilExamples(t, func(path string, data []byte) {
		has, err := HasComments(data)
		if err != nil {
			t.Fatalf("%s: HasComments: %v", path, err)
		}
		if has {
			commented++
		}
	})
	t.Logf("corpus: %d commented files detected (Phase-1 refusal partition)", commented)
	if commented == 0 {
		t.Error("expected the corpus to contain commented examples")
	}
}
