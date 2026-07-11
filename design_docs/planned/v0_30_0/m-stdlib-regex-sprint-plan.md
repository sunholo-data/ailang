# Sprint Plan — m-stdlib-regex (Linear-Time RE2 Regex Builtin)

**Design doc**: [m-stdlib-regex.md](m-stdlib-regex.md)
**Sprint ID**: M-STDLIB-REGEX
**Target**: v0.30.0 · v1.0 bar clause 4 (orchestration flagship, R7)
**Estimated**: 2 days (~3 milestones) · Risk: **low–medium** (purely additive; one real correctness gotcha)
**Planner model**: claude-opus-4-8
**Premises verified live**: yes — read-only against HEAD `v0.29.2-35-gb62ab5433` (binary rebuilt, `--version` == `git describe`)

---

## Goal

Ship `std/regex` backed by Go's `regexp` (= RE2, linear-time by construction): `compile → Result[Regex,string]`
plus total match functions (`isMatch`, `findFirst`, `findAll`, `replaceAll`, `split`) with capture groups.
Closes the regex half of v1.0 bar clause 4. **Purely additive** — new builtin file, new `_regex_*` names,
new `std/regex` module. No grammar, no AST, no existing behavior changes.

## Reality-check findings (live against HEAD — de-risk the executor)

These correct/confirm the design doc; the executor should not re-derive them:

| # | Finding | Impact on plan |
|---|---------|----------------|
| **F1 — BYTE vs RUNE (the real gotcha)** | `_str_slice`/`_str_len` are **rune-indexed (UTF-8 aware)** (`internal/eval/builtins_string.go:77,88`). Go `regexp` `FindStringSubmatchIndex` returns **byte offsets**. | `RegexMatch.start/end/groups[].start/end` **MUST be converted byte→rune** so spans are consistent with `std/string` indexing. This is M1's headline correctness requirement + a multibyte test fixture. `.text` is sliced from the byte offsets (correct regardless); only the exposed integer indices need conversion. |
| **F2 — embed is a glob** | `std/embed.go` is `//go:embed *.ail` (`std/embed.go:10`). | `std/regex.ail` **auto-embeds by existing** in `std/`. The doc's "+1 LOC to embed.go if the list is explicit" is **not needed** — do not touch embed.go. |
| **F3 — registration pattern** | Builtins register via `registerXBuiltins()` calls in `internal/eval/builtins.go` init() (`registerStringBuiltins()`, `registerJSONBuiltins()`, …). | Add `registerRegexBuiltins()` to that init() list; mark each builtin `IsPure: true`. |
| **F4 — CHANGELOG location** | Root `CHANGELOG.md` **and** `changelogs/v0.18-current.md` both exist; the current active changelog is `changelogs/v0.18-current.md` (prior sprint convention, its D2). | Add the `[Unreleased]` entry to `changelogs/v0.18-current.md`; only touch root `CHANGELOG.md` if it proves to be the canonical aggregate (executor confirms). |
| **F5 — stability tier** | `docs/docs/reference/stability.md` has an **Experimental** tier with a listed surface (`std/net`, effect-refinement, …). | Add a `std/regex → Experimental` row/mention at introduction (the cheap-to-be-wrong direction). |

Verified absent (nothing to un-break): `grep -rn "_regex_\|std/regex" internal/ std/` → 0 matches; `internal/eval/builtins_regex.go` does not exist.

---

## Milestones

### M1 — Go RE2 builtins + compile cache + unit tests (~1 day, ~400 LOC)

Create `internal/eval/builtins_regex.go`: the six `_regex_*` builtins delegating to a memoized `*regexp.Regexp`,
marshalling results with the **same helpers** `builtins_json.go` (Result + record) and `builtins_string.go`
(`_str_split` list) already use — do not invent marshalling.

- `_regex_compile(pattern) -> Result[Regex,string]`: `regexp.Compile`; cache the compiled ptr **or** the compile
  error keyed by pattern string under a `sync.Mutex`; return `Ok(())`/`Err(msg)` in the AILANG Result shape.
  Never panic on a bad pattern (CP2).
- `_regex_is_match`, `_regex_find_first`, `_regex_find_all`: use `FindStringSubmatchIndex` /
  `FindAllStringSubmatchIndex`; **convert byte offsets → rune indices (F1)** for the exposed `start`/`end`;
  slice `.text` from byte offsets; non-participating optional groups report `start = -1`.
- `_regex_replace_all`: `ReplaceAllString` (Go `$1`/`${name}` repl syntax — document the subset). `_regex_split`: `Split(s, -1)`.
- Register via `registerRegexBuiltins()` in `builtins.go` init() (F3), all `IsPure: true`.
- Go unit tests (`builtins_regex_test.go`): compile ok + err-no-panic for `"("` / `"(?=x)"` (lookaround) / `"(a)\\1"` (backref);
  isMatch true/false; findFirst spans+groups; findAll multiple; replaceAll `$1`; split on `\s+`; **linear-time bound**
  (`(a+)+$` vs `strings.Repeat("a",40)+"!"` under a hard wall bound, e.g. 100ms); **multibyte rune-index fixture** (F1);
  compile-twice-consistent (cache memoization).

**Acceptance**: `go test ./internal/eval/... -run Regex -count=1` green; linear-time bound test passes; byte→rune conversion proven by a multibyte fixture; no panic on any bad pattern; ≥80% coverage on `builtins_regex.go`; `make lint` clean on the new file.

### M2 — std/regex.ail module + examples (~0.5 day, ~170 LOC)

Create `std/regex.ail` (auto-embedded, F2): `export type Regex = Regex(string)`, `export type RegexMatch = {...}`,
and the six `pure func` wrappers exactly as the design doc's Public API (already `ailang check`-verified).

- `examples/regex_basics.ail` (isMatch/find/split), `examples/regex_capture.ail` (group extraction), and one
  **orchestration-flavored** example (validate + extract fields from a log/LLM line — feeds clause-4 flagship #19).
- Register examples on the website via raw-loader per coding-standards (never inline).
- `make verify-examples` green over the full corpus (no pre-existing example regresses).

**Acceptance**: `ailang check std/regex.ail` clean; `ailang run examples/regex_capture.ail` prints extracted groups; all three examples run; `make verify-examples` green.

### M3 — integration tests, docs, polish (~0.5 day, ~180 LOC)

- Stdlib-level test (`.ail` golden or Go harness) over the public API incl. cross-module use with `std/string`+`std/result`.
- Docs: `docs/LIMITATIONS.md` (regex now present, RE2 subset = no backref/lookaround caveat), `docs/docs/reference/stability.md`
  (`std/regex → Experimental` row, F5), `changelogs/v0.18-current.md` `[Unreleased]` (F4). Teaching-prompt note if the prompt enumerates stdlib modules.
- Full-suite gates: `make test` + `make lint` + `make check-file-sizes` + `make verify-examples` all green.

**Acceptance**: all success criteria in the design doc §"Success Criteria" met; `make test`/`make lint`/`make check-file-sizes` green; docs updated; builtin-registry test confirms every prior builtin still present after `registerRegexBuiltins()`.

---

## Conflict-surface fixtures to keep green (from design doc)

- `std/string.ail` consumers (`examples/` using String.split/find) — unchanged.
- `std/json.ail` (marshalling pattern copied) — still type-checks/runs.
- `make verify-examples` full corpus — no regression.
- builtin-registry/startup tests — all prior builtins still registered after regex registration.

## Sequencing

M1 → M2 → M3 strictly (M2 wrappers call M1 builtins; M3 tests the whole surface). Single-worktree, single executor.
No inter-milestone barrier issues — it's a linear build.

## GPU

**None.** Pure Go builtin + AILANG stdlib + cloud-model evaluation. Does not touch ollama/local models → no `rig.lock`.

## Risks

| Risk | Mitigation |
|------|-----------|
| Byte↔rune offset mismatch (F1) | M1 headline requirement + multibyte fixture; convert exposed indices, slice text from byte offsets. |
| Nested record/list marshalling into `eval.Value` is fiddly | Copy exact patterns from `builtins_json.go` (records) + `builtins_string.go` (`_str_split` list). |
| Users expect PCRE backref/lookaround, hit `Err` | Loud specific `compile` error; LIMITATIONS + module-doc note; example framing RE2 tradeoff as a safety feature. |
| Hidden compile-cache nondeterminism/leak | Pure memoization keyed by pattern string, mutex-guarded, deterministic; compile-twice test; bounded/unbounded both deterministic. |

## Success Metrics (sprint-level)

- `std/regex` with 6 functions, all `ailang check`-clean.
- Linear-time regression test green (no catastrophic backtracking).
- Capture groups correct (spans rune-indexed, consistent with `std/string`); multibyte fixture green.
- `compile` returns `Err` (never panic) for bad/unsupported patterns.
- ≥2 runnable examples (+1 orchestration); `make verify-examples` green.
- ≥80% coverage on `builtins_regex.go`; `make test` + `make lint` + `make check-file-sizes` green.
- Docs: LIMITATIONS + stability tier + CHANGELOG updated.

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-stdlib-regex-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-STDLIB-REGEX.json`
