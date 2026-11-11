---
reviewBy: 2027-03-08
---

# AILANG Known Limitations

> **Canonical page:** the maintained, live-verified limitations list is the published reference —
> **[docs/docs/reference/limitations.md](docs/docs/reference/limitations.md)** (rendered at
> <https://ailang.sunholo.com/docs/reference/limitations>). This root file is a pointer + short
> summary; the website copy is authoritative and carries per-entry repro transcripts + verified-at
> dates.

**All entries below were live-verified at AILANG `v0.33.1-72-g4b46bb97e` on 2026-08-17.** Per the
[M-V1-STABILITY-PROMISE](design_docs/implemented/v0_29_0/m-v1-stability-promise.md) entry policy, every
open limitation is a reproducible artifact with a verified-at date, and fixed items move to a
dated "Resolved" list — this file is no longer allowed to freeze at a past version.

## Open limitations (summary)

See the [canonical page](docs/docs/reference/limitations.md) for repros, transcripts, and
workarounds. Verified open at v0.33.1 (2026-08-17):

| Limitation | Kind | Verified-at repro |
|---|---|---|
| **Concrete function-effect annotation upper bounds** | Deferred Phase 3 | Verified 2026-10-08 on the v0.53.0 implementation: storing an `! {IO, Env}` function in an `! {IO}` field still checks; see the canonical page and existing #573/#1326. |
| **Y-combinator / recursive lambdas** | Design constraint (Hindley-Milner occurs-check) | `let Y = \f. (\x. f(x(x)))(\x. f(x(x)))` → `occurs check failed`. Use named `func` recursion. |
| **If-else multi-statement branches need braces** | Design constraint (no layout-sensitive parsing) | bare `let` in an `else` → "if-else branches require explicit braces". Wrap in `{ … }`. |
| **Duplicate record types with identical fields** | Go-codegen only (interpreter unaffected) | `--emit-go` may pick the first structurally-matching struct. `ailang run` returns the correct value. |
| **WASM type-checker depth limit** | WASM-host only (CLI unaffected) | deeply-recursive or pathologically-slow type structure trips a 2 s wall-clock budget; structured `budget exceeded` error. |
| **`?` error-propagation operator** | Not yet implemented (planned) | `r?` → `PAR_NO_PREFIX_PARSE: unexpected token in expression: ?`. Use explicit `match` on `Result`. |
| **Typed quasiquotes** | Not yet implemented (planned) | quasiquote syntax not accepted. Use `"${expr}"` interpolation / `concat([..])`. |
| **CSP concurrency (channels / session types)** | Deferred | no channel/session-type surface in the parser. |
| **Native terminal platform / mid-call `std/ai.step()` abort** | Platform and cancellation gaps | `std/terminal` supports scoped native input on macOS/Linux with host lifecycle integration; Windows/WASM return `Unsupported`. Interrupting an in-flight AI request remains separate. |
| **Regex backreferences / lookaround** | Design constraint (RE2 linear-time guarantee) | `std/regex` wraps Go's RE2 engine → **no backreferences, no lookahead/lookbehind**. `regex.compile("(a)\\1")` / `compile("(?=x)")` return `Err(message)` (never panic). This is the deliberate price of the linear-time, no-catastrophic-backtracking guarantee. |
| **Strict `take` after `flatMap` / allocating `map` bounds results, not peak memory** | Design constraint (strict evaluation) | Use `takeFlatMap(n, f, xs)` or, when `f` allocates, `takeMap(n, f, xs)`; [canonical entry](docs/docs/reference/limitations.md#strict-evaluation-taken-flatmapf-xs-bounds-the-result-not-the-peak) has the v0.33.0 repro and remaining bounds. |
| **JSON integers outside int64 are floats** | Design constraint (no big integers) | `std/json.decode` gives an exact `JInt(int)` for an integer literal that fits int64; a larger one (`9223372036854775808`) decodes as `JNumber` with the nearest float, so its low digits are rounded, and `getInt` returns `None` for it. `asNumber`/`getNumber` on a `JInt` above 2^53 also round (use `getInt` or match `JInt(n)`). Keep ids outside int64 as JSON strings. Through JSON text the numeric value round-trips, the constructor does not: `encode` prints a whole-number `JNumber` as `3`, so `decode(encode(jnum(3.0)))` is `Ok(JInt(3))`, not `Ok(JNumber(3.0))` (ruled 2026-11-10; compare numbers with `asNumber`/`getInt`). Replaces the old "all JSON numbers are floats, integers above 2^53 lose precision" behaviour (v0.58.x). Added 2026-11-10. |
| **A `requires` clause is an unchecked assumption by default** | Verifier and runtime scope | The verifier assumes a function's precondition while proving that function's `ensures` and never checks it where the function is called; `ailang run` checks it only with `--verify-contracts`. A `VERIFIED` result means "ensures holds whenever requires holds"; nothing stops a caller passing an argument that breaks the `requires`. Guard inputs in code where it matters. Added 2026-11-10. |
| **Runtime contracts: evaluator only, off by default, not in named tests** | Runtime coverage gap | `run --bytecode --verify-contracts` is refused (CTR004, exit 2; it used to run unchecked, #1956). `run` without `--verify-contracts` checks nothing, and a `test "…" { … }` body does not check the contracts of the functions it calls. See the canonical page. Added 2026-11-10. |
| **Contract verification treats floats as real numbers** | Verifier model gap (runtime unaffected) | `ailang verify` / `ai-check` encode `float` as an SMT real, which has no NaN, so the solver can prove `x == x` for a value that is NaN at runtime. Runtime `==` is IEEE (NaN ≠ NaN, #1274), and so are `<` `<=` `>` `>=` (false with a NaN operand, #1419). Don't rely on a verified float-equality or float-ordering contract when NaN is possible; guard with `std/math.isNaN`. Added 2026-09-25. |
| **Go codegen derived `Eq` on NaN-holding values** | Go-codegen only (interpreter and VM unaffected) | `--emit-go` derives Eq with `reflect.DeepEqual`, which treats a slice as equal to itself even when it holds NaN. `ailang run` gives the IEEE answer. Not exercised end-to-end. Added 2026-09-25. |
| **Go codegen user float arithmetic may fuse on arm64** | Go-codegen only (interpreter and VM unaffected) | `ailang run` (interpreter and `--strict-bytecode`) and every `std/math` function return identical bits on every CPU (#1465). A compiled program's own `x * y + z` is emitted as the Go expression `x*y + z`, which the Go compiler may fuse into one FMA on arm64, so `--emit-go` output can differ in the last bit from `ailang run` and between arm64 and amd64. Added 2026-10-02. |
| **Cross-ADT constructor tag collisions in unresolved bytecode switches** | Bytecode VM soundness gap (deterministic) | If static typing cannot identify the scrutinee and multiple ADTs declare every matched tag, the VM compiler chooses the first ADT in source declaration order. Output is deterministic, but programs should avoid relying on same-named tags across ambiguous ADTs until runtime tag identity is carried explicitly. Added 2026-09-29. |
| **Quadratic `::` prepend / 10,000-frame evaluator recursion cap** | Runtime limitations (representation fix in progress) | Flat-slice `ListValue` makes every cons copy its tail; deep recursion returns `RT_REC_003`. Prefer builtin-backed `map` / `takeMap`, or `foldl` for non-list aggregation; `--max-recursion-depth N` raises only the depth wall, not the cons memory cost. See the [canonical entry](docs/docs/reference/limitations.md#prepending-with--is-quadratic-evaluator-recursion-is-capped-at-10000-frames). |
| **`Map.insert` / `Set.insert` / `Array.set` copy the whole structure (O(n) per update)** | Runtime representation (persistent structures planned) | `std/set` and structured keys (tuples, records, ADTs) work since 2026-10-28, but every single-element update copies: 20,000 sequential `Set.insert`s of `(int, int)` take 5.6 s on the evaluator and 6.5 s on `--strict-bytecode` (×4 per doubling). Build with `fromList` / `updateMany` / `accumulate` (one copy) where the elements are known up front; a search loop that inserts once per visited state is O(n²) until the persistent Map/Set phase of [m-persistent-collections](design_docs/planned/v0_55_0/m-persistent-collections.md) lands. Added 2026-10-28. |
| **Module-qualified type names do not parse (`S.Set[int]`)** | Parser papercut (type syntax has no qualified names) | With `import std/set as S`, both `let e: S.Set[int] = S.empty()` and `func f(s: S.Set[int])` fail with `PAR_UNEXPECTED_TOKEN … got . instead`; calls such as `S.insert(...)` are fine. Import the type by name alongside the alias: `import std/set as S` plus `import std/set (Set)`, then write `Set[(int, int)]` (see `examples/runnable/visited_set_bfs.ail`). The same holds for any module-exported type. Added 2026-10-28. |
| **`--bytecode` is ~1.8× slower than the evaluator on copy-bound list building** | Runtime performance (representation width) | `foldl(\acc x. x :: acc, [], xs)` at 40,000 elements: 0.75 s evaluator, 1.35 s VM (bench/vm_hof_callbacks). Each `::` copies the whole list, and a VM value is 40 bytes against the evaluator's 16-byte interface, so the VM allocates 2.5× the bytes and runs more GC cycles. The profile is in [m-foldl-cons-cost-model](design_docs/implemented/v0_52_0/m-foldl-cons-cost-model.md#phase-3-profile-2026-10-02). Per-callback cost is not the cause: O(1)-step `foldl`/`map`/`filter` run 20–30× faster on the VM than on the evaluator. Build lists with `map`/`filter`/`mapAccumL`; the copy itself goes away only with the cons representation fix (D-19). Added 2026-10-02. |
| **Go codegen has no tail-call elimination** | Go-codegen only (interpreter and VM unaffected) | `--emit-go` output for a 10M-step tail-recursive loop dies with `fatal error: stack overflow`. `ailang run --strict-bytecode` runs it at constant depth. Values are boxed as `interface{}`. Added 2026-10-28. |
| **`--bytecode` falls back to the evaluator; VM ~2× slower than CPython** | Runtime coverage and performance | 18 of 205 evaluator-passing `examples/runnable` files fall back to the evaluator (61 before #1989, which made prelude `println`, local `letrec` and module-level value `let` native). The rest are unwired effectful builtins, contract-module ADTs, and user ADTs or closures passed to evaluator-only functions. Since 2026-11-09 the fallback is no longer silent: `--bytecode` prints a stderr warning naming the bridged functions, and `ailang check --strict-bytecode` reports each one as [BC001](docs/docs/reference/errors/bc001.md); `--strict-bytecode` on `run`/`test` fails before dispatch. `fib(32)`: VM 0.33 s (0.50 s before M-VM-QUICK-WINS), CPython 0.14 s. See [M-VM-COVERAGE](design_docs/planned/v0_55_0/m-vm-coverage-letrec-prelude-bridge.md). Added 2026-10-28. |
| **`--bytecode` recomputes a module-level value at each reference** | Runtime performance (no VM globals yet) | `let table = buildTable()` is evaluated once on the evaluator but once per reference on the VM, which calls the binding's initialiser wherever it is read. The answer is the same, because module-level bindings cannot carry effects, but an expensive initialiser read inside a hot loop is repeated. Bind it to a local in the function that loops, or pass it as a parameter. Once-only image globals are planned in [m-vm-coverage-letrec-prelude-bridge](design_docs/planned/v0_55_0/m-vm-coverage-letrec-prelude-bridge.md) (Phase 2b). Added 2026-10-28. |
| **VM list views retain backing storage** | Memory retention (shared slices since v0.54.10) | A small live tail/take/drop view can retain its whole source backing allocation. Capacity caps prevent append writes into the source but do not release storage. See the [canonical entry](docs/docs/reference/limitations.md#vm-list-views-retain-backing-storage). |

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

- **Effectful callbacks nested in list/tuple/ADT arguments** (fixed for v0.55.0, verified
  2026-10-26, [#1718](https://github.com/sunholo-data/ailang/issues/1718)) — a pure caller of
  `applyHead([logIt], x)`, `applyFst((logIt, x))` or `applyOpt(Some(logIt), x)` is now
  rejected with `Missing effects: IO`. Migration for the companion #1826 fix: a concrete
  `! {IO}` callback mapped with closed-pure `std/list.map` is a type error; use `mapE`.

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

## Same-named types across imports

Since M-V1-NOMINAL-TYPE-IDENTITY (M2) a nominal type is its **defining module plus its
name**: `q/a`'s `type T` and `q/b`'s `type T` are different types, and mixing them is
`TC_NOMINAL_001` at `check` (it used to type-check and fail at run time). A local `type T`
no longer captures an imported `T`, an imported alias body keeps the types it names, and a
module may declare its own `Map`, `List`, `Json`, … (its own wins over the built-in).
Using the two types disjointly (`Ma.desc(Ma.mk())`, `Mb.desc(Mb.mk())`) is fine.
A bare name never depends on import order: two explicit imports of one type or constructor
name are `MOD015`; a type name that two modules the program imports or reaches both declare is
`TC_AMBIGUOUS_TYPE_001` unless it is declared locally, imported by name, a built-in or a prelude
name; a constructor two imported modules offer is resolved in a pattern by the scrutinee's type
(once inference has decided it, whatever the order of the code), and is `TC_AMBIGUOUS_CTOR_001`
where nothing decides it and in an expression. A built-in name (`Map`, `List`, `Bytes`, …)
means the built-in unless the module declares or imports its own: a module that writes `Map`
meaning a dependency's `Map` must import it by name (`import dep (Map)`); the `TC_NOMINAL_001`
diagnostic says so. The implicit prelude (`Option`, `Result`, `Some`, `None`, `Ok`, `Err`) is
the lowest-precedence binding: a module's own declaration or import of any of them wins.
The REPL/WASM module registry follows the same rules, except that it has no import closure:
every loaded module is reachable, and a constructor offered by exactly one directly imported
module is used before the other loaded modules are considered.

Still open:

- An undeclared type name is still an opaque type of the module that writes it rather
  than `TC_UNBOUND_TYPE_001` (#1485); duplicate funcs/types/constructors in one module
  are not yet `MOD007` (#1278).
- Three back-end tables key by bare type name: the bytecode VM's ADT tables (two same-named
  ADTs with *different* constructors used in one program fail under `--strict-bytecode` with
  `BC001`; the evaluator runs them), the SMT encoder behind `check --verify`, and the Go
  emitter (`--emit-go`), which strips the module and names Go types by bare name.
- Records are structural, so a record type's `==` is decided by its declared name: if any
  imported module declares a record of the same name with `deriving (Eq)`, a same-shaped
  record of another module gets `==` too. The comparison is structurally correct, and a record
  with a field that has no `==` (a function) is still rejected.
- A single-constructor ADT wrapping a record (`type W = W({x: int})`) keeps its identity
  against every other nominal type, but still converts implicitly to and from its record
  (for field access), so one such type can reach another through a record-typed value, and a
  bare record is accepted where it is expected — a soundness hole (#2239, with #2233 and
  #2234; pre-existing).
- An uppercase name in a type-parameter list (`func id[T](x: T)`) is not a type variable: `T`
  is a type constructor (#2240). Inline function-literal annotations are not enforced (#2238).
