# AILANG Known Limitations

> **Canonical page:** the maintained, live-verified limitations list is the published reference —
> **[docs/docs/reference/limitations.md](docs/docs/reference/limitations.md)** (rendered at
> <https://ailang.sunholo.com/docs/reference/limitations>). This root file is a pointer + short
> summary; the website copy is authoritative and carries per-entry repro transcripts + verified-at
> dates.

**All entries below were live-verified at AILANG `v0.33.1-72-g4b46bb97e` on 2026-08-17.** Per the
[M-V1-STABILITY-PROMISE](design_docs/planned/v1_0_0/m-v1-stability-promise.md) entry policy, every
open limitation is a reproducible artifact with a verified-at date, and fixed items move to a
dated "Resolved" list — this file is no longer allowed to freeze at a past version.

## Open limitations (summary)

See the [canonical page](docs/docs/reference/limitations.md) for repros, transcripts, and
workarounds. Verified open at v0.33.1 (2026-08-17):

| Limitation | Kind | Verified-at repro |
|---|---|---|
| **Y-combinator / recursive lambdas** | Design constraint (Hindley-Milner occurs-check) | `let Y = \f. (\x. f(x(x)))(\x. f(x(x)))` → `occurs check failed`. Use named `func` recursion. |
| **If-else multi-statement branches need braces** | Design constraint (no layout-sensitive parsing) | bare `let` in an `else` → "if-else branches require explicit braces". Wrap in `{ … }`. |
| **Duplicate record types with identical fields** | Go-codegen only (interpreter unaffected) | `--emit-go` may pick the first structurally-matching struct. `ailang run` returns the correct value. |
| **WASM type-checker depth limit** | WASM-host only (CLI unaffected) | deeply-recursive or pathologically-slow type structure trips a 2 s wall-clock budget; structured `budget exceeded` error. |
| **`?` error-propagation operator** | Not yet implemented (planned) | `r?` → `PAR_NO_PREFIX_PARSE: unexpected token in expression: ?`. Use explicit `match` on `Result`. |
| **Typed quasiquotes** | Not yet implemented (planned) | quasiquote syntax not accepted. Use `"${expr}"` interpolation / `concat([..])`. |
| **CSP concurrency (channels / session types)** | Deferred | no channel/session-type surface in the parser. |
| **Raw-mode single keypress / mid-call `std/ai.step()` abort** | Narrow input gaps | line input (`readLine`, `asyncReadStdinLines`) works; raw keypress is out-of-core by design. |
| **Regex backreferences / lookaround** | Design constraint (RE2 linear-time guarantee) | `std/regex` wraps Go's RE2 engine → **no backreferences, no lookahead/lookbehind**. `regex.compile("(a)\\1")` / `compile("(?=x)")` return `Err(message)` (never panic). This is the deliberate price of the linear-time, no-catastrophic-backtracking guarantee. |
| **Strict `take` after `flatMap` / allocating `map` bounds results, not peak memory** | Design constraint (strict evaluation) | Use `takeFlatMap(n, f, xs)` or, when `f` allocates, `takeMap(n, f, xs)`; [canonical entry](docs/docs/reference/limitations.md#strict-evaluation-taken-flatmapf-xs-bounds-the-result-not-the-peak) has the v0.33.0 repro and remaining bounds. |
| **Contract verification treats floats as real numbers** | Verifier model gap (runtime unaffected) | `ailang verify` / `ai-check` encode `float` as an SMT real, which has no NaN, so the solver can prove `x == x` for a value that is NaN at runtime. Runtime `==` is IEEE (NaN ≠ NaN, #1274). Don't rely on a verified float-equality contract when NaN is possible; guard with `std/math.isNaN`. Added 2026-09-25. |
| **Go codegen derived `Eq` on NaN-holding values** | Go-codegen only (interpreter and VM unaffected) | `--emit-go` derives Eq with `reflect.DeepEqual`, which treats a slice as equal to itself even when it holds NaN. `ailang run` gives the IEEE answer. Not exercised end-to-end. Added 2026-09-25. |
| **Quadratic `::` prepend / 10,000-frame evaluator recursion cap** | Runtime limitations (representation fix in progress) | Flat-slice `ListValue` makes every cons copy its tail; deep recursion returns `RT_REC_003`. Prefer builtin-backed `map` / `takeMap`, or `foldl` for non-list aggregation; `--max-recursion-depth N` raises only the depth wall, not the cons memory cost. See the [canonical entry](docs/docs/reference/limitations.md#prepending-with--is-quadratic-evaluator-recursion-is-capped-at-10000-frames). |

### If-Else Branches Require Explicit Braces {#if-else-branches-require-explicit-braces}

AILANG is not layout-sensitive, so a multi-statement `if`/`else` branch must be wrapped in braces.
Without them, only the first `let` is parsed as the branch and you get
"if-else branches require explicit braces when using let bindings":

```ailang
-- ❌ fails
if x > maxX then [] else
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest

-- ✅ works
if x > maxX then [] else {
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest
}
```

Single-expression branches don't need braces. See the
[canonical page](docs/docs/reference/limitations.md) for the full entry.

## Execution policy residuals {#execution-policy-residuals}

What `ailang run --policy` in **restricted** mode does and does not guarantee, after
M-EXECUTOR-POLICY-HARDENING (v0.41.0). The confined-execution claim covers the mediated effects
(`IO`, `FS`, `Net`, `Clock`, `Rand`, `Stream`) and the `ailang_only` lane's tools. Everything below
is outside it and is stated rather than papered over.

| Residual | Kind | What to do about it |
|---|---|---|
| **Hard links, device nodes, `/proc`, bind mounts seeded into the sandbox** | Root-relative APIs do not sever them (`os.Root` contract) | The launcher provisions a private per-task directory and never seeds it with these; a host that can change mount topology is outside the attacker model. |
| **CPU and memory** | No portable effect-level bound; Go's soft memory target is not a hard limit | Run the worker in a container/cgroup; `AILANG_EVAL_MAX_RSS` on the eval rig. A configured hard limit the platform cannot enforce fails startup rather than pretending. |
| **`trusted_host` mode** | An explicit host-integration grant with no confinement claim: full environment, proxy semantics, `Process`/`AI`/`Env`/`Secret` | Provenance only: `security_mode` is banked. Not accepted by the `ailang_only` lane. |
| **HTTP proxy in restricted mode** | Refused (`E_NET_PROXY_REFUSED`): a destination cannot be pinned behind a proxy | A constrained proxy protocol is future work; use `trusted_host` knowingly or unset the proxy. |
| **Loopback / private / metadata destinations in restricted mode** | Refused with no override — local Ollama development is outside restricted mode | Address-scoped grants are future work; `trusted_host` honours a loopback entry in `net_allow`. |
| **CLI ops of `ailang_cli` run as a separate unconfined process** | The path is validated inside the root at request time (and must not be a symlink); an in-root symlink swapped afterwards is a window the check cannot vouch for | Keep CLI grants short (`cli_allow`); the ops are read-only or write only the file they are given. |
| **The AI effect's HTTP client is not the Net authorizer** | `ai.call` reaches the pinned provider through `http.DefaultClient` (no pinned dial, no redirect re-check, proxy honoured); the host is the operator's, not the program's, and spend is capped by `[budgets] AI` | Pin `ai_provider`; state the budget; routing the client through the shared round tripper is the follow-on (ADC token fetches need the metadata exception). |
| **Persistence through the artifact** | Anything written inside the sandbox may be committed and later run with CI's or the next session's authority | `fs_deny_write` for the supply-chain paths; PR review; `artifact_patterns`. |
| **Confined git is a hardened subprocess, not an in-process reader** | `git status/diff/log` run the real `git` with a built argv, a stripped environment and the config knobs forced off; a git bug or a config key not covered by the hardening is outside the claim | `.git/` is read-only to the agent so the config is the launcher's; the schema admits a short flag list; an in-process `std/git` (go-git) would remove the subprocess and is the planned follow-on. |
| **Windows and `GOOS=js`** | No descendant termination claim / documented `os.Root` TOCTOU | Restricted mode refuses at startup with a named reason; `trusted_host` runs with those weaker guarantees. |
| **Allowed endpoints are not vetted** | Public-only denial does not make data sent to an allowlisted host safe | `net_allow` is the operator's judgement. |

## Resolved (were documented as broken; re-verified working at v0.33.1)

- **Polymorphic arithmetic lambdas** — `let add = \x. \y. x + y in add(3.14)(2.71)` → `5.85`
  (fixed v0.7.0, [m-poly-arithmetic-fix](design_docs/implemented/v0_7_0/m-poly-arithmetic-fix.md)).
- **`match` inside block-body lambdas in HOF arguments** —
  `map(\item. { let s = match item { 0 => "zero", _ => "ok" }; s }, [0,1,2])` → `[zero, ok, ok]`
  ([design doc archived](design_docs/archive/v0_13_0_m-dx-match-in-hof-block-lambda.md)). The old
  ML/Haskell `match … with | …` form is **retired** — it now emits `PAR019`; use brace-form
  `match x { pat => expr }`.
- **Multi-statement block expressions** — `{ e1; e2; e3 }` sequencing works fully; the old
  `let _ = … in` workaround is no longer needed.
- **Forgiving statement separators (v0.29+, M-SYNTAX-AI-FORGIVING)** — a `;`-separated
  sequence is now accepted directly in a `=`-body (`func f() = s1; s2; e`), and a
  **newline** is a soft statement separator inside `{ }` blocks (`{ let x = e\n rest }`).
  Both parse to the same AST as the `;`-in-braces form; a `;` is only *required* to
  separate two statements on the **same line**. Records still use commas.
- **String interpolation** — `"Value: ${x}"` → `Value: 42` (v0.12.1); `++` is now list-only
  (string `++` is a type error, v0.13.0).
- **Pattern guards** — `match x { n if n > 100 => …, n if n > 0 => …, _ => … }` (v0.6.2).
- **URL parsing** (v0.30.0, M-STDLIB-URL-PARSE) — `std/net` now parses URLs, not just builds them.
  `parseUrl(s) -> Result[Url, string]` splits a URL into `{scheme, host, port, path, query, fragment}`
  (RFC-3986, backed by Go `net/url`; `Err` on malformed input, never a panic), and
  `parseQuery(s) -> [{name, value}]` decodes a query string into order-preserving, percent-decoded
  pairs (the inverse of `urlEncodeForm`). Both **pure** — no `Net` capability. Previously an author
  had to hand-roll a `std/string.split` chain (order-sensitive, RFC-ignorant); that workaround is no
  longer needed.
- **Module-less file silent success** (v0.30.0, M-MODLESS-FAIL-LOUD) — a `.ail` file with top-level
  `export func` declarations but **no `module` line** used to type-check clean, print `✓ Running`, and
  **exit 0 with no output** (the entry never ran, since a module-less file exports nothing). `ailang
  run`/`check` now **fail loudly** with `MOD014: no 'module' declaration … Fix: add 'module
  <canonical/path>' as the first line`. The fix names the exact module line to add. Bare-expression
  files (`1 + 1` → `2`) are unaffected — they have no top-level funcs and still evaluate.

## Reporting New Limitations

File an issue at <https://github.com/sunholo-data/ailang/issues> with: AILANG version
(`ailang --version`), a minimal repro, expected vs actual behavior, and whether it's a bug or a
design constraint.
