#!/usr/bin/env bash
# routed_docs.sh — the labelled set for the M-AI-DECIDE-SYSTEM-ONE shadow lane-router.
#
# Emits one JSON row per design doc that DECLARES its PROGRAM.md §4 lane in
# prose, with the label, the quoted sentence it was read from, and the state the
# decision model will see (problem statement + goals + files, ≤ 6k chars).
#
# The labels were read by hand on 2026-09-18 from the sentence quoted in each
# row (`evidence`) — the docs state the lane in too many spellings for a regex
# to be trusted, and the design doc's "45 routed docs" turned out to be 45 docs
# with a *routing section*, of which these are the ones whose lane is legible.
# The `evidence` field is what makes the label auditable. Add a row by adding a
# line below; never by editing the extractor.
#
# Usage: tools/decisions/routed_docs.sh > .ailang/state/decisions/labels.jsonl
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# path | lane | evidence (verbatim fragment from the doc)
# (a plain heredoc, not $(…): bash 3.2 mis-parses quotes inside a command
# substitution heredoc, and the rig shell is bash 3.2)
python3 "$(dirname "$0")/routed_docs.py" <<'TSV'
design_docs/implemented/v0_30_0/m-eval-elo-priority-rotation.md	extension	**Lane: extension** (rig tooling). The change lives entirely in `tools/launchd/os-rotation-filler.sh`
design_docs/implemented/v0_30_0/m-gemini-repo-mount.md	extension	executor/eval-harness plumbing in the extension lane; **no core-floor change**
design_docs/implemented/v0_30_0/m-mission-adaptive-multiprovider-routing.md	extension	Harness/mission-infrastructure lane — **not** a motoko-core change, **not** an AILANG language feature
design_docs/implemented/v0_32_0/m-eval-standard-confidence-gating.md	extension	**Lane: extension** (eval harness tooling). Nothing in `internal/{parser,...}` is touched.
design_docs/planned/20260918_timeout_python_timeout.md	extension	Extension lane only; no core-floor changes.
design_docs/planned/20260918_compile_error_ailang_compilation_failures.md	extension	Introduce a Compat Front-End (compat) as an extension lane
design_docs/planned/m-motoko-fmt-remeasurement-instrument.md	extension	mission default bias holds: everything here is harness lane or extension lane
design_docs/planned/v0_30_0/m-diag-primitive-field-suggestions.md	extension	**Lane**: Extension (per PROGRAM.md "default bias: if it can be an extension, it is an extension")
design_docs/planned/v0_31_0/m-decision-entropy-monitor.md	extension	monitoring/extension lane — zero language surface
design_docs/planned/v0_31_0/m-mem-budget-runtime.md	extension	**Lane**: Extension (PROGRAM.md routing — runtime capability, zero core-syntax change)
design_docs/planned/v1_1_0/m-game-engine-effects.md	extension	routed through the extension lane (apps/ or pkg/ citizen, NOT compiler core)
design_docs/planned/v0_29_0/m-contracts-as-code-vertical.md	extension	**Extension, not core — and the package now exists.**
design_docs/implemented/v0_30_0/m-module-less-run-fail-loud.md	ailang_fix	**AILANG fix** — `internal/pipeline` + a new `MOD014` diagnostic. Not motoko, not core.
design_docs/implemented/v1_0_0/m-v1-memory-footprint.md	ailang_fix	**Lane**: AILANG fix (PROGRAM.md §4 — runtime substrate defects; no motoko surface).
design_docs/planned/m-list-cons-quadratic.md	ailang_fix	**Lane: AILANG fix.** Argued against the program's default bias
design_docs/planned/v0_35_0/m-partial-accessor-shape.md	ailang_fix	**Lane**: AILANG fix (stdlib + CLI). No motoko surface, no core change.
design_docs/planned/v0_40_0/m-pkg-quality-ladder.md	ailang_fix	**Lane** (PROGRAM.md): AILANG fix / tooling — registry validator, `cmd/ailang` package commands
design_docs/planned/20260918_non_agentic_ailang_non_agentic.md	ailang_fix	link from PROGRAM.md as AILANG fix lane
design_docs/implemented/v0_32_0/m-recorded-stream-api.md	core_floor_fix	Route this as a **core-floor addition** to `std/ai`, `internal/builtins`, and `internal/effects`, then re-freeze
design_docs/planned/m-pkg-multi-namespace-auth.md	other	**Lane:** registry infra (`cmd/registry-validator`) — not core, not an extension.
TSV
