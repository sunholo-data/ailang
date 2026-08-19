# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 (history: charter STATUS + [log](motoko-mission-log.md)).
> **Namespaced** — the bare `mission-dashboard.md` is not ours (V1 iter-216); it holds motoko's stale
> iter-7 snapshot, left alone.

**As of**: 2026-08-19 · iteration **13** · release `v0.33.1` · loop `dev.ailang.mission-motoko`, 12h

## Now
- **Just landed (iter 13)**: queue item **6** UN-PARKED — Mark resolved `D-MOTOKO-FMT-1`
  *precondition* (attended, 2026-08-19), and the provider-resolution trace he made a precondition is
  RUN. **O4 CLOSED by measurement**; doc gains §12 + rows V25–V32.
- **The answer**: both halves of the reviewer's fear are true, of *different* lanes — so the fix is a
  CONDITION, never a deletion. fmt arms reach **no** OpenRouter path (`GuessProvider("ollama/…")`
  → `ollama`, env var `""`; both profiles pin `localhost:11434/v1`); OpenRouter lanes have the Go
  preflight as their **only** hard stop (motoko's own check merely warns and proceeds).
- **The blocker it found**: the condition is **not expressible where the check sits**.
  `HealthCheck(ctx)` takes no task, and `cfg.MotokoModel` is never set from `models.yml` — so an
  `if` at `healthcheck.go:64` would read the hardcoded OpenRouter default for every lane. D1 is a
  plumbing change, not a one-liner; three costed options in §12.2.

## Next
1. **Item 6 → normal sprint**: planner → executor on D1 (+ the §12.2 plumbing decision) + D1b + D2.
2. Item **7** — profile restoration design (untagged head behind it).
3. Item **8** — repin the stale OpenRouter motoko models.

## Gated / parked
- **Phase 0 CLOSED** (re-measured iter 13): G1 `#154` OPEN · G2 rc=128 (control rc=0) · G3 registry
  `latest=2.2.0`, no 5.x · G4 unrunnable · G5 (Arni's ABI declaration) unchanged. Rows **10/11/12**
  stay parked. Rows **9/13/14** need a green tree / earlier design.
- **Upstream is acting on `#165`**: `arniwesth/mot-100-fix-output-headroom` now **7** commits ahead of
  `main_dst` incl. `da999ac fix(compaction): reserve provider output headroom`, their PR **#166**.

## Loop health
- Routing: controller `claude:claude-opus-5`; designer rotation pointer `claude:claude-fable-5`
  (untouched). Executor chain `codex:gpt-5.6-sol` → `pi:deepseek(:floor)` → `opus`.
- Metered **$0.00** of $5 this iteration. No GPU, no `rig.lock`. `make quick-install` NOT run.
- dev **verified green**: 16 exact-SHA checks, 0 not-green, `runs_total=2`. dev CI is **V1's** to own.
## Parked on Mark
- **none** — decision ledger valid, **3** rows, **0 OPEN** (`scripts/mission_decisions.sh --open`).
- Bookkeeping issue **#743** (rotates Mondays 07:00 local); 0 directives since the watermark.
