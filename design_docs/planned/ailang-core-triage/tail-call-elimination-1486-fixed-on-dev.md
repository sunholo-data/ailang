# Re: #1486 tail-call elimination in the interpreter — fixed on dev

- **Date**: 2026-10-02
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/implemented/v0_51_1/m-eval-tail-calls.md
- **Searched**: `tail call`, `tail-call`, `1486` across `design_docs/` (grep; rg unavailable in this env); verified repo state via `git log` and `ls cmd/ailang/`
- **Estimate**: n/a (no change required)

The message is a completion notice from `stapledons_godot` confirming #1486 is fixed on
ailang dev (3c77de8e4, not yet released), not a new report. It is already fully covered by
`design_docs/implemented/v0_51_1/m-eval-tail-calls.md`, which is marked **IMPLEMENTED**,
targets v0.51.1, and lists #1486 as its primary issue (line 8). Verified against the repo:
HEAD is `1b39aa7d` "docs(m-eval-tail-calls): sprint complete, design doc to implemented/v0_51_1";
the doc's planned artifacts exist — `cmd/ailang/tail_call_parity_test.go` (the #1486 repro,
interpreter vs strict VM) and `cmd/ailang/tail_call_trace_test.go`, with the tail-call frame
reuse in `internal/eval/eval_apply.go` (`evalCoreApp` apply path).

The message's technical claims match the design doc point for point: tail position handling
for if branches / let-letrec bodies / match arms / block ends, frame reuse including mutual
recursion, retained nested frames for active `ensures`/`@limit`/`@min`/rand-mode calls,
byte-identical enter/exit trace events (doc's D6 / A2), the RT_REC_003 → runs-until-killed
behaviour change, and the 364 MB → 54 MB memory drop (doc's A9 / V6). The interpreter-vs-VM
parity-without-depth-flag outcome is the doc's A1 justification.

No action needed beyond awareness that the fix is unreleased — the release lane
(`release-manager`) will pick it up with the v0.51.1 cut. If the reporter wants follow-up on
the related #1487 (`ailang test --bytecode`) or #1317 (unsafe depth ceilings), those are
already cross-referenced in the same doc's Issues line.
