package executor_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (decision D2(a), ratified by Mark 2026-08-27).
//
// No executor may name a model. The registry decides; an executor that cannot
// resolve one fails loudly.
//
// WHY THIS GUARD EXISTS AT ALL. Four of the ten literals were
// claude-haiku-4-5. Drop an agent's pin and it silently ran a model nobody
// chose, on an account nobody was watching — invisible until an invoice or an
// outage. A fallback that changes which model runs, and so what it costs, is a
// data-integrity fallback, which CLAUDE.md Critical Principle 2 forbids.
//
// NOT because "Claude OAuth is being retired" — an earlier version of this
// comment said that, and it is wrong. Anthropic announced that change for
// 2026-06-15 and PAUSED it that day (verified 2026-08-27); `claude -p` still
// draws on subscription. Cloud Run jobs also DO run Claude on subscription
// OAuth via CLAUDE_CODE_OAUTH_TOKEN. The defect never needed that premise.
//
// WHY IT SCANS factory.go AND NOT JUST THE FIVE SUBPACKAGES. The defect was TEN
// literals in two layers, not five. factory.go's DefaultConfig() fills
// cfg.*Model, so the per-executor defaults only ever fired when it had not.
// A guard scoped to internal/executor/<harness>/ would have passed while the
// upstream half kept every unpinned agent on Anthropic — it would have measured
// the wrong layer and reported success.
func TestNoExecutorNamesAModel(t *testing.T) {
	// Provider/model shapes that must never appear as a default in executor code.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"(openrouter/)?anthropic/[a-z0-9.-]+"`),
		regexp.MustCompile(`"claude-[a-z0-9.-]+"`),
		regexp.MustCompile(`"haiku"|"sonnet"|"opus"`),
		regexp.MustCompile(`"gpt-[0-9][a-z0-9.-]*"`),
		regexp.MustCompile(`"z-ai/[a-z0-9.-]+"`),
		regexp.MustCompile(`"(openrouter/)?(deepseek|moonshotai|minimax|qwen)/[a-z0-9.-]+"`),
	}

	var offenders []string
	scanned := 0

	err := filepath.Walk("../executor", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Tests may name models: pinning one explicitly is the CORRECT behavior
		// under D2(a), so forbidding it here would punish the fix.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		scanned++
		for i, line := range strings.Split(string(raw), "\n") {
			// Comments explain the history; only code counts.
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "//") {
				continue
			}
			for _, re := range patterns {
				if m := re.FindString(line); m != "" {
					offenders = append(offenders,
						filepath.ToSlash(path)+":"+itoa(i+1)+"  "+m)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	// Control: an absence assertion over zero files is vacuously true. The M1
	// lesson — the first version of that milestone's guard passed on an empty
	// package because its control could not fail.
	if scanned < 5 {
		t.Fatalf("instrument check failed: only %d non-test .go files scanned under "+
			"internal/executor; the assertion below would pass vacuously", scanned)
	}

	if len(offenders) > 0 {
		t.Errorf("executor code names %d model(s). The registry decides which model runs; "+
			"an executor that cannot resolve one must fail loudly (D2(a)):\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
