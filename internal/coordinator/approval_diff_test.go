package coordinator

import (
	"strings"
	"testing"
)

const samplePatch = `diff --git a/docs/internal/probe.md b/docs/internal/probe.md
new file mode 100644
--- /dev/null
+++ b/docs/internal/probe.md
@@ -0,0 +1,2 @@
+line one
+line two
diff --git a/internal/x.go b/internal/x.go
--- a/internal/x.go
+++ b/internal/x.go
@@ -1,3 +1,2 @@
-old
+new
`

func TestFilesFromPatch(t *testing.T) {
	files := filesFromPatch(samplePatch)
	if len(files) != 2 || files[0] != "docs/internal/probe.md" || files[1] != "internal/x.go" {
		t.Errorf("got %v", files)
	}
}

func TestDiffStatFromPatch(t *testing.T) {
	// +++/--- headers must not count as changes: 3 adds, 1 del, 2 files.
	got := diffStatFromPatch(samplePatch)
	if got != "+3 -1 across 2 files" {
		t.Errorf("got %q", got)
	}
	one := diffStatFromPatch(strings.Split(samplePatch, "diff --git a/internal")[0])
	if !strings.HasSuffix(one, "1 file") {
		t.Errorf("singular noun expected, got %q", one)
	}
}

// A truncated stored diff must SAY so — an approval card silently showing a
// partial diff is the same blind-approve failure with better production values.
func TestCapPatch(t *testing.T) {
	long := strings.Repeat("x", 100)
	capped := capPatch(long, 10)
	if !strings.Contains(capped, "truncated") {
		t.Errorf("truncation must be marked, got %q", capped)
	}
	if capPatch("short", 10) != "short" {
		t.Error("under-cap patch must pass through unchanged")
	}
}
