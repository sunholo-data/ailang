
🧭 AILANG Deferred Features (Post-Vision Alignment)

Version: 2025-10-21
Supersedes: design_docs/20251018_deferred_features.md
Context: After Vision Benchmark adoption and AI-First Design Principles

⸻

1. Purpose

This document tracks deferred or re-scoped features after the Vision transition.
Earlier sprints (v0.3.12–v0.3.15) were designed around human-developer ergonomics (e.g., LSP, fix-its, wildcard imports).
AILANG is now formally repositioned as an AI-First deterministic language, optimized for agent collaboration, reproducibility, and low token entropy.

⸻

2. AI-First Design Filter

Every feature must satisfy at least one of the following:

✅ Reduce syntactic entropy
✅ Preserve semantic clarity
✅ Increase determinism
✅ Lower token cost

Features that exist solely for human convenience (e.g., IDE integrations, syntax sugar) are now secondary.

⸻

3. Updated Classification of Deferred Items

Feature	Old Version	New Classification	New Target	Rationale
import M (*)	v0.3.16	🟡 Optional Ergonomics	v0.3.17	Convenience feature, minimal AI impact. Safe post-benchmark cleanup.
FX001 Fix-It (! {IO})	v0.3.16	🟢 AI-DX Core	v0.3.17	Evolves into auto-capability inference; high value for token and DX reduction.
LSP / Local Daemon	v0.4.0	🔴 Human-DX Only	⟶ Reframed as Agent Bridge (v0.4.0)	Replace human IDE loop with MCP/A2A AI protocol.
Effect Composer	v0.4.1	🟢 AI-DX Core	v0.4.1	Enables auto-capability inference and minimal effect sets.
Test Generator	v0.4.2	🟡 Training Aid	v0.4.2	Generates benchmarks & dataset examples; not runtime-critical.
Extended JSON (Schema/Streaming)	v0.4.3	🟢 Runtime Capability	v0.4.3	Expands JSON support for structured reasoning and PPA data use cases.


⸻

4. Integration with Vision Benchmarks

Vision Benchmark	Required Feature	Status / Target
Referential Transparency	Deterministic Builtins, normalize	✅ v0.3.15
Canonical Code Structure	Prelude + Normalizer	✅ v0.3.16
Multi-Agent Collaboration	Agent Bridge (MCP/A2A)	🔜 v0.4.0
Token Efficiency	Auto-Capability Composer	🔜 v0.4.1
Deterministic Replay	Canonical AST + Schema Verification	✅ v0.3.15
AI Tool Interop	Agent Bridge Protocol (ABP)	🔜 v0.4.0


⸻

5. Revised Roadmap Sequence

Version	Focus	Core Deliverables	AI-First Impact
v0.3.15	Deterministic Tooling	normalize, suggest, apply, schema gates	Deterministic transformation pipeline
v0.3.16	Benchmark Parity	Entry-module prelude, parse fixes, CI gates	Minimal syntactic entropy for examples
v0.3.17	Auto-Capability Inference	FX001 → effect composer prototype	Lower token cost, better AI inference
v0.4.0	Multi-Agent Runtime	Agent Bridge (ABP) replacing LSP	Parallel AI coding, MCP/A2A compatible
v0.4.1	Effect Composer	Static + dynamic capability inference	True auto-capability inference
v0.4.2	Test Generator	Property-based synthesis	Model training, coverage expansion
v0.4.3	Extended JSON	Schema validation + streaming	Larger datasets, structured inputs
v0.4.4	Token Optimization	Canonical import pruning	Benchmark efficiency, minimal entropy


⸻

6. Feature Details

🟡 Import Wildcard Syntax: import M (*)

Purpose: Convenience for AI-generated imports; expands at parse time.
Design:
	•	Parser recognizes (*), elaborator replaces with full export list.
	•	Deterministic (no runtime lookup).
Reason for Defer: Parser churn not justified during benchmark stabilization.
Target: v0.3.17 (post-benchmark).

⸻

🟢 FX001 Diagnostic → Auto Capability Inference

Purpose: Replace static fix-it with effect inference engine.
Design:
	•	Detect missing effects from call graph.
	•	Suggest or apply ! {E} automatically.
	•	Integrate with normalize and apply.
Value: Core AI-DX feature — AIs emit correct code without manual effect tuning.
Target: v0.3.17–v0.4.1 (unified with Effect Composer).

⸻

🔵 Agent Bridge (Replaces LSP)

Purpose: AI-to-AI protocol for orchestration (MCP/A2A compatible).
Design:
	•	Deterministic JSON-RPC protocol.
	•	Calls: normalize, suggest, apply, verify, eval.
	•	Compatible with A2A & MCP protocol standards.
Impact: Enables multi-agent collaboration, deterministic coding swarms.
Target: v0.4.0.

⸻

🟢 Effect Composer

Purpose: Infer minimal ! {} effect sets automatically.
Example:

func process() {
  let text = readFile("x")   -- requires FS
  println(text)              -- requires IO
}
-- becomes:
func process() -> () ! {IO, FS}

Target: v0.4.1.
Impact: Core to AI code correctness and minimal capability exposure.

⸻

🟡 Test Generator

Purpose: Generate property-based tests from type signatures for AI training data.
Example:
add(x,y) → commutativity, associativity, identity tests.
Target: v0.4.2.
Impact: AI training, benchmarking, and continuous self-validation.

⸻

🟢 Extended JSON Features

Purpose: Support for structured reasoning, large input streams, and schema validation.
Scope:
	•	Unicode escape sequences (\uXXXX)
	•	Streaming decode
	•	Schema validation
	•	decodeInto[T] (typed decode)
Target: v0.4.3.
Impact: Enables structured data ingestion and analysis workloads.

⸻

7. Philosophical Principles for Future Sprints

AI-First Design Principle:
Every new feature must reduce syntactic entropy or improve deterministic replay.

Reject features that:
- Exist solely for human convenience (IDE UX, syntax sugar)
- Introduce nondeterminism in parsing, evaluation, or imports
- Increase token cost per operation


⸻

8. Decision Log

Decision	Date	Summary
Sprint Split (v0.3.12 → v0.3.15)	2025-10-18	Original 38-hour sprint split into JSON Decode (Phase 1) and Tooling (Phase 2).
Vision Realignment	2025-10-21	Re-classified roadmap under AI-First DX and Vision benchmarks.
Remove LSP / Replace with Agent Bridge	2025-10-21	LSP no longer relevant; AI agents use MCP/A2A protocols.
Auto Capability Composer replaces FX001	2025-10-21	Fold diagnostic fix-it into capability inference engine.


⸻

9. Future Tracking
	•	Milestone: “v0.3.17 – AI-DX Ergonomics”
	•	Issues to Create:
	•	#AIDEX-17-001: Auto-capability inference prototype (FX001→Composer)
	•	#AIDEX-17-002: Import wildcard syntax desugaring
	•	#AIDEX-40-001: Agent Bridge Protocol (MCP/A2A)
	•	#AIDEX-41-001: Static + dynamic effect composer
	•	#AIDEX-42-001: Property-based test generator
	•	#AIDEX-43-001: Extended JSON streaming/schema validation

⸻

10. Summary

Scope	Status	Category	Priority
JSON Decode (v0.3.14)	✅ Done	Core	P0
Deterministic Tooling (v0.3.15)	🏗️ In Progress	Core	P0
Prelude + Benchmarks (v0.3.16)	✅ Done	Core	P0
Import Wildcard + FX001	⏳ Planned (v0.3.17)	Ergonomic / AI-DX	P1
Agent Bridge + Composer	🔜 v0.4.0 – v0.4.1	Core AI-DX	P0
Test Generator / Extended JSON	🕓 v0.4.2 – v0.4.3	Optional Runtime	P2


⸻

📘 References
	•	design_docs/implemented/M-LANG-JSON-DECODE.md
	•	design_docs/implemented/M-DX1-Developer-Experience.md
	•	design_docs/planned/20251013_auto_caps_capability_inference.md
	•	benchmarks/VISION_BENCHMARKS.md
	•	prompts/v0.3.16.md (Entry-Module Prelude Prompt)

⸻

Summary:
This document supersedes the original v0.3.12–15 sprint ticket and redefines all deferred features under the Vision-aligned AI-First philosophy.
The next immediate step is v0.3.17: Auto Capability Inference — merging FX001, normalize, and effect composer logic into one unified inference path, paving the way for multi-agent parallel coding in v0.4.0.
