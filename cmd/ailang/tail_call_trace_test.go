package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateTailGoldens = flag.Bool("update-tailcall-goldens", false, "rewrite testdata/tailcall/*.trace.golden from the current binary")

// M-EVAL-TAIL-CALLS (#1486) D6: tail-call elimination must leave the deep
// trace's function events exactly as nested evaluation produced them — same
// names, args, results, depths and span parentage, in the same order. The
// goldens were captured from the evaluator BEFORE tail calls were eliminated,
// so this compares against real pre-change output. Timestamps, durations and
// trace ids vary per run and are dropped; span ids are renumbered by first
// appearance. Failure paths are covered in internal/eval (a failed run emits no
// trace events at the CLI, so they are not observable here).
func TestTailCallTraceMatchesNestedEvaluation(t *testing.T) {
	bin := buildAilang(t)
	cases := []struct {
		file  string
		n     string
		extra []string
	}{
		{"loop", "3", nil},
		{"mutual_io", "3", []string{"--caps", "IO"}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			src := filepath.Join("cmd", "ailang", "testdata", "tailcall", c.file+".ail")
			args := append([]string{"run", "--quiet", "--emit-trace", "jsonl", "--trace-tier", "deep"}, c.extra...)
			args = append(args, "--entry", "main", "--args-json", c.n, src)
			stdout, stderr, code := runAilangBin(t, bin, args...)
			got := normalizeTrace(t, stdout) + fmt.Sprintf("exit=%d\nerror=%s\n", code, firstErrorLine(stderr))
			golden := filepath.Join("testdata", "tailcall", c.file+".trace.golden")
			if *updateTailGoldens {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden %s: %v", golden, err)
			}
			if got != string(want) {
				t.Errorf("trace differs from nested-evaluation golden %s\n--- got ---\n%s--- want ---\n%s", golden, got, want)
			}
		})
	}
}

// normalizeTrace keeps program output lines verbatim and rewrites each trace
// event without its run-varying fields.
func normalizeTrace(t *testing.T, stdout string) string {
	t.Helper()
	spans := map[string]int{}
	canon := func(id any) any {
		s, ok := id.(string)
		if !ok || s == "" {
			return id
		}
		if _, seen := spans[s]; !seen {
			spans[s] = len(spans) + 1
		}
		return fmt.Sprintf("s%d", spans[s])
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.ReplaceAll(stdout, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "{") {
			if line != "" {
				b.WriteString("out: " + line + "\n")
			}
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("bad trace line %q: %v", line, err)
		}
		delete(ev, "timestamp_ns")
		delete(ev, "trace_id")
		ev["span_id"] = canon(ev["span_id"])
		if p, ok := ev["parent_span_id"]; ok {
			ev["parent_span_id"] = canon(p)
		}
		for _, k := range []string{"function", "module", "effect"} {
			if m, ok := ev[k].(map[string]any); ok {
				delete(m, "duration_ns")
			}
		}
		keys := make([]string, 0, len(ev))
		for k := range ev {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			v, _ := json.Marshal(ev[k])
			parts[i] = k + "=" + string(v)
		}
		b.WriteString(strings.Join(parts, " ") + "\n")
	}
	return b.String()
}

func firstErrorLine(stderr string) string {
	for _, l := range strings.Split(stderr, "\n") {
		if strings.Contains(l, "Error") || strings.Contains(l, "error") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}
