// Command agenttask renders the exact agent-mode task prompt the eval harness
// sends for a benchmark, so fine-tune episodes are built from the same text the
// model sees at eval time (m-eval-finetuning-data-pipeline, pilot-2).
//
// Usage:
//
//	go run ./tools/finetune/agenttask -solution /abs/ws/benchmark/solution.ail benchmarks/<id>.yml ...
//
// Prints one JSON object per line: {"id", "caps", "expected", "task_prompt"}.
// The system prompt is deliberately NOT emitted: the eval's system prompt is
// pi's own prompt plus the arm's teaching prompt, which the episode builder
// takes from a captured request.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func main() {
	solution := flag.String("solution", "", "absolute solution path to render into the prompt (required)")
	timeout := flag.Int("timeout", 1800, "timeout seconds rendered into the prompt")
	flag.Parse()
	if *solution == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: agenttask -solution /abs/ws/benchmark/solution.ail <spec.yml>...")
		os.Exit(2)
	}
	enc := json.NewEncoder(os.Stdout)
	failed := 0
	for _, path := range flag.Args() {
		spec, err := eval_harness.LoadSpec(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed++
			continue
		}
		cfg := eval_harness.AgentBenchmarkConfig{TimeoutSeconds: *timeout}
		_, task, _, err := eval_harness.GenerateAgentPromptsWithSystemPrompt(spec, cfg, "ailang", "", *solution)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed++
			continue
		}
		id := spec.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		_ = enc.Encode(map[string]any{"id": id, "caps": spec.Caps, "expected": spec.ExpectedOut, "task_prompt": task})
	}
	if failed > 0 {
		os.Exit(1)
	}
}
