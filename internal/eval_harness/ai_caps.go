package eval_harness

import "strings"

// aiHandlerArgs returns the extra `ailang run` flags needed when a benchmark
// grants the AI capability.
//
// M-EVAL-FAILURE-ATTRIBUTION: granting `--caps AI` without also configuring a
// handler is a guaranteed runtime failure — the CLI warns
// "--caps AI requires --ai <model> flag" and every AI.call fails. The harness
// was doing exactly that: it appended `--caps` and nothing else, so all three
// benchmarks declaring caps: ["AI", ...] (ai_effect_json_schema,
// ai_effect_summarize, multi_agent_handoff) failed with
// "no AI model configured" and the failure was banked as runtime_error —
// charged to the MODEL, for a flag the HARNESS forgot to pass.
//
// The stub, not a real model. A benchmark is graded by byte-exact stdout, so
// routing AI.call to a live provider would make the expected output depend on
// a second model's non-deterministic response — unrunnable by construction and
// a violation of A1 (Determinism). NewStubAIHandler exists for precisely this
// ("flag-shape testing without any real provider call") and costs nothing.
//
// Caps are matched case-insensitively because the value is author-written YAML.
//
// NOTE: wiring this does NOT make those three benchmarks pass. They are
// tier: vision and their expected_stdout is a prose placeholder
// (`<valid JSON with "name" and "age" keys, e.g. …>`), which cannot byte-match.
// This fix removes the HARNESS-caused failure so the remaining one is honestly
// attributable to the benchmark spec. Whether prose-spec vision benchmarks
// should be schedulable at all is a curation question, deliberately out of
// scope here.
func aiHandlerArgs(caps []string) []string {
	for _, c := range caps {
		if strings.EqualFold(strings.TrimSpace(c), "AI") {
			return []string{"--ai-stub"}
		}
	}
	return nil
}
