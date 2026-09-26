package main

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// updateCLIReference is set by `make docs-cli`. Without it this test is the
// CI diff gate; with it, it is the generator.
var updateCLIReference = flag.Bool("update-cli-reference", false, "rewrite docs/docs/reference/cli.md from the dispatch table")

func cliReferenceFile() string { return filepath.Join("..", "..", cliReferencePath) }

// TestCLIReferenceMatchesTable is the gate. It renders FRESH from the dispatch
// table and compares against the CHECKED-IN page — that direction on purpose:
// a gate that reads the generated artifact measures the build, not the intent,
// which is how this sprint's prompt-truth fix passed on a clean checkout and
// went red after the next build (the fix had been applied to the mirror the
// build overwrites).
func TestCLIReferenceMatchesTable(t *testing.T) {
	want := renderCLIReference()
	path := cliReferenceFile()

	if *updateCLIReference {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatalf("write %s: %v", cliReferencePath, err)
		}
		t.Logf("wrote %s (%d bytes)", cliReferencePath, len(want))
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v\nRun 'make docs-cli' to generate it.", cliReferencePath, err)
	}
	// Compare content, not line endings: `* text=auto` checks the page out
	// with CRLF on Windows, where this gate reported line 1 (`---`) as
	// differing from itself (dev red at 89a984dd6, test-windows only).
	if normalizeLineEndings(string(got)) == normalizeLineEndings(string(want)) {
		return
	}
	t.Errorf("%s is stale: the dispatch table and the page disagree.\n"+
		"Run 'make docs-cli' and commit the result.\n%s",
		cliReferencePath, firstDifference(string(got), string(want)))
}

// firstDifference names the first line that differs, so the failure says WHICH
// command drifted instead of printing two 6 KB documents.
func firstDifference(got, want string) string {
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	n := len(gotLines)
	if len(wantLines) < n {
		n = len(wantLines)
	}
	for i := 0; i < n; i++ {
		if gotLines[i] != wantLines[i] {
			return "first difference at line " + strconv.Itoa(i+1) +
				":\n  page:  " + gotLines[i] + "\n  table: " + wantLines[i]
		}
	}
	return "the page has " + strconv.Itoa(len(gotLines)) +
		" lines, the table renders " + strconv.Itoa(len(wantLines))
}

// TestCLIReferenceCoversEveryRoute is the arm the byte-compare cannot be: the
// gate above still passes if renderCLIReference silently drops a section, as
// long as the checked-in page drops it too. This one asserts against the TABLE
// that every route reaches the page.
func TestCLIReferenceCoversEveryRoute(t *testing.T) {
	page := string(renderCLIReference())
	for i := range allCommands {
		c := &allCommands[i]
		if isGroupName(c.Name) {
			continue // the drawers are headings, not rows
		}
		for _, route := range append([]string{c.Name}, c.Aliases...) {
			if !strings.Contains(page, "`ailang "+route+"`") {
				t.Errorf("route %q is in the dispatch table but not on the CLI reference page", route)
			}
		}
	}
}

// normalizeLineEndings folds CRLF to LF so a checkout under `text=auto` on
// Windows compares equal to the LF text the renderer produces.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
