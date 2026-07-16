package quorum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ArtifactDir is where machine-readable quorum verdicts land. It seeds Phase E
// (the assignment-table evidence). One JSON per doc per run.
const ArtifactDir = ".ailang/state/mission-quorum"

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a doc path into a filesystem-safe slug for the artifact name.
func Slug(docPath string) string {
	base := filepath.Base(docPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ToLower(base)
	base = slugRe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "doc"
	}
	return base
}

// WriteJSONArtifact writes the machine-readable quorum verdict to
// <dir>/<slug>-<iso8601>.json and returns the path. The iso timestamp in the
// filename is sanitized (colons → dashes) for filesystem safety.
//
// The write is collision-proof: two runs on the same doc within the same
// second (or the same injected timestamp in tests) would otherwise share a
// name and clobber, so we open with O_CREATE|O_EXCL and, on collision, retry
// with a numeric suffix. Every run yields its own file (Principle 2 — evidence
// is never silently overwritten).
func WriteJSONArtifact(dir string, q *QuorumResult) (string, error) {
	if dir == "" {
		dir = ArtifactDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create artifact dir: %w", err)
	}
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal quorum: %w", err)
	}
	tsSafe := strings.NewReplacer(":", "-", "/", "-").Replace(q.ISOTimestamp)
	base := filepath.Join(dir, fmt.Sprintf("%s-%s", Slug(q.Doc), tsSafe))
	// Try <base>.json first, then <base>-1.json, <base>-2.json, ... until an
	// O_EXCL create succeeds — no existing artifact is ever clobbered.
	for i := 0; ; i++ {
		path := base + ".json"
		if i > 0 {
			path = fmt.Sprintf("%s-%d.json", base, i)
		}
		f, oerr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if os.IsExist(oerr) {
			continue
		}
		if oerr != nil {
			return "", fmt.Errorf("create artifact: %w", oerr)
		}
		_, werr := f.Write(b)
		cerr := f.Close()
		if werr != nil {
			return "", fmt.Errorf("write artifact: %w", werr)
		}
		if cerr != nil {
			return "", fmt.Errorf("close artifact: %w", cerr)
		}
		return path, nil
	}
}

// MarkdownBlock renders the human-readable routing-evidence block appended to
// the mission log (Gate 4). One line per reviewer + the synthesis verdict +
// any named absentees. Deterministic (no map iteration).
func MarkdownBlock(q *QuorumResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#### Design-quorum review — `%s` (%s)\n\n", q.Doc, q.ISOTimestamp)
	fmt.Fprintf(&b, "- **Synthesis: %s** (total $%.4f)\n", strings.ToUpper(string(q.Synthesis.Verdict)), q.Synthesis.TotalCostUSD)
	for _, o := range q.Reviewers {
		if o.Present {
			fmt.Fprintf(&b, "- `%s` → **%s** ($%.4f) — %s\n", o.Model, o.Result.Verdict, o.CostUSD, o.Result.StrongestObjection)
		} else {
			fmt.Fprintf(&b, "- `%s` → **ABSENT** (%s) — degraded to N-1, not a silent pass\n", o.Model, o.AbsentReason)
		}
	}
	if q.ControllerInSession != nil {
		fmt.Fprintf(&b, "- controller (in-session, not an API call) → **%s** — %s\n", q.ControllerInSession.Verdict, q.ControllerInSession.Note)
	}
	if q.Tier2 != nil {
		fmt.Fprintf(&b, "- **Tier-2 escalation** (%s) — verified the contested premise\n", q.Tier2.Decision.Reason)
		for _, o := range q.Tier2.Reviewers {
			if o == nil {
				continue
			}
			if o.Present {
				fmt.Fprintf(&b, "  - `%s` (tier2) → **%s** ($%.4f) — %s\n", o.Model, o.Result.Verdict, o.CostUSD, o.Result.StrongestObjection)
			} else {
				fmt.Fprintf(&b, "  - `%s` (tier2) → **ABSENT** (%s) — degraded to N-1, not a silent pass\n", o.Model, o.AbsentReason)
			}
		}
	}
	if len(q.Synthesis.BlockingObjections) > 0 {
		b.WriteString("- Blocking objections (return to author before planning):\n")
		for _, obj := range q.Synthesis.BlockingObjections {
			fmt.Fprintf(&b, "  - %s\n", obj)
		}
	}
	return b.String()
}

// AppendMarkdownToLog appends the markdown block to the mission log file. If
// logPath is empty, it is a no-op returning ("", nil) — the caller may prefer
// to print the block for the controller to paste. Creating the file is allowed
// (the log may not exist in a test fixture) but NOT the parent dirs beyond one
// level, to avoid surprising writes.
func AppendMarkdownToLog(logPath string, q *QuorumResult) (string, error) {
	if logPath == "" {
		return "", nil
	}
	block := "\n" + MarkdownBlock(q)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open mission log %s: %w", logPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		return "", fmt.Errorf("append to mission log: %w", err)
	}
	return logPath, nil
}
