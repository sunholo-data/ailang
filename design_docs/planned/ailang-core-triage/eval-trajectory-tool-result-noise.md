# Eval tool-result noise: cloudtrace deprecation stderr + brain Read hook injection

- **Date**: 2026-11-06
- **Class**: bug (two distinct noise sources polluting eval/finetune tool results)
- **Recommend**: item 1 — duplicate-of `design_docs/planned/v0_50_0/m-otlp-exporter-migration.md`; item 2 — direct-fix
- **Searched**: `cloudtrace opentelemetry` in go.mod/internal, `M-OTLP` in design_docs/planned; `BRAIN Context for` repo-wide, `AILANG_EVAL_RUN` in extensions/hooks, `m-brain-context`, `ft_clean.py`
- **Estimate**: 1–2 lines in `~/.ailang/hooks/brain_context.sh` (item 2's guard; source-of-truth file for the hook is not in this repo — see caveat below)

Both items pollute every tool result in every pi agent eval (and 423 fine-tune
trajectories), so they belong on the same page, but they have different homes and
different verdicts.

**Item 1 — cloudtrace deprecation stderr: already ruled on.** The deprecated
`opentelemetry-operations-go/exporter/trace` (go.mod:15) logs via a `sync.Once`-guarded
`log.Println` in the library (`cloudtrace.go:40-46,198-199`) on first construction; the
repo's single call site is `cloudExporter` in `internal/platform/otel/sdk.go` (~:236),
which runs whenever `GOOGLE_CLOUD_PROJECT`/`OTLP_GOOGLE_CLOUD_PROJECT` resolves.
`design_docs/planned/v0_50_0/m-otlp-exporter-migration.md` (Status: Planned, P1) already
documents this exact mechanism, names the same call site, and rejects the
`log.SetOutput(io.Discard)` workaround (its R1) as racy — so the report's "or at least
silence the log" alternative is already considered and rejected there. New information
from this report worth folding into that doc's Problem Statement: the eval harness
injects `GOOGLE_CLOUD_PROJECT` into every agent child (`internal/executor/environment.go`
BuildEnvironment sets it alongside `OTLP_GOOGLE_CLOUD_PROJECT`), so the noise reaches
every bash result in every agent eval and 423 fine-tune trajectories — that raises the
doc's present-day-damage evidence but does not change the plan. Interim mitigation for
already-captured trajectories: `tools/finetune/ft_clean.py` (noise normaliser from the
pilot-2 tooling) could strip the 250-char prefix.

**Item 2 — brain Read hook injection into evals: new, small fix.** The reporter calls it
a "pi extension", but the `━━━ 🧠 BRAIN: Context for <dir>/ ━━━` header matches
`brain_context.sh`, the PreToolUse(Read) hook from M-BRAIN-CONTEXT
(`design_docs/implemented/v0_10_0/m-brain-context.md`, header echoed at doc line ~229,
installed at `~/.ailang/hooks/brain_context.sh`). It fires on every Read regardless of
eval mode. The suggested stand-down is exactly the pattern banked 2026-11-06 for the
session-protocol gate (`changelogs/unreleased/2026-11-06-eval-run-disarms-session-gate.md`;
`AILANG_EVAL_RUN=1` is injected by `BuildEnvironment` only for eval children and is
refused in `Task.ExtraEnv`, so it cannot be spoofed by the task). In the shell hook that
is a one-liner after the `TOOL_NAME` check: `[ "${AILANG_EVAL_RUN:-}" = "1" ] && exit 0`.
Caveat: the hook's source is not in this repo (not in `scripts/hooks/`, not in
`cmd/ailang/pi_assets/`, nothing matches `find / -name brain_context.sh` here) — the
installed copy lives on the operator machine, so the fix must be applied where the
hook's source of truth lives or it will be lost on reinstall.

Suggested routing: do item 2 as the immediate one-line fix; update
`m-otlp-exporter-migration.md`'s Problem Statement with the eval-harness injection
evidence when item 1's M0 spike starts.
