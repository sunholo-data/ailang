package pkg

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/strutil"
)

const smokeNotesLimit = 2048

// CLI color/progress uses CSI; OSC covers terminal hyperlinks in diagnostics.
var smokeANSI = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

func cleanSmokeOutput(s string) string {
	s = strings.ToValidUTF8(s, "�")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	return strings.TrimSpace(smokeANSI.ReplaceAllString(s, ""))
}

func firstSmokeDiagnostic(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		text := strings.TrimSpace(line)
		lower := strings.ToLower(text)
		for _, prefix := range []string{"error:", "panic:", "fatal error:", "fail:", "par_"} {
			if strings.HasPrefix(lower, prefix) {
				return text
			}
		}
	}
	for _, line := range lines {
		text := strings.TrimSpace(line)
		lower := strings.ToLower(text)
		if text == "" {
			continue
		}
		noise := false
		for _, prefix := range []string{"warning:", "info:", "info ", "debug:", "debug ", "trace:", "trace ", "at ", "goroutine ", "→", "✓"} {
			if strings.HasPrefix(lower, prefix) {
				noise = true
				break
			}
		}
		if strings.TrimLeft(line, " \t") != line && strings.Contains(text, ".go:") {
			noise = true
		}
		if !noise {
			return text
		}
	}
	return ""
}

func smokeCap(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	marker := " [truncated]"
	if len(marker) > limit {
		marker = ""
	}
	return strutil.TruncateBytes(s, limit, marker)
}

// smokeTail reserves its marker inside the budget and starts at a rune boundary.
func smokeTail(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	marker := "[...]\n"
	if limit <= len(marker) {
		return ""
	}
	start := len(s) - (limit - len(marker))
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return marker + s[start:]
}

func summarizeSmokeOutput(output string, limit int, forcedHead string) string {
	clean := cleanSmokeOutput(output)
	diagnostic := firstSmokeDiagnostic(clean)
	head := forcedHead
	if head == "" {
		head = diagnostic
	} else if diagnostic != "" && diagnostic != head {
		head += ": " + diagnostic
	}
	if head == "" {
		return smokeTail(clean, limit)
	}
	if forcedHead == "" && len(clean) <= limit && strings.HasPrefix(clean, head) {
		return clean
	}
	head = smokeCap(head, limit)
	if clean == "" || clean == head {
		return head
	}
	separator := "\n[context]\n"
	remaining := limit - len(head) - len(separator)
	if remaining <= len("[...]\n") {
		return head
	}
	return head + separator + smokeTail(clean, remaining)
}

// SmokeFailureNotes preserves setup/process/timeout causes using the existing
// quality Notes field. Its byte budget includes context, prefixes and markers.
func SmokeFailureNotes(result *SmokeResult, runErr error, timeout time.Duration) string {
	if runErr != nil {
		return summarizeSmokeOutput(runErr.Error(), smokeNotesLimit, "")
	}
	if result == nil {
		return "smoke returned no result"
	}
	if result.Passed {
		return ""
	}
	head := ""
	if result.TimedOut {
		head = fmt.Sprintf("timed out after %s", timeout)
	} else if firstSmokeDiagnostic(cleanSmokeOutput(result.Output)) == "" {
		head = fmt.Sprintf("smoke exited with code %d", result.ExitCode)
	}
	return summarizeSmokeOutput(result.Output, smokeNotesLimit, head)
}
