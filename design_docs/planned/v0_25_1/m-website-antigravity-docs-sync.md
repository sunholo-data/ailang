# M-WEBSITE-ANTIGRAVITY-DOCS-SYNC: Retire Gemini CLI from the Website, Document Managed Agents (Antigravity)

**Status**: Planned
**Target**: v0.25.1
**Priority**: P1 (the user-facing install path is actively broken — see Problem Statement)
**Estimated**: 0.5–1 day (docs-only, no Go code)
**Dependencies**:
- [m-antigravity-cli-migration.md](../../implemented/v0_22_0/m-antigravity-cli-migration.md) (the *code* migration this doc completes the *docs* for — already shipped in v0.22.0)
- No code dependencies. This is a documentation-only sprint.

---

## Executive Summary

The **code** retired the Gemini CLI executor in v0.22.0 (M-MANAGED-AGENTS) and replaced it with the Vertex AI **Managed Agents API** — the `antigravity-preview-05-2026` agent, in [internal/executor/managed_agents/](../../../internal/executor/managed_agents/). But the **website was never fully synced**: ~18 public docs/components still present "Gemini CLI" as a live, supported coding harness. One of them — the Getting Started bootstrap install — is now an instruction that **fails**, because Google stopped serving Gemini CLI requests for free/Pro/Ultra tiers on **2026-06-18** (five days before this doc was written).

This sprint is the docs-completion follow-on to M-MANAGED-AGENTS: sweep the website, replace stale Gemini CLI references with the correct successor **per context**, and do it without rewriting historical benchmark data or touching the unrelated `gemini_live` / `gemini_files` standard-library packages.

**Scope decision (set by user, 2026-06-23):** *Docs sync only. No new Go code.* "Antigravity" in this doc means the already-shipped Managed Agents executor (`antigravity-preview` agent), **not** the standalone Antigravity CLI (`agy`). Adopting `agy` as a new subprocess executor was considered and explicitly deferred (see Non-Goals).

---

## Problem Statement

**Current State:**
- ~18 website files reference "Gemini CLI" as a live coding harness/executor (full inventory in the Solution Design table).
- The codebase has **no** Gemini CLI executor — it was deleted in v0.22.0. The coordinator registers `claude, codex, opencode, motoko, pi, managed_agents` (verified: [internal/coordinator/provider_executor.go:10-15](../../../internal/coordinator/provider_executor.go#L10-L15)); there is no `gemini`.
- [docs/docs/guides/evaluation/harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md) is the **one** page already corrected (it has a Managed Agents section + retirement note) — so this sprint's job is the *other* pages.

**Why it's now P1, not cosmetic:**
- [Getting Started](../../../docs/docs/guides/getting-started.mdx#L25-L29) tells users to run `gemini extensions install https://github.com/sunholo-data/ailang_bootstrap.git`. As of **2026-06-18** Gemini CLI stopped serving requests for AI Pro/Ultra and free users ([Google Developers Blog](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/)). A first-run user following our docs hits a dead tool.
- The same dead-install instruction appears in [microrag.md](../../../docs/docs/guides/microrag.md#L267) and the homepage ([src/pages/index.jsx](../../../docs/src/pages/index.jsx#L405)).

**Impact:**
- **New users / AI agents** reading Getting Started are sent to a deprecated tool — first-impression failure.
- **Eval/benchmark readers** see "Gemini CLI" presented as a current harness, which no longer matches `models.yml` or the executor registry — undermines trust in the benchmark pages.
- **Cost-model docs are wrong by omission**: Gemini CLI was effectively free at point-of-use for many users; Managed Agents bills Vertex pricing ($1.50/$9.00 per 1M). Readers planning eval spend get a stale picture.

---

## Goals

**Primary Goal:** Make the public website truthfully reflect that Gemini CLI is retired and the Managed Agents API (Antigravity) is its successor — with the correct successor named *per context* — without rewriting historical benchmark data.

**Success Metrics:**
- `grep -ri "gemini cli" docs/` returns **only** intentional historical references (the design-docs.md changelog index) — verified by an allowlist.
- Zero install instructions on the website point at a tool that no longer serves requests.
- Benchmark dashboards still render historical `gemini`-harness rows with an accurate (historical) label, and any new `managed_agents` rows render with an Antigravity label.
- `make -C docs build` (Docusaurus) succeeds with no broken internal links.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **What replaces the Getting-Started `gemini extensions install` block** | It's a user-install step; Managed Agents is a *hosted API* with nothing to install, so there is no clean 1:1 substitute. | human | design | low |
| **How to treat historical `gemini` benchmark rows in the dashboards** | Relabeling them "Managed Agents" would falsely attribute past runs to a harness that didn't produce them (violates "report faithfully"). | human | design | low |
| **Whether coordinator docs name `managed_agents` as a live coordinator provider** | The executor is registered, but the coordinator's *provider/config* layer still has lingering `"gemini"` strings; documenting `managed_agents` as a coordinator provider must match real dispatch behavior. | human + verify | design | low |

### Recommended resolutions (defaults if no objection)

1. **Getting-Started block** → **Remove** the dead `### Gemini CLI` install block. Add one neutral line: *"Gemini CLI was retired by Google on 2026-06-18; an Antigravity CLI plugin for AILANG is tracked separately."* Keep Claude Code + Codex CLI as the live install paths. (Porting `ailang_bootstrap` to an Antigravity CLI plugin is real work in a *different repo* and is out of scope — see Non-Goals.)
2. **Historical dashboard rows** → **Additive labels.** Keep the `gemini` harness key labeled `"Gemini CLI (retired)"` for pre-v0.22 data; **add** a `managed_agents: "Managed Agents (Antigravity)"` label for new data. Never rewrite a past run's harness attribution.
3. **Coordinator docs** → Replace `gemini` provider references with `managed_agents` **only after** a 5-minute verification that the coordinator can actually dispatch to it as a provider. If it cannot (provider layer still CLI-only), document **claude** as the live provider and mark gemini *retired* rather than inventing a managed_agents provider path.

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Getting-Started: **resolved** — removed the dead block, added a retirement `:::note`, kept Claude Code + Codex; homepage quickstart card swapped Gemini CLI → Codex CLI.
- [x] Dashboards: **resolved** — additive labels (`gemini: "Gemini CLI (retired)"` kept for historical rows; `managed_agents` added). No historical relabeling.
- [x] Coordinator: **resolved** — Phase 0 verification showed the coordinator provider layer is still claude/gemini-only (no `managed_agents` provider dispatch). Documented **claude** as the live provider and marked the gemini provider retired; did **not** invent a managed_agents coordinator provider.

> **Implementation status (2026-06-23):** Implemented on `dev` (docs-only, no code).
> 18 files edited (+ `docs/internal/EXECUTOR_SHAPE.md`); Docusaurus build passes;
> allowlist grep clean (only design-doc history titles + intentional "(retired)"
> notes remain). Move this doc to `implemented/v0_25_1/` at release.

## Solution Design

### Overview

A context-aware find-and-replace across the docs tree, plus three surgical edits (dashboards, telemetry, getting-started) where naïve replacement would be wrong. Three buckets:

- **Bucket A — User-facing "use your agent" pages.** Successor is *not* Managed Agents (you can't install a bootstrap into a hosted API). Remove the dead Gemini CLI install path; keep live agents.
- **Bucket B — Eval / benchmark / coordinator / telemetry pages.** Successor *is* the Managed Agents API (`antigravity-preview` agent). Replace prose, tables, and config snippets.
- **Bucket C — Deliberately untouched.** Unrelated products (`gemini_live`, `gemini_files`), and historical design-doc index titles.

### Architecture

This is documentation; there are no software components. The "architecture" is the **replacement policy per reference kind**, captured in the inventory below.

### File Inventory & Remediation

Reference kinds and counts come from a full sweep of `docs/`. **Verify** each file's current line numbers at edit time (they drift).

#### Bucket A — User-facing install / "plug in your agent" (remove dead path)

| File | What's there now | Action |
|------|------------------|--------|
| [docs/docs/guides/getting-started.mdx](../../../docs/docs/guides/getting-started.mdx#L25-L29) | `### Gemini CLI` + `gemini extensions install ...` | Remove block; add retirement note (Decision #1). Update the "Add to agent" buttons if they list Gemini. |
| [docs/docs/guides/microrag.md](../../../docs/docs/guides/microrag.md#L191) | "installed via the Claude Code plugin or Gemini CLI extension"; `gemini extensions install ...` | Drop the Gemini CLI extension mention / install line; keep Claude Code plugin path. |
| [docs/docs/guides/quick-start-examples.mdx](../../../docs/docs/guides/quick-start-examples.mdx#L46) | "With Claude Code or Gemini CLI:" | "With Claude Code or another supported agent:" (drop Gemini CLI). |
| [docs/src/pages/index.jsx](../../../docs/src/pages/index.jsx#L405) | `<span>Gemini CLI</span>` quickstart chip; L1232 "Plug AILANG into Claude Code, Gemini CLI, or any agent…" | Remove/replace the Gemini CLI chip; reword the prose to current live agents. |

#### Bucket B — Eval / benchmark / coordinator / telemetry (→ Managed Agents / Antigravity)

| File | What's there now | Action |
|------|------------------|--------|
| [docs/docs/guides/evaluation/model-configuration.md](../../../docs/docs/guides/evaluation/model-configuration.md#L179) | "harnesses — Claude Code, Gemini CLI, Codex, opencode"; `gemini \| npm i -g @google/gemini-cli`; `# gemini-3-flash → gemini executor` | Replace harness list + table row with `managed_agents` (ADC auth, no CLI); fix the executor-mapping comment. |
| [docs/docs/guides/evaluation/architecture.md](../../../docs/docs/guides/evaluation/architecture.md#L70) | `--agent` help "(agentic coding via Claude/Gemini CLI)"; example "Claude Code / Gemini CLI" | "Claude Code / Managed Agents / …". |
| [docs/docs/guides/evaluation/cost-and-speed-budgets.md](../../../docs/docs/guides/evaluation/cost-and-speed-budgets.md#L20) | executor list "claude/opencode/gemini/codex/pi"; "no incremental usage from Gemini CLI" | Replace `gemini` with `managed_agents`; note **post-hoc-only** budgeting (Managed Agents reports usage only at `interaction.completed`). |
| [docs/docs/benchmarks/explorer.md](../../../docs/docs/benchmarks/explorer.md#L4) | "Claude CLI, Gemini CLI, opencode, Codex" (×2) | Update harness list; note Gemini CLI rows are historical. |
| [docs/docs/benchmarks/performance.md](../../../docs/docs/benchmarks/performance.md#L29) | "Agentic CLI (Claude Code / Gemini CLI / opencode / Codex)" | Update list to include Managed Agents; annotate Gemini CLI as historical. |
| [docs/docs/guides/coordinator.md](../../../docs/docs/guides/coordinator.md) | L3, L42 `default_provider: claude # "claude" or "gemini"`, L887/948/975/985/1376 diagrams & tables naming Gemini CLI | See Decision #3 — replace with `managed_agents` **iff verified**, else document claude-only + gemini retired. |
| [docs/docs/guides/coordinator-setup.md](../../../docs/docs/guides/coordinator-setup.md#L7) | "AI agents (Claude Code or Gemini CLI)" | Same policy as coordinator.md. |
| [docs/docs/guides/collaboration-hub.md](../../../docs/docs/guides/collaboration-hub.md#L144) | Mermaid `Claude Code /<br/>Gemini CLI` | Update diagram label. |
| [docs/docs/guides/database-architecture.md](../../../docs/docs/guides/database-architecture.md#L48) | session-ID table "Claude Code/Gemini CLI" | Update label. |
| [docs/docs/guides/telemetry.md](../../../docs/docs/guides/telemetry.md) | **Surgical** — see note below | Remove dead Gemini-CLI-executor telemetry; keep `gemini.generate` (standard-mode AI provider still exists). |

**Telemetry surgery (telemetry.md).** Two different "gemini" spans live here:
- `gemini.generate` (AI provider, standard-mode text gen via [internal/ai/gemini](../../../internal/ai/gemini/)) — **still exists, keep it.**
- `gemini.execute` (the deleted executor) + the entire "Gemini CLI native OTEL" config section (env var `GEMINI_TELEMETRY_ENABLED`, the `geminicli.com` link, the CLI→ailang span hierarchy) — **dead, remove or replace with the Managed Agents span shape.** Managed Agents emits spans from the in-process executor (HTTP/SSE), not from an external CLI's OTEL exporter — so the "two CLIs both export traces" framing no longer applies to the Gemini path.

#### Dashboard React components — additive labels only (Decision #2)

| File | What's there now | Action |
|------|------------------|--------|
| [docs/src/components/BenchmarkExplorer/index.jsx](../../../docs/src/components/BenchmarkExplorer/index.jsx#L7) | `gemini: 'Gemini CLI'` in `HARNESS_LABEL` | Keep key, relabel `'Gemini CLI (retired)'`; **add** `managed_agents: 'Managed Agents (Antigravity)'`. |
| [docs/src/components/OSVersionTrend/index.jsx](../../../docs/src/components/OSVersionTrend/index.jsx#L12) | same `HARNESS_LABEL` | same |
| [docs/src/components/BenchmarkDashboard/BenchmarkGallery.jsx](../../../docs/src/components/BenchmarkDashboard/BenchmarkGallery.jsx#L7) | same `HARNESS_LABEL` | same |
| [docs/src/components/BenchmarkDashboard/AgentRadar.jsx](../../../docs/src/components/BenchmarkDashboard/AgentRadar.jsx#L161) | prose "Claude Code, Gemini CLI" | update prose to current harness set |

> **Why additive, not rename:** historical benchmark rows with `harness: "gemini"` were genuinely produced by Gemini CLI before v0.22.0. Relabeling them "Managed Agents" would mis-attribute real measurements. New runs use `harness: "managed_agents"`. Both labels coexist.

#### Bucket C — Deliberately untouched (record so nobody "fixes" them)

| File | Why leave it |
|------|--------------|
| [docs/docs/packages/sunholo/gemini_live.mdx](../../../docs/docs/packages/sunholo/gemini_live.mdx) | Gemini **Live API** (WebSocket) stdlib package — unrelated to the CLI. |
| [docs/docs/packages/sunholo/gemini_files.mdx](../../../docs/docs/packages/sunholo/gemini_files.mdx) | Gemini **Files API** stdlib package — unrelated to the CLI. |
| [docs/docs/packages/sunholo/index.mdx](../../../docs/docs/packages/sunholo/index.mdx) | References the above packages. |
| [docs/docs/reference/effects.md](../../../docs/docs/reference/effects.md) | Gemini Live **demo** reference, not the CLI executor. |
| [docs/docs/design-docs.md](../../../docs/docs/design-docs.md) | Links to historical design docs whose **titles** contain "Gemini CLI" (e.g. "Retire Gemini CLI…"). Accurate history — leave verbatim. |
| [docs/docs/guides/evaluation/harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md) | **Already synced** in v0.22.0 (M4). Verify-only; optionally tighten the now-past-tense "deprecates on 2026-06-18" wording. |

### Standard-mode Gemini is NOT retired (clarity for editors)

Direct Vertex `generateContent` for Gemini models ([internal/ai/gemini](../../../internal/ai/gemini/)) is unchanged. Gemini models still appear in **standard-mode** eval results. Only **agent-mode** moved from Gemini CLI → Managed Agents. Any doc edit that implies "Gemini is gone from AILANG" is wrong — the *CLI harness* is gone; the *model* is not.

### Implementation Plan

**Phase 0: Verify (~20 min)**
- [ ] Confirm coordinator can dispatch to `managed_agents` as a provider (read [provider_executor.go](../../../internal/coordinator/provider_executor.go) + [cloud_dispatcher.go](../../../internal/coordinator/cloud_dispatcher.go)). Result drives Decision #3.
- [ ] `grep -rin "gemini cli\|gemini-cli\|gemini extensions install\|@google/gemini-cli" docs/` → confirm the live inventory against the table above; note any drift.
- [ ] Confirm the dashboard data source still emits `harness: "gemini"` rows (historical) and whether any `managed_agents` rows exist yet.

**Phase 1: Bucket A — user-facing (~1.5 h)**
- [ ] getting-started.mdx: remove dead block + retirement note + fix AddToAgent buttons
- [ ] microrag.md, quick-start-examples.mdx, index.jsx

**Phase 2: Bucket B — eval/coordinator/telemetry (~2.5 h)**
- [ ] model-configuration.md, architecture.md, cost-and-speed-budgets.md
- [ ] benchmarks/explorer.md, performance.md
- [ ] coordinator.md, coordinator-setup.md, collaboration-hub.md, database-architecture.md
- [ ] telemetry.md (surgical — keep `gemini.generate`, remove `gemini.execute` + CLI OTEL section)

**Phase 3: Dashboards + verify (~1 h)**
- [ ] Additive `HARNESS_LABEL` in the 3 components; prose in AgentRadar
- [ ] `make -C docs build`; check no broken links
- [ ] Final allowlist grep (success metric #1)
- [ ] harness-setup.md: tighten past-tense wording (optional)

### Files to Modify/Create

**New files:** none.

**Modified files (docs only, ~18):** see the File Inventory tables above. No `.go`, no `models.yml`, no executor code.

## Examples

### Example 1: Getting Started install block

**Before:**
```markdown
### Gemini CLI

​```bash
gemini extensions install https://github.com/sunholo-data/ailang_bootstrap.git
​```
```

**After:**
```markdown
> **Gemini CLI was retired by Google on 2026-06-18.** Its successor, the
> Antigravity CLI (`agy`), does not yet have an AILANG plugin (tracked
> separately). Use Claude Code or Codex CLI below.
```

### Example 2: Benchmark dashboard label (data integrity)

**Before:**
```jsx
const HARNESS_LABEL = { claude: 'Claude CLI', gemini: 'Gemini CLI', codex: 'Codex', ... };
```

**After:**
```jsx
const HARNESS_LABEL = {
  claude: 'Claude CLI',
  gemini: 'Gemini CLI (retired)',          // historical rows — do not relabel
  managed_agents: 'Managed Agents (Antigravity)',
  codex: 'Codex', ...
};
```

## Success Criteria

- [ ] `grep -ri "gemini cli" docs/` returns only the design-docs.md history index (allowlisted)
- [ ] No website install instruction points at Gemini CLI
- [ ] Dashboards render historical `gemini` rows with a "(retired)" label; `managed_agents` label exists for new rows
- [ ] telemetry.md keeps `gemini.generate`, drops `gemini.execute` + the Gemini CLI OTEL section
- [ ] Bucket C files unchanged
- [ ] `make -C docs build` passes; no broken internal links
- [ ] CHANGELOG.md updated (docs section)

## Testing Strategy

**Manual / mechanical:**
- Allowlist grep (success metric #1) before/after.
- `make -C docs build` (Docusaurus link + MDX check).
- Visual spot-check of the benchmark dashboard with historical data to confirm labels read correctly.

**No unit/integration tests** — this sprint touches no Go code.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of the Getting-Started retirement note — agent may choose, within Decision #1.
- Whether to also add a short "Managed Agents (Antigravity)" subsection to the eval *overview* page for discoverability — agent may add if low-cost.
- Whether to tighten harness-setup.md's past-tense deprecation wording now — agent may choose.

## Non-Goals

- **Adding the Antigravity CLI (`agy`) as a subprocess executor.** Now technically viable (it gained `--model`, headless `-p`, ADC since the v0.22.0 rejection) but has a known [non-TTY stdout-drop bug](https://antigravitylab.net/en/articles/integrations/antigravity-cli-agy-headless-non-tty-stdout-ci) that's hazardous for an eval harness. Out of scope — would be a separate code sprint reopening [m-antigravity-cli-migration.md](../../implemented/v0_22_0/m-antigravity-cli-migration.md)'s rejection.
- **Porting `ailang_bootstrap` to an Antigravity CLI plugin.** Different repo ([sunholo-data/ailang_bootstrap](https://github.com/sunholo-data/ailang_bootstrap)), depends on Google publishing the Antigravity plugin install command (not yet documented).
- **Any Go code changes** — executors, `models.yml`, coordinator provider layer. The lingering `"gemini"` provider strings in [internal/coordinator/](../../../internal/coordinator/) are noted but not fixed here.
- **Rewriting historical benchmark data** — explicitly forbidden (Decision #2).
- **Touching `gemini_live` / `gemini_files` packages** — unrelated products.

## Timeline

Single ~0.5–1 day session: Phase 0 (verify) → Phase 1–3 (edits) → build + grep gate → CHANGELOG. No multi-week rollout.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Editor relabels historical `gemini` rows as Managed Agents, mis-attributing real measurements | High (data integrity) | Decision #2 + additive-label example + success criterion; called out in three places in this doc |
| `gemini.generate` (live, standard-mode) deleted alongside `gemini.execute` (dead) | Med | Explicit "telemetry surgery" note distinguishing the two spans |
| Coordinator docs claim a `managed_agents` provider path that doesn't actually dispatch | Med | Phase 0 verification gates Decision #3 |
| Line numbers in this doc drift before edit time | Low | Table says "verify at edit time"; Phase 0 re-greps the inventory |
| Bucket C files get "helpfully" edited | Low | Bucket C table records *why* each is left alone |
| Getting-Started left with no Gemini path and no replacement reads as a gap | Low | Neutral retirement note + two live agents (Claude Code, Codex) remain |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a documentation-accuracy change. It introduces no language semantics, syntax, effects, or runtime behavior, so most axioms are N/A (scored 0). It is scored honestly rather than padded.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime change |
| A2: Replayability | 0 | No trace change (telemetry edit only removes docs for a deleted span) |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | 0 | No type-system change |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Accurate docs are what AI agents read first; a dead install instruction wastes agent turns/tokens. Fixing it directly serves the machine-first reader. |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Corrects the cost story: surfaces Managed Agents' Vertex pricing + post-hoc-only budgeting where docs previously implied a free Gemini CLI path |
| A10: Composability | 0 | No change |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | 0 | No change |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): change *improves* machine-first accuracy

### Decision Thresholds

Net +2 → ✅ Proceed to implementation. No −1 on A1/A3/A4/A7.

## Related Documents

**Implemented (parent / informs design):**
- [m-antigravity-cli-migration.md](../../implemented/v0_22_0/m-antigravity-cli-migration.md) — the **code** migration this doc completes the docs for. Its M4 (documentation) updated harness-setup.md + the coordinator *rule*, but missed the ~18 public website files this doc sweeps. **This doc is the unfinished half of that doc's M4.** (Did not surface in neural search; linked manually.)
- [m-managed-agents-sprint-plan.md](../../implemented/v0_22_0/m-managed-agents-sprint-plan.md) — sprint plan for the code migration.
- [m-docs-accuracy-website-update.md](../../implemented/v0_6_0/m-docs-accuracy-website-update.md) (0.40) — prior precedent for a website-accuracy sweep.
- [m-pkg-explorer-website.md](../../implemented/v0_10_0/m-pkg-explorer-website.md) (0.42) — prior website/component work.

**Planned (checked, not overlapping):**
- [m-cascade-observability.md](../v0_15_0/m-cascade-observability.md) (0.36) — observability, not docs sync. No overlap.

## References

- [Design Axioms](/docs/references/axioms)
- [Transitioning Gemini CLI to Antigravity CLI — Google Developers Blog](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/) (deprecation date 2026-06-18; extensions become "Antigravity plugins", no 1:1 parity yet)
- [Build with Google Antigravity — Google Developers Blog](https://developers.googleblog.com/build-with-google-antigravity-our-new-agentic-development-platform/)
- [Antigravity CLI overview — Google docs](https://antigravity.google/docs/cli-overview)
- [Antigravity CLI (agy) headless / non-TTY stdout bug](https://antigravitylab.net/en/articles/integrations/antigravity-cli-agy-headless-non-tty-stdout-ci) — why `agy` is not yet harness-safe
- [internal/executor/managed_agents/](../../../internal/executor/managed_agents/) — the shipped executor "Antigravity" refers to in this doc
- [harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md) — the reference template for the Managed Agents prose

## Future Work

- Reopen the Antigravity CLI (`agy`) executor decision once the non-TTY stdout bug is fixed upstream — would restore a *local* Gemini-family agent harness alongside the hosted Managed Agents path.
- Port `ailang_bootstrap` to an Antigravity CLI plugin once Google publishes the install command, restoring the Getting-Started one-liner for `agy` users.
- Clean up the lingering `"gemini"` provider strings in [internal/coordinator/](../../../internal/coordinator/) (code, separate sprint).

---

**Document created**: 2026-06-23
**Last updated**: 2026-06-23

DESIGN_DOC_PATH: design_docs/planned/v0_25_1/m-website-antigravity-docs-sync.md
