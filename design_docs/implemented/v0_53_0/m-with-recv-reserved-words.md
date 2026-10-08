# M-WITH-RECV-RESERVED-WORDS: Keep `with`/`recv` Reserved, Fix the Diagnostic Floor They Stand On

**Status**: Implemented on dev in #1639 (28694e3ad; closes #1623), ships in v0.53.0. Prior status: Implemented locally (2026-10-08).
**Target**: v0.53.0
**Priority**: P2 — real DX defect with one measured user report; class-wide (every reserved word) but low-frequency; no soundness impact
**Estimated**: 1.5–2 days
**Dependencies**: None hard. **Coordination** (not blocking): [M-PARSER-ERROR-CASCADE-SUPPRESSION](../../planned/m-parser-error-cascade-suppression.md) (P1, planned) owns the general suppression policy and a `parseParams` recovery point; this doc implements only the *reserved-keyword cause-naming* at that site plus a param-list-local sync, and defers general declaration-boundary suppression to that doc.
**Source**: [ailang-core triage PR #1620, Secondary 1](../../planned/ailang-core-triage/yaml-decode-loses-mapping-key-order.md) — user session `inbox_1791394438696_00b01c33` via ailang-core
**Class**: bug (DX), parser diagnostics. **No grammar change** in this revision.

## Decision

**Option (c): keep `with` and `recv` reserved, and make the reservation honest** — one clear `PAR_RESERVED_KEYWORD` diagnostic at *every* name position (today the parameter position emits **nothing** and cascades six unrelated errors), param-list-local recovery to stop that cascade, per-word suggestions naming *why* the word is reserved and what to name instead, and a real `reserved-keywords` reference page (the diagnostic's HelpURL is **404 today**).

Do **not** unreserve (a) or contextualize (b) either word in this revision. Both are recorded as the **named release trigger** on each word's owning lane instead:

- `recv` (with `send`, `timeout`): released or consumed **when [M-CSP-SESSION-TYPES](../../planned/v1_1_0/m-csp-session-types.md) freezes its syntax design**. That doc currently plans `recv` as a *function name* (`recv(ch)`, `recv : Chan[T -> S] -> (T, Chan[S]) ! Chan`), which would require unreserving — but the lane is unfrozen (its own samples use `match ... with`, a shape AILANG rejects) and has not done its parser conflict analysis. Unreserving today on unfrozen-lane evidence is speculative in the same direction as the reservation it would remove.
- `with`: released or consumed **when [M-EFFECT-HANDLERS](../../planned/v1_1_0/m-effect-handlers.md) freezes its syntax decision** (`handle e with H` vs `with H handle e` vs `match e with effect ...` — explicitly "pending parser conflict analysis" in that doc). Until then `with` keeps backing PAR019 (the measured ML `match x with` trap).

This is the same evidence discipline [M-DIALECT-KEYWORD-DIAGNOSTICS](../../planned/v1_0_0/m-dialect-keyword-diagnostics.md) applied in reverse. That doc ruled: *one measured occurrence buys one diagnostic, not a language-surface change; grammar-surface moves (aliases, keyword behavior) need named demand triggers* (two independent failures per spelling, five for aliases). It **deliberately does not cover this topic**: it rules narrowly on `case` (an identifier that was never reserved) and explicitly defers keyword-scope decisions behind evidence triggers — it never rules on *reserved-word scope*, i.e. whether `with`/`recv` should stay reserved, become contextual, or be freed. This doc supplies that missing ruling with the same pattern: **one report → diagnostic floor; grammar change → trigger.**

Two properties of option (c) settle it over (a)/(b):

1. **The diagnostic defect is class-wide; unreserving fixes two instances.** Every one of the 41 lexer keyword-map entries (`internal/lexer/token.go:265-306`) that a model picks as a parameter name hits the same silent-skip → six-error cascade (V3). An agent that names a variable `send`, `timeout`, `select`, or `channel` — all far more natural agent-chosen names than `with` — is left with the same buried cause. Unreserving `with`/`recv` leaves the class broken for the rest — most visibly the other seven zero-use reserved words (`send`, `timeout`, `channel`, `spawn`, `parallel`, `select`, `assert`; V2). The unified fix is the diagnostic, not the two-instance patch (CLAUDE.md §3: design one unified fix, not case-by-case).
2. **Reservations are reversible policy; identifier acceptance is load-bearing surface.** If `with` is unreserved today and the handler lane later freezes `handle e with H` as true keyword syntax, every interim program using `with` as a name breaks. Under (c) the only cost of waiting is a rename plus a clear error — and the release trigger is one line when the owning lane rules.

Note the trichotomy is really a dichotomy: (a) unreserve and (b) contextual keyword are **grammar-equivalent** — under both, `with`/`recv` lex as IDENT everywhere and become legal names (the contextual-keyword mechanism `LookupIdentContextual` returns IDENT unconditionally, `internal/lexer/token.go:319-330`; the only difference is vestigial map/enum entries). (c) alone preserves the grammar. Demand of one report does not buy a grammar change under this repo's own established standard.

## Demand Evidence

One user session (`inbox_1791394438696_00b01c33`, via ailang-core) picked `with`/`recv` as names and hit the cascade. Triage (PR #1620) classified it row 4 (keyword-surface) + row 3 (unreserve vs contextual vs diagnostic) and routed here.

The *mechanism*, verified live on `AILANG v0.52.5, commit 7200786` (V-log below):

| Name position | Fixture | Today's diagnostics | Names the cause? |
|---|---|---|---|
| `let` binding name | `let recv = 5` | `PAR_RESERVED_KEYWORD` **first**, then 3 unrelated (`PAR_NO_PREFIX_PARSE` ×2, `PAR015`) | ✅ but buried by cascade |
| function name | `func recv(ch: int)` | `PAR_RESERVED_KEYWORD` first, then ≥3 unrelated | ✅ but buried |
| **parameter name** | `func f(with: int)` | **6 unrelated** — `PAR_UNEXPECTED_TOKEN` ×2, `PAR_NO_PREFIX_PARSE` ×3, `PAR015`; `parseParams` (`internal/parser/parser_lambda.go:170,190`) skips the non-IDENT name **silently** (`if p.curTokenIs(lexer.IDENT)` fails, empty name, no error) | ❌ **never** |
| record field name | `type P = { with: int }` | `PAR_FIELD_NAME_EXPECTED` (generic) + 2 more | ❌ |
| record pattern field | `{a: recv} => recv` | generic `PAR_UNEXPECTED_TOKEN` + hint | ❌ |
| import alias | `import std/list as recv` | `IMP001` (clear, no reservation naming; no cascade) | ⚠️ partial |

So the triage sentence "get `expected identifier, got reserved keyword`, then a cascade" is accurate for locals and function names, and **too kind for parameters**: the keyword diagnostic does not fire at all there. The cascade after a *correct* `PAR_RESERVED_KEYWORD` (locals) comes from `peekError` (`internal/parser/parser_error.go:95-131`) reporting and then continuing to parse mid-corruption.

Also verified: the diagnostic's HelpURL `https://ailang.sunholo.com/docs/reference/reserved-keywords` **returns 404** (V9). A page exists only as a stale orphan in the pre-reorg layout (`docs/reference/reserved-keywords.md`); the served site builds `docs/docs/` and `docs/sidebars.js` has no `reference/reserved-keywords` entry. Every `PAR_RESERVED_KEYWORD` emitted today links to a dead page. (The orphan page also misdescribes `with` as "Pattern guard in match (future feature)" — guards are `if`; `with` backs PAR019.)

## Current State of the Two Words

- `with` — in the keyword map (`internal/lexer/token.go:275`), enum (`:39`), `tokens` map (`:164`), `IsKeyword` (`:395`). **Exactly one parser use**: PAR019, the ML/Haskell `match x with` trap detector at `internal/parser/parser_expr.go:222` inside `parseMatchExpression` (V2; the only other reference, `internal/parser/parser_error.go:117`, is suggestion *rendering*, not a parse decision). PAR019 fires with a two-suggestion actionable fix (V5).
- `recv` — keyword map (`:295`), enum (`:59`), `tokens` map (`:184`), `IsKeyword` (`:399`). **Zero parser uses** (V2). It sits in the reserved session/concurrency cluster `SEND, RECV, TIMEOUT, AS, DERIVING` (`:399`) — but note the cluster is not uniformly reserved-for-nothing: `AS` is live (import aliasing, `internal/parser/parser_file.go:301,330,360`) and `DERIVING` is live (`internal/parser/parser_type_decl.go:62`). The zero-use set is `SEND, RECV, TIMEOUT, CHANNEL, SPAWN, PARALLEL, SELECT, ASSERT` (V2).
- `RECV` has no AST, elaboration, typechecker, or eval surface: `grep -rln RECV internal/ cmd/` hits only `internal/lexer/token.go` (V2).

## Goals

1. **One clear, coded, first-position diagnostic for a reserved word in any name position** — parameter, `let` binding, function name, record field — naming the word, the reservation, and a copyable alternative.
2. **Stop the keyword-caused cascade** at the sites this doc touches (param list; binding), without hiding genuinely independent errors.
3. **A real reserved-keywords reference page** at the URL the diagnostic already prints, with an accurate per-word table (current use or owning lane + trigger).
4. **A named, reviewable release/consume trigger per word**, so "is this still reserved, and why?" has an owner and a resolution path instead of archaeology.
5. Preserve every diagnostic that already works (`PAR_RESERVED_KEYWORD` at locals/function names stays first; PAR019 unchanged; `IMP001` unchanged).

## Non-Goals

- **No grammar or lexer change**: no unreservation, no contextualization, no new keywords, no new acceptance. `internal/lexer/token.go`'s keyword map, enum, and `IsKeyword` are untouched.
- No general parser-error suppression policy — that is M-PARSER-ERROR-CASCADE-SUPPRESSION's (P1) scope; this doc only adds keyword-specific cause-naming and the minimal recovery the two docs' boundary needs.
- No PAR019 redesign (its post-error cascade is the suppression doc's policy, not this doc's; this doc changes no `match` parsing).
- No ruling on the *other* zero-use reserved words (`send`, `timeout`, `channel`, `spawn`, `parallel`, `select`, `assert`) beyond: they stay reserved, they gain the same diagnostic floor (goal 1 is class-wide by construction), and each names its owning lane on the page.
- No evaluation pass-rate claim. This is an agent self-repair ergonomics fix; measure via `ailang eval-paired` smoke tier as usual, but do not gate on it.

## Solution Design

### R1 — Parameter-position diagnostic (the reported defect)

In `parseParams` (`internal/parser/parser_lambda.go:157`, both the first-param and `COMMA`-loop copies at `:170` and `:190`): when the current token at a name position is a keyword (`!p.curTokenIs(lexer.IDENT) && p.curToken.IsKeyword()`), emit `PAR_RESERVED_KEYWORD` with the existing message shape (`expected identifier, got reserved keyword '<word>'`, `internal/parser/parser_error.go:96`) **at that token's position**, then synchronize within the parameter list: consume tokens until `RPAREN` (inclusive) or `COMMA` (then continue parsing the next parameter normally). The parameter under construction is dropped (`params` unchanged) — do not fabricate a name.

This is deliberately *param-list-local*: no declaration-boundary jumping here. A `func` whose parameter list was corrupted still surfaces its own downstream diagnostics honestly; only the mid-param cascade is stopped.

### R2 — Keyword-aware branch at record field name

In `parseRecordFieldDef` (`internal/parser/parser_type.go:342-346`), when the failing token is a keyword, the `PAR_FIELD_NAME_EXPECTED` report gains the same "reserved keyword '<word>'" message and suggestions instead of the generic "Add field name". No recovery beyond what exists (return `nil` as today).

### R3 — Per-word suggestions in `peekError`

Extend the context-specific suggestion switch (`internal/parser/parser_error.go:103-121`) with the reservation's *reason and owner*, so the diagnostic is actionable rather than merely true:

- `SEND, RECV, TIMEOUT`: "reserved for the planned CSP/session-types lane (see design_docs/planned/v1_1_0/m-csp-session-types.md); use `tx`/`rx`/`deadline`/`expires` instead".
- `CHANNEL, SPAWN, PARALLEL, SELECT`: "reserved for the planned concurrency lane; use `chan`/`fork`/`concurrent`/`pick` instead".
- `WITH` (existing `MATCH, WITH` case): keep, but name the real reason — PAR019 dialect detection plus the planned `handle ... with` handler syntax — and suggest `using`/`given`/`w`.
- `ASSERT`: "reserved for the planned assertion builtin; use `check`/`verify`".

Suggestion *wording* is implementer-adjustable; the per-word table on the reference page (R4) is the source of truth and the two must stay in sync (test: suggestion text mentions the same alternative names as the page).

### R4 — Reserved-keywords reference page (fixes the 404)

Create `docs/docs/reference/reserved-keywords.md` in the **current** layout with sidebar entry `reference/reserved-keywords` in `docs/sidebars.js` (Reference group, near `language-syntax`). Per word: current use, or owning lane + release trigger. Delete the stale orphan `docs/reference/reserved-keywords.md` (pre-reorg layout, not built, links nowhere in the sidebar) and regenerate `docs/llms.txt` (it embeds the stale page's text around line 1154). The HelpURL printed by `PAR_RESERVED_KEYWORD` (`internal/parser/parser_error.go:129`) then resolves — no parser change to the URL itself.

The page's count claim must match the lexer: the current map has **41** entries (V1), not the stale page's "43".

### R5 — Release/consume triggers recorded

No code. The triggers are this doc's Decision section, mirrored per-word on the reference page: each zero-use reserved word names its owning lane; a lane freezing its syntax design must rule each of its words consumed (real keyword) or released (delete from the `keywords` map — then also enum/`tokens`/`IsKeyword`, and the reference page). Until a lane rules, the word stays reserved with the R1-R3 floor. This gives the (a)/(b) question a first-class answer with an owner instead of a standing ambiguity.

### Files

- `internal/parser/parser_lambda.go:157-195` — R1 (both `curTokenIs(lexer.IDENT)` copies; factor the name-position check into a small helper rather than duplicating).
- `internal/parser/parser_type.go:342-346` — R2.
- `internal/parser/parser_error.go:103-131` — R3 (suggestion switch; message construction shared with R1/R2).
- `docs/docs/reference/reserved-keywords.md` (new), `docs/sidebars.js`, `docs/llms.txt` (regenerated), delete `docs/reference/reserved-keywords.md` — R4.
- `internal/parser/*_test.go` (new focused fixtures; extend `error_recovery_test.go`) — tests.
- **Untouched**: `internal/lexer/` (no change at all), `internal/elaborate/`, `internal/types/`, `internal/eval/`, `internal/codegen/`, formatter, CLI production code.

Estimated ~60–90 production lines + ~150 test lines.

## Conflict Surface

Parser name-position sites only; no lexer change.

| Site | Other traffic through the site | Regression risk | Required guard |
|---|---|---|---|
| `parser_lambda.go:170,190` (`parseParams` name positions) | every function and lambda parameter — plain `IDENT`, `name: type`, multi-param `COMMA` lists, zero-arg `()` | recovery over-consuming past `RPAREN` into the function's type/body/next declaration | param-list-local sync bound tests: typed/untyped, multi-param, nested lambda, two-defect file; the two-arg `\a b.` form if reachable at this site |
| `parser_type.go:342` (`parseRecordFieldDef`) | all record-type field defs — `IDENT : type` traffic | none new (no recovery added; message only changes for the keyword branch) | existing record-type fixtures; generic-bad-token case still emits old message |
| `parser_error.go:103-121` (suggestion switch) | rendering of *every* reserved-keyword diagnostic | changing existing suggestions users/tests pin | existing `PAR_RESERVED_KEYWORD` tests keep passing except where suggestion text is asserted; update those assertions in the same commit |
| docs (page, sidebar, llms.txt) | every docs build; the 404 target | none — the URL gains a page | site build passes; `curl` the route in the implementation report |

**Programs that must still work post-change** (all in the V-log; add as fixtures):

1. `module probe/valid_control` — `func double(x: int) -> int = x * 2` — checks `ok` today (V8), must stay `ok`.
2. `let with = 5`-style fixtures keep failing with `PAR_RESERVED_KEYWORD` **first** (V4 baseline), and `match x with {...}` keeps firing exactly one PAR019 (V5 baseline) — PAR019 is untouched.
3. The five identifier contexts from M-DIALECT-KEYWORD-DIAGNOSTICS (a parameter, binding, bare value, field access, call named `case`) — `case` is IDENT traffic, unaffected.
4. Existing `error_recovery_test.go` multi-error fixtures: genuinely independent errors must both still appear (no suppression added here).
5. Import alias fixture still emits exactly one `IMP001` (V7 baseline).

**Intentional incompatibility**: none for valid AILANG. Invalid programs that name a parameter with a reserved word change from six uncoded/uncorrelated diagnostics to exactly one coded `PAR_RESERVED_KEYWORD` (plus honest independent errors elsewhere in the file). No lexer tokens, grammar productions, or accepted programs change.

## Milestones

**M1 — Parameter/field diagnostic + suggestions (1 day)**, independently landable.
- R1 (param site, both copies), R2 (field site), R3 (suggestion switch).
- Focused parser fixtures; two-defect and recovery-boundary tests; suggestion/page sync test.

**M2 — Reference page + 404 fix (0.5 day)**, landable after M1 (the page's per-word table cites the suggestion text).
- R4 page, sidebar, delete orphan, regenerate `docs/llms.txt`; R5 triggers recorded on the page.

**M3 — Quorum (optional, pre-sprint)**: `ailang design-quorum` per the design-doc-creator skill; not a gate for this P2 doc.

## Acceptance Criteria

Each AC names its V-log baseline (measured on v0.52.5 in this doc) and the mutation it kills.

1. `ailang check --relax-modules --format agent` on `func f(with: int) -> int` and `func double(recv: int) -> int` fixtures exits nonzero with **exactly one** `PAR_RESERVED_KEYWORD` naming `with`/`recv`, and **zero** of the baseline cascade codes (`PAR_UNEXPECTED_TOKEN` at `:`, `PAR_NO_PREFIX_PARSE`, `PAR015` from the param corruption — V3 baseline). **Mutation killed**: suggestion-only reporting without param-list sync (cascade would persist); reporting at the wrong position.
2. A multi-parameter fixture `func g(a: int, recv: int, b: int) -> int` recovers: diagnostics name `recv`; `a` and `b` parameters still parse and type-check (fixture with `g(1, 2, 3)` body). **Mutation killed**: sync that over-consumes past `RPAREN` or eats `b`'s binding.
3. A two-defect fixture (reserved word in a parameter *and* an independent malformed expression later in the file) reports both the `PAR_RESERVED_KEYWORD` and the independent error. **Mutation killed**: global error-list clearing or sync to EOF.
4. `let recv = 5` and `func recv(...)` fixtures keep `PAR_RESERVED_KEYWORD` **first** with count unchanged or reduced vs V4 baseline (1 clear + ≤3 cascade), and the new suggestions present. **Mutation killed**: suggestion-switch regression that reorders or drops the primary.
5. `match x with { ... }` fixture output is byte-identical to V5 baseline (one PAR019, same suggestions). **Mutation killed**: any accidental touching of `parseMatchExpression`.
6. `docs/docs/reference/reserved-keywords.md` exists, is in `docs/sidebars.js`, the orphan `docs/reference/reserved-keywords.md` is deleted, `docs/llms.txt` regenerated, and `curl -s https://ailang.sunholo.com/docs/reference/reserved-keywords` returns 200 post-deploy (locally: the docusaurus build emits the route). The page lists 41 keywords with owning-lane/trigger for each zero-use word. **Mutation killed**: landing the page in the stale layout (still 404).
7. `go test ./internal/parser/... ./internal/lexer/...` exits 0 (lexer must be untouched — any lexer test churn means scope escape). **Mutation killed**: grammar change smuggled into a diagnostics doc.
8. Existing `PAR_RESERVED_KEYWORD` tests and `error_recovery_test.go` fixtures pass, with suggestion-text assertions updated in the same commit. **Mutation killed**: stale assertions masking rendering regressions.

## Test Plan and Mutation Matrix

| Test | Downstream observable | Mutation killed |
|---|---|---|
| param keyword fixture (with, recv, send) | exactly one `PAR_RESERVED_KEYWORD` at the word's position | silent skip (today's base); wrong position |
| multi-param recovery fixture | `a`,`b` still parse; call type-checks | sync over-consume |
| two-defect fixture | both diagnostics present | suppression to EOF |
| let/func-name fixtures | primary stays first; count ≤ baseline | switch regression |
| PAR019 fixture diff | byte-identical to V5 | match-parser drift |
| field-name fixture | keyword-aware message at `{with: int}` | generic message kept for keyword case |
| valid control + `case` identifier contexts | `ok` / unchanged | stolen identifier traffic |
| docs build + route check | route resolves, orphan gone | stale-layout landing |

## Risks

| Risk | Mitigation |
|---|---|
| Param sync mis-bounded (eats a valid next parameter or declaration) | M1 tests 2; sync loop halts at `RPAREN`/`COMMA`/`EOF` only |
| Suggestion churn breaks downstream pinning (eval prompts cite suggestion text) | grep `docs/docs/prompts/` + eval fixtures for pinned text in the sprint; prompt table itself needs **no change under (c)** (no grammar change) — V12 |
| Docs-layout drift repeats the 404 (page built in stale location) | AC6 route assertion; sync via the docs registry script that generated sidebars.js header |
| Doc overlaps M-PARSER-ERROR-CASCADE-SUPPRESSION's `parseParams` RP (that doc cites `parser_expr.go:750`, now `parser_lambda.go:157` — drifted) | this doc names the current site; sprint planner should update the cascade doc's citation and sequence the two RPs; keyword cause-naming is complementary to suppression, not duplicated |
| A future lane freeze flips a word to released/consumed, making this doc's page stale | R5 trigger is recorded per word on the page; page update is part of any lane freeze that rules |
| Demand evidence stays at n=1 | AC outputs are cheap; if a second independent report arrives before sprint, re-open the (a)/(b) question with the trigger satisfied |

## Axiom Compliance

| Axiom | Score | Rationale |
|---|---:|---|
| A1 Determinism | 0 | no runtime/semantic change |
| A2 Replayability | 0 | no trace change |
| A3 Effect Legibility | 0 | no effect surface change |
| A4 Explicit Authority | 0 | no capability change |
| A5 Bounded Verification | +1 | exact-count fixtures with baselines and named mutations |
| A6 Safe Concurrency | 0 | reservation for the concurrency lane unchanged |
| A7 Machines First | +1 | the machine-readable *first* diagnostic names the cause and a copyable fix; today the param case emits six uncoded/uncorrelated errors |
| A8 Minimal Syntax / Frozen Core | +1 | the *conservative* ruling: zero grammar change, no speculative release or reservation growth |
| A9 Cost Visibility | 0 | — |
| A10 Composability | 0 | — |
| A11 Structured Failure | +1 | one real mistake → one coded, action-linked failure; cascade stopped locally, independent errors preserved |
| A12 System Boundary | 0 | — |

**Net: +4.** No hard A1/A3/A4/A7 violation.

## Verification Log

Probes run with PATH `ailang` (`AILANG v0.52.5, Commit 7200786`); fixtures in a temp dir (MOD010 auto-relax observed on the temp path). `--relax-modules --format agent` used throughout, per the M-DIALECT probe convention. Output below is abridged to the diagnostic lines; full transcripts ran in-session.

| ID | Exact command | Observed output |
|---|---|---|
| V1 | `grep -n '"with":\|"recv":\|^WITH\|^RECV\|WITH:\s*"with"\|RECV:\s*"recv"' internal/lexer/token.go` | `39 WITH`, `59 RECV`, `164 WITH: "with"`, `184 RECV: "recv"`, `275 "with": WITH`, `295 "recv": RECV`, `395 MATCH, WITH, ...`, `399 SEND, RECV, TIMEOUT, AS, DERIVING`. Keyword-map entry count via `awk '/^var keywords/,/^}/' \| grep -c ':'` → **41**. |
| V2 | `grep -rn "lexer.WITH\b\|lexer.RECV\b" internal/ cmd/ --include="*.go"` (scoped to parser); same for `SEND/TIMEOUT/CHANNEL/SPAWN/PARALLEL/SELECT/ASSERT` | `lexer.WITH`: exactly 2 hits — `internal/parser/parser_expr.go:222` (PAR019 trap, `parseMatchExpression`) and `internal/parser/parser_error.go:117` (suggestion rendering). `lexer.RECV`: **zero** parser hits; `grep -rln RECV internal/ cmd/` → only `internal/lexer/token.go`. `SEND`, `TIMEOUT`, `CHANNEL`, `SPAWN`, `PARALLEL`, `SELECT`, `ASSERT`: zero parser hits. Cluster counterexamples: `AS` live at `parser_file.go:301,330,360`; `DERIVING` live at `parser_type.go:408`, `parser_type_decl.go:62`. |
| V3 | `ailang check --relax-modules --format agent /tmp/kwprobe/recv_param.ail` (and `with_param.ail`) | rc≠0; six diagnostics, **zero** `PAR_RESERVED_KEYWORD`: `PAR_UNEXPECTED_TOKEN` "expected ), got :" (2:17), `PAR_UNEXPECTED_TOKEN` "expected {, got :" (2:17), `PAR_NO_PREFIX_PARSE` `:` (2:17), `PAR_NO_PREFIX_PARSE` `)` (2:22), `PAR_NO_PREFIX_PARSE` `->` (2:24), `PAR015` bare assignment (2:27). Cause `recv` never named. Mechanism: `internal/parser/parser_lambda.go:170,190` `if p.curTokenIs(lexer.IDENT)` → silent empty name. |
| V4 | `ailang check --relax-modules --format agent /tmp/kwprobe/recv_local.ail` (and `with_local.ail`) | rc≠0; `PAR_RESERVED_KEYWORD at 3:7: expected identifier, got reserved keyword 'recv'` **first**, with suggestions + HelpURL; then 3 cascade: `PAR_NO_PREFIX_PARSE` `recv` (3:7), `PAR015` (3:12), `PAR_NO_PREFIX_PARSE` `recv` (4:3). `with_local.ail` identical shape with `with` + "Pattern matching keywords cannot be used as names" (the `parser_error.go:117` case). |
| V5 | `ailang check --relax-modules --format agent /tmp/kwprobe/match_with.ail` | `PAR019 at 2:50: 'match ... with' is not valid AILANG syntax (ML/Haskell pattern detected)` + 2 suggestions; then 3 cascade (`PAR_NO_PREFIX_PARSE` `with`, uncoded `expected ; or }, got =>`, `PAR_NO_PREFIX_PARSE` `=>`). PAR019 unchanged by this doc (AC5 baseline). |
| V6 | `ailang check --relax-modules --format agent /tmp/kwprobe/field.ail` | `PAR_FIELD_NAME_EXPECTED at 2:12: expected field name` (generic) + `PAR_TYPE_RBRACE_MISSING` + `PAR_NO_PREFIX_PARSE` — reservation never named (R2 baseline). Pattern-field fixture `probe/pattern.ail`: generic `PAR_UNEXPECTED_TOKEN` + hint. |
| V7 | `ailang check --relax-modules --format agent /tmp/kwprobe/alias.ail` | exactly one `IMP001 at 2:20: expected module alias identifier after 'as'` — clear, no cascade, no reservation naming (acceptable; no change planned). |
| V8 | `ailang check --relax-modules --format agent /tmp/kwprobe/valid_control.ail` | `ok`, rc=0 — the negative control for every probe above. |
| V9 | `curl -s -o /dev/null -w "%{http_code}" https://ailang.sunholo.com/docs/reference/reserved-keywords` | **404**. Site builds `docs/docs/` (`docs/docusaurus.config.js:133-139`, routeBasePath `/docs`); no `reserved-keywords` entry in `docs/sidebars.js` Reference group (`grep -n reference docs/sidebars.js` → only the `docs/docs/reference/` files); stale orphan exists at `docs/reference/reserved-keywords.md` (pre-reorg layout, claims "43 keywords", describes `with` as "Pattern guard in match (future feature)" — guards are `if`); `docs/llms.txt:1154` embeds the stale text. |
| V10 | `grep -rn "reserved" docs/docs/prompts/current.md` | prompt §Reserved Keywords at line 2295: "AILANG reserves 43 keywords", `with` under Control Flow, `recv` under Concurrency — **no prompt change under (c)** (no grammar change), but the count is stale vs the 41-entry map (fix opportunistically in the next prompt revision, not this doc's AC). |
| V11 | `sed -n '310,332p' internal/lexer/token.go; grep -rn -A3 "func (p \*Parser) peekIsContextualKeyword" internal/parser/` | Option (b)'s mechanism exists: `LookupIdentContextual` (`token.go:319-330`) returns IDENT for `test/tests/property/properties`; parser precedent `peekIsContextualKeyword` (`parser_test_decl.go:164-166`) = `peekTokenIs(IDENT) && Literal == keyword`. Recorded because the (a)≡(b) grammar-equivalence claim rests on it. |
| V12 | `grep -n "recv\|with" design_docs/planned/v1_1_0/m-csp-session-types.md; grep -n "with" design_docs/planned/v1_1_0/m-effect-handlers.md` | m-csp lines 123-145: `recv(ch)`, `recv : Chan[T -> S] -> (T, Chan[S]) ! Chan`, `send : Chan[!T -> S] -> T -> Chan[S] ! Chan` — **function form**; line 123 sample also uses `match ... with` (a shape AILANG rejects → lane samples are unfrozen sketches). m-effect-handlers line 95: "`handle expr with H`"; line 108: syntax form "pending parser conflict analysis below", high-impact, owner: human. |
| V13 | `git log --oneline -3 -- internal/lexer/token.go; git status --short` | token.go history shows the triage commit (#1620) as the only recent touch; worktree clean apart from this document (probes lived in /tmp). |

## Related Documents

- [M-DIALECT-KEYWORD-DIAGNOSTICS](../../planned/v1_0_0/m-dialect-keyword-diagnostics.md) — the sibling this doc completes. Distinction: that doc rules on an *unreserved* identifier's dialect trap (`case`→`match`) and is **PARKED needs-human-review** on its arbitrary-lookahead detection mechanism; this doc's detection sites are all fixed-position (parameter name, field name — no lookahead past unbounded expressions), so the parking does not block it. Its evidence-trigger pattern is the standard this doc applies to reserved-word scope.
- [M-PARSER-ERROR-CASCADE-SUPPRESSION](../../planned/m-parser-error-cascade-suppression.md) — P1 general suppression policy (cap, primary-prominence, declaration-boundary recovery points incl. `parseParams`). Boundary: that doc suppresses *reporting*; this doc adds *cause-naming* and param-list-local recovery for the reserved-word class. Its `parseParams` citation (`parser_expr.go:750`) has drifted to `parser_lambda.go:157`.
- [M-CSP-SESSION-TYPES](../../planned/v1_1_0/m-csp-session-types.md) — owning lane for `recv`/`send`/`timeout` reservations; plans function form.
- [M-EFFECT-HANDLERS](../../planned/v1_1_0/m-effect-handlers.md) — prospective consumer of `with` (`handle e with H`); syntax unfrozen.
- [Triage PR #1620, Secondary 1](../../planned/ailang-core-triage/yaml-decode-loses-mapping-key-order.md) — provenance and the row-3/row-4 routing. Its two sibling items (yaml mapping-key order, std yaml encode) are separate work items per its dispatch note.
- [M-V1-STABILITY-PROMISE](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/) — pre-1.0: no tiered promise covers keyword surface yet; the per-word trigger table (R5) is the seed of one.

## Dispatch Note

Independent of the triage report's other two items (yaml key order, yaml encode) — different subsystem (parser diagnostics + docs), different urgency. Quorum (optional per skill) before sprint-planner if the controller wants an off-Anthropic second opinion.

---

**Document created**: 2026-10-07