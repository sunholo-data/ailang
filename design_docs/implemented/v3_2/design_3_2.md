AILANG v3.2 — Final Design Specification

The AI-First Programming Language

⸻

Executive Summary

AILANG is a purely functional language designed for AI-assisted programming, with determinism, explicitness, and machine readability as its core. It minimizes ambiguity for LLM-based coding agents, while remaining usable for humans.
	•	Reactive loop: Code → errors/tests → structured feedback → fixes.
	•	Proactive loop (new): AI can propose plans before coding, validate architecture, and scaffold safely.
	•	Context drift protection: Decisions, snapshots, and stable identifiers prevent AI agents from “losing the plot” across sessions.

File extension: .ail

⸻

Design Principles
	1.	AI-First
	•	ANF Core representation
	•	Stable node IDs (SIDs) across transformations
	•	Decision ledger for “why”
	•	Token-lean output (compact JSON modes)
	2.	Explicitness Everywhere
	•	Errors as values
	•	All effects in type signatures
	•	Dictionary passing, no hidden resolution
	3.	Deterministic Execution
	•	Lockfiles for environments
	•	Snapshots for known-good checkpoints
	•	Replay from decision ledger
	4.	Context Drift Protection
	•	Decision ledger (persistent trail of type/constraint choices)
	•	Micro-traces (slice of execution state)
	•	Compact mode for token efficiency
	5.	Proactive Planning (NEW)
	•	:propose to validate structured module plans
	•	:scaffold to generate boilerplate from plans
	•	Plans become artifacts for training and reasoning

⸻

Core Type System

(same as v2/v3 with Eq, Ord, Num, Fractional, Show classes; explicit effect rows; row-polymorphic records)

⸻

New Features in v3.1 → v3.2

Error-as-Values

All runtime/type errors are structured values:

{
  "sid": "N#42",
  "phase": "typecheck",
  "message": "defaulting failed for Num",
  "fix": { "suggestion": "add type annotation", "confidence": 0.9 },
  "context": {
    "constraints": ["Num α2", "α2 ~ Int"],
    "decisions": ["defaulted α2 -> Int"],
    "trace_slice": ":trace-slice N#42"
  }
}

:effects Introspection

λ> :effects 1 + 2
[Int] requires {Num}

Lets AI agents check effects/types before evaluation.

Test Reporter

$ ailang :test --json
{
  "suite": "user.ail",
  "summary": { "passed": 9, "failed": 1 },
  "tests": [
    { "name": "reverse empty", "status": "pass" },
    { "name": "division by zero", "status": "fail", "error": { ...Error } }
  ]
}

Planning & Scaffolding Protocol (NEW v3.2)
	•	:propose plan.json → validates structured plan
	•	:scaffold --from-plan plan.json → generates boilerplate

(see schemas in earlier message)

⸻

REPL Commands (v3.2)
	•	:why — last 3 decisions from ledger
	•	:trace-slice <sid> — show node journey
	•	:dump-core — ANF Core view
	•	:snapshot <name> — checkpoint
	•	:replay <ledger> — reconstruct state
	•	:effects <expr> — introspect effects
	•	:propose plan.json — validate architecture plan
	•	:scaffold --from-plan plan.json — generate skeleton code

⸻

Implementation Priorities
	1.	Error JSON encoder ✅
	2.	Test reporter (:test --json) ✅
	3.	:effects command ✅
	4.	Planning protocol (:propose, :scaffold)

⸻

Go Scaffolds

1. Error JSON Encoder

// internal/errors/json_encoder.go
package errors

import (
	"encoding/json"
)

type AILANGError struct {
	SID      string      `json:"sid"`
	Phase    string      `json:"phase"`
	Message  string      `json:"message"`
	Fix      *ErrorFix   `json:"fix,omitempty"`
	Context  *ErrorContext `json:"context,omitempty"`
}

type ErrorFix struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

type ErrorContext struct {
	Constraints []string `json:"constraints"`
	Decisions   []string `json:"decisions"`
	TraceSlice  string   `json:"trace_slice"`
}

func (e AILANGError) ToJSON() string {
	data, _ := json.MarshalIndent(e, "", "  ")
	return string(data)
}

Usage:

err := AILANGError{
  SID: "N#42",
  Phase: "typecheck",
  Message: "defaulting failed for Num",
}
fmt.Println(err.ToJSON())


⸻

2. Test Reporter

// internal/test/reporter.go
package test

import (
	"encoding/json"
	"fmt"
)

type TestResult struct {
	Name   string        `json:"name"`
	Status string        `json:"status"`
	Error  interface{}   `json:"error,omitempty"`
}

type TestSuite struct {
	Suite   string       `json:"suite"`
	Summary Summary      `json:"summary"`
	Tests   []TestResult `json:"tests"`
}

type Summary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

func ReportJSON(suite string, results []TestResult) {
	summary := Summary{}
	for _, r := range results {
		if r.Status == "pass" {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	report := TestSuite{Suite: suite, Summary: summary, Tests: results}
	data, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(data))
}


⸻

3. Effects Inspector

// internal/repl/effects.go
package repl

import (
	"fmt"
	"ailang/internal/types"
)

func InspectEffects(expr string) {
	// TODO: parse + typecheck expr
	// Placeholder for demonstration:
	effects := []string{"FS", "Net"}
	typ := "Result[string, IOError]"
	fmt.Printf("Type: %s\nEffects: %v\n", typ, effects)
}


⸻

Conclusion

AILANG v3.2 is the most AI-friendly PL design to date, combining:
	•	Reactive clarity (structured errors/tests/effects)
	•	Proactive scaffolding (plans + proposals)
	•	Context stability (lockfiles, snapshots, replay)
	•	Token efficiency (compact JSON modes)

It positions itself not just as a Haskell/ML-inspired language, but as the first language explicitly designed for AI agents to generate, debug, and evolve code safely.