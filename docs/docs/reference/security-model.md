---
sidebar_position: 5
title: Security Model
description: Capability-based authority, policy confinement, information-flow labels, contract verification bounds, and supply-chain provenance in AILANG.
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# AILANG Security Model

AILANG enforces **explicit authority** at the language and runtime boundary: side effects must appear in function types (`! {IO, FS, ...}`), and capabilities must be granted when running a program. However, **AILANG is not an OS sandbox** — capabilities like `FS`, `Process`, and `Env` grant broad host access unless narrowed by `AILANG_FS_SANDBOX`, `--policy` (Experimental), or OS-level isolation (containers, VMs, seccomp).

---

## Capabilities (`--caps`)

Every value accepted by `ailang run --caps` corresponds to a runtime capability. Capabilities marked **Stable** belong to Stable stdlib modules; **Experimental** capabilities belong to Experimental modules and may evolve in `1.x` minors (see [Stability Promise](/docs/reference/stability)).

| Capability | Tier | What it grants at runtime |
|---|---|---|
| `IO` | Stable | Console stdout/stderr/stdin (`print`, `println`, `readLine`, `readLineOpt`, `exit`) and `std/terminal` TTY control |
| `FS` | Stable | Read, write, mkdir, rename, walk, and delete across every host path the OS user can reach, unless narrowed by `AILANG_FS_SANDBOX` or `--policy` (`fs_sandbox` / `fs_deny_write`) |
| `Env` | Stable | Read every environment variable in the process snapshot (including credentials stored in env vars) and CLI arguments (`getArgs`), unless restricted via `--allow-env` / `--env-snapshot` |
| `Clock` | Stable | Monotonic/wall-clock timestamps (`now`) and `sleep` (supports deterministic virtual time with `--virtual-time`) |
| `Rand` | Stable | Random number generation (`randInt`, `randFloat`, `randBool`, `randSeed`; accepted by `--caps` and auto-grant, though omitted from `run --help`'s short `--caps` summary) |
| `Net` | Experimental | Outbound HTTP/HTTPS via `std/net` and `std/web`. Enforces HTTPS-only and blocks localhost, private IPs, and cloud metadata (`169.254.169.254`) by default unless widened with `--net-allow-http`, `--net-allow-localhost`, `--net-allow-metadata`, or `--net-allow-domains`; 30s timeout and 5 MB body cap by default |
| `Process` | Experimental | Spawns host commands (`std/process`). **Treat as full host access**: `Process` is not confined by `AILANG_FS_SANDBOX` alone (use `--process-allowlist` or `--policy` restricted mode) |
| `AI` | Experimental | Invokes configured LLM providers (`std/ai`), spending provider quota/credits (supports `--ai-stub` for offline testing) |
| `Stream` | Experimental | Opens bidirectional WebSocket and SSE connections (`std/stream`; WSS-only and private-IP blocking by default, 1 MB message cap) |
| `Secret` | Experimental | Resolves gated secrets (`std/secret`, e.g. `op://...`) into IFC-tainted values |
| `Trace` | Experimental | Emits OpenTelemetry-compatible custom spans and events (`std/trace`) |
| `SharedMem` | Experimental | Reads and writes shared key-value memory (`std/sharedmem`, `std/sem`) |
| `SharedIndex` | Experimental | Queries and upserts similarity-search indexes (`std/sharedindex`) |
| `DOM` | Experimental | Emits replayable DOM patches in browser/Cognitive-OS hosts (`std/dom`) |
| `Msg` | Experimental | Sends and receives Lamport-clocked fabric messages (`std/cognition`) |
| `Cog` | Experimental | Accesses Cognitive-OS event-log sinks (`std/cognition`) |

*(Note: `Debug` is a compile-time ghost effect with no runtime capability flag; `--release` erases it. `Declassify` is a compile-time IFC authority checked in effect rows and package `[effects].max` ceilings.)*

---

## Granting Authority (`--caps`, `--caps auto`, and `[bin]` Shims)

- **Explicit grant (`--caps IO,FS`):** Only listed capabilities are granted; calling any ungranted effect fails at runtime.
- **`--caps auto` and `[bin]` shims:** Passing `--caps auto` grants **every capability declared in the entrypoint's signature**. Installed package `[bin]` shims (`ailang pkg install`) default to `caps = "auto"` when `[bin].caps` is omitted in `ailang.toml`.
- **Rule for untrusted code:** Always inspect a module or package before running it with `--caps auto` or installing its `[bin]` shims — use `ailang inspect <file>`, `ailang iface <module>`, or `ailang pkg info <pkg>` to review declared effects and `[effects].max` ceilings.

---

## Confinement with `--policy` (Experimental)

For untrusted or agent-submitted programs, run with an operator-owned TOML policy file:

```bash
ailang run --policy /etc/ailang/agent-policy.toml untrusted.ail
```

- **Supervised execution:** The parent process resolves the policy, starts a child worker in its own process group, enforces `timeout_ms`, byte/output caps (`max_source_bytes`, `max_output_bytes`, `max_fs_transfer_bytes`), `allowed_caps`, `fs_sandbox`, `fs_deny_write` (read-only protected globs such as `.github/**`, `.git/**`), `net_allow`, and `[budgets]`.
- **Fail-closed flag refusal:** Widening CLI flags (`--caps`, `--entry`, `--env*`, `--net-*`, `--stream-*`, `--stdlib-path`, `--ai*`, `--fs-max-bytes`, etc.) are refused by name when `--policy` is set.
- **Modes:** `security_mode = "restricted"` (default) permits only effects with confined adapters; `trusted_host` is an explicit operator opt-in for host integrations with no confinement claim.
- See the [Agent Tool Policy guide](/docs/guides/agent-tool-policy) for the full policy schema.

### Package & Registry Confinement

When `AILANG_AGENT_POLICY` is set, all `ailang pkg` commands and the module loader refuse registry network fetches, `git clone`/`fetch`, and writes under `~/.ailang`. Operators can provision a read-only `AILANG_PACKAGE_ROOT` (`<vendor>/<name>/<version>`). Registry and git dependencies are verified against `ailang.lock`'s `content_hash` before compilation (content verification checks integrity against the lockfile, so protect the lockfile and package root from untrusted writes). See [SECURITY.md](https://github.com/sunholo-data/ailang/blob/dev/SECURITY.md#registry-and-package-confinement).

---

## Information-Flow Control (IFC) Labels (Experimental)

IFC labels (`T<label>`, sink refinements `T{not LABEL}`, and `! {Declassify}`) track explicit data flow in the type system at zero runtime cost and cross module boundaries (`#1134`).

**What IFC does NOT cover:**
- **Explicit flows only:** Labels do not track *implicit* control-flow channels (branching on a secret value to choose an unlabelled constant).
- **No runtime / storage persistence:** Labels erase at compile time and do not survive `FS`, `Net`, or `IO` round-trips.
- **Export annotation boundary:** Within one module, sink constraints are inferred for unannotated helper parameters (`#1719`); across module boundaries, an importer sees only what the exported signature explicitly annotates.

See the [IFC Labels guide](/docs/guides/ifc-labels) for examples and rules.

---

## Contracts & Z3 Static Verification

Functions may carry `requires { ... }` and `ensures { ... }` clauses checked statically by Z3 (`ailang verify` / `ailang check --verify`) or dynamically on the evaluator (`ailang run --verify-contracts`).

- **Bounded & model-scoped proofs:** Recursive functions and `map`/`filter`/`foldl` lambdas are unrolled to bounded depth unless bounded by `requires`; `int` is encoded as unbounded SMT `Int` (potential 64-bit runtime wraps surface as `int_overflow` caveats, or violations under `--int-model=int64`), and `float` inputs are assumed non-NaN.
- **Precondition trust:** `requires` is assumed when proving a function's own `ensures` and checked at calls made by *verified* callers; unverified callers and ordinary `ailang run` without `--verify-contracts` do not check `requires`.
- **Bytecode VM restriction:** `--verify-contracts` is evaluator-only and is refused alongside `--bytecode` / `--strict-bytecode` (`CTR004`, exit 2).
- **Exit codes:** `0` for `verified`, `no_obligations`, or `partial` (unless `--strict`); `1` for `unproved`, `inconclusive` (unless `--allow-inconclusive`), `counterexample`, `error`, or missing Z3 (`unavailable`); `2` for usage errors. See [Contracts & Verification](/docs/guides/contracts#exit-codes).

---

## Determinism Is Not Isolation

Program determinism — byte-identical outputs across the evaluator and bytecode VM for pure code and virtualized `Clock`/`Rand` — enables replay and auditability, **not** host isolation. A deterministic program granted `FS`, `Net`, `Env`, or `Process` still reads or modifies external host state unless confined by `--policy`, `AILANG_FS_SANDBOX`, or an OS sandbox.

---

## Supply Chain & Provenance

- **Release artifacts:** GitHub Releases (`v0.14.1`+) are keyless-signed with **Sigstore cosign** (`.sig` + `.pem` logged in Rekor) and carry **SLSA Build Level 3 provenance** (`multiple.intoto.jsonl`, `v0.15.x`+) alongside `SHA256SUMS`. See [SECURITY.md](https://github.com/sunholo-data/ailang/blob/dev/SECURITY.md#verifying-release-artifacts) for verification commands.
- **Dependencies:** `ailang.lock` (`ailang.lock/v1`) pins exact versions, `content_hash`, `interface_hash`, and declared effects, checked before compilation (`ailang lock --check`). Note that published registry packages are hash-verified via `ailang.lock`, **not** yet publisher-signed.

---

## Reporting Vulnerabilities

Please do **not** open public GitHub issues for security vulnerabilities. Report privately via **GitHub Security Advisories**:

- <https://github.com/sunholo-data/ailang/security/advisories/new>

See [SECURITY.md](https://github.com/sunholo-data/ailang/blob/dev/SECURITY.md) for response SLAs and scope.
