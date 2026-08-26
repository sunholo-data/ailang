package coordinator

import (
	"fmt"
	"strings"
)

// #921 support: helpers to summarize a git patch for the approval record.
//
// approvalDiffCapBytes bounds the persisted patch. Firestore documents cap at
// ~1MB; 200KB leaves generous room for the rest of the record while covering
// any reviewable diff — a patch bigger than this should be read in a real
// tool, and the stat + file list still say what moved.
const approvalDiffCapBytes = 200 * 1024

// filesFromPatch lists the paths a unified diff touches, in order.
func filesFromPatch(patch string) []string {
	var files []string
	seen := map[string]bool{}
	for _, line := range strings.Split(patch, "\n") {
		if !strings.HasPrefix(line, "diff --git a/") {
			continue
		}
		// "diff --git a/path b/path" — take the b/ side so renames show the
		// destination the merge would produce.
		if i := strings.LastIndex(line, " b/"); i >= 0 {
			f := line[i+3:]
			if f != "" && !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files
}

// diffStatFromPatch renders a one-line "+N -M across K files" summary.
func diffStatFromPatch(patch string) string {
	add, del := 0, 0
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			add++
		case strings.HasPrefix(line, "-"):
			del++
		}
	}
	files := len(filesFromPatch(patch))
	noun := "files"
	if files == 1 {
		noun = "file"
	}
	return fmt.Sprintf("+%d -%d across %d %s", add, del, files, noun)
}

// capPatch bounds a patch for storage, marking the truncation honestly rather
// than serving a silently incomplete diff.
func capPatch(patch string, capBytes int) string {
	if len(patch) <= capBytes {
		return patch
	}
	return patch[:capBytes] + "\n… (diff truncated for storage; full diff on the owning machine)"
}
