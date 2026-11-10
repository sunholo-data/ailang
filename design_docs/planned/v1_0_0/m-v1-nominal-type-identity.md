# M-V1-NOMINAL-TYPE-IDENTITY: a type is its name *and* its module, and a name must resolve to exactly one thing

**Status**: Planned
**Target**: v0.58.x (the last breaking window before the 1.x promise)
**Priority**: P0 (release-readiness review §3.1 a and d; "the two findings I would not ship 1.0 with")
**Estimated**: 9–14 working days across 6 milestones (an estimate, not a plan; see §11)
**Dependencies**: none to start. Consumes [m-v1-downstream-compat-gate](m-v1-downstream-compat-gate.md) for
acceptance. Interacts with the exhaustiveness fix (#2146, review §3.1b), which needs the qualified ADT
identity introduced here to find the right constructor set.
**Planner-Lane**: opus-required (type representation, unifier, interfaces, cache key)
**Supersedes (type namespace only)**: M4 of [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md);
[m-unbound-type-names](../v0_57_1/m-unbound-type-names.md) (#1485); the type/constructor half of
[m-import-vs-import-collision](../v0_54_20/m-import-vs-import-collision.md) and
[m-import-alias-collision](../v0_55_0/m-import-alias-collision.md) (#1980). Their verification logs stay
the evidence; this doc unifies them behind one type-name resolver so they are not built three times.
**Measured at**: `dev` = `51f6636` (binary built from it), 2026-11-10.

## 1. Problem

AILANG identifies a nominal type by its bare name. `types.TCon` is `struct { Name string }` and
`Equals` compares only `Name` (`internal/types/types.go:57-69`). Every module that declares `type T`
mints the same `TCon{"T"}`, so two unrelated ADTs unify. Four defects follow from the same root, or from
the absence of one authoritative "what does this name mean here" step:

| # | Defect | Repro (all run on `51f6636`) | Today |
|---|---|---|---|
| a | Two same-named ADTs from different modules are one type | `q/a: type T = A(int)`, `q/b: type T = A(string)`; `main` imports `mk` from a and `T, A` from b, matches `A(s) => toUpper(s)` | `check` ✓, `run` → `_str_upper: arg 0 - expected string, got int`; `--strict-bytecode` → `vm: BUILTIN_CALL: __str_upper: expected string` |
| a′ | Same, with disjoint constructors | b's `T = B(string)`, match `B(s)` on a's value | `check` ✓, `run` → `no pattern matched` |
| a″ | A **local** ADT captures an imported one | `main` declares `type T = A(string)` and calls a's `mk()` | `check` ✓, `run` → `_str_upper` error. `LIMITATIONS.md:145` says local capture "retains TC_TYPE_SHADOW_001"; that is true only for aliases |
| a‴ | Structural twins cross silently | a and b each `type Color = Red \| Green`; `b.name(a.pick())` | `check` ✓, prints `green` — works by accident; this is the class the fix deliberately rejects |
| d1 | Undeclared type names (#1485) | `func f(r: Nope) -> int`, `type W = { x: Nada }` | `check` ✓ |
| d2 | Duplicate top-level funcs (#1278) | `f(x) = x + 1` then `f(x) = x * 100` | `check` ✓, `f(2)` = 200 (last wins) |
| d3 | Same type name imported from two modules (#1980) | `import a (Leg)` + `import b (Leg)` | `check` ✓, last import wins |
| d4 | Duplicate type declarations in one module (new) | `type T = A(int)` twice | `check` ✓ |
| d5 | Two local ADTs declare the same constructor (new) | `type U = A \| B`, `type V = A \| C`; `f(A)` | `check` ✓, `A` silently means V's |

Probe sources: `scratchpad/dd-nominal/r/{q,v2,v3,v6,v8,v9,v10,v12,v13,v14,v15}` in this session's
scratchpad. A legal program that uses two same-named ADTs *disjointly* (`v15`: `Ma.desc(Ma.mk())`,
`Mb.desc(Mb.mk())`, different constructor orders) prints the right answers on both the evaluator and
the VM today, and must keep doing so.

**Where bare names are the identity today** (line numbers at `51f6636`):

- **Type representation:** `TCon{Name}` (`internal/types/types.go:57`). About 126 sites read a
  TCon's name across `internal/` (40 in `types`, 12 in `pipeline`, 7 each in `repl` and `gen/lower`).
- **Annotation → type:** the elaborator's `astTypeToInternalType` default arm returns
  `&types.TCon{Name: typ.Name}` for any unknown name (`internal/elaborate/file_funcs.go:353-359`); this
  is both d1 and the minting point for a.
- **Imports:** `resolveSelectiveImports` binds types, aliases and constructors by bare name, and
  auto-imports *every* alias (`pipeline_module_imports.go:278`) and *every* constructor (`:297`, first
  write wins) of each direct dependency; transitive aliases are pulled from the whole import closure
  (`:151`). `resolveConstructorImport` names the factory `$adt.make_<T>_<C>` (`:316`), so `q/a.T.A` and
  `q/b.T.A` share one `ExternalTypes` key and the later import overwrites the earlier scheme.
- **Checker tables:** `ctorTypes` (ctor → ADT name) and `adtTypeParams` (ADT name → arity), local
  overriding imported (`pipeline_module_compile.go:90-111`).
- **Interfaces:** `iface.ConstructorScheme{TypeName, CtorName, …}` and `TypeExport{Name, Arity}`
  (`internal/iface/iface.go:26-47`). The JSON renders a TCon as `typ.Name` (`internal/iface/json.go:255`);
  the registry hash projection (`iface/hash_projection.go:23`) and the stdlib goldens hash that string.
  The cache digest (`iface/digest.go:30`) also renders bare names.
- **Instances:** derived Eq is keyed by lowercased bare name (`types/dictionaries.go:422`), so two
  modules deriving Eq on `Row` collide and the duplicate is swallowed (audit row 9 of the M-TYPE-NAME-SHADOW doc).
- **Run time:** `eval.TaggedValue` already carries `ModulePath` (`internal/eval/value.go:429`) and the
  constructor resolver reports `ambiguous constructor … found in multiple modules`
  (`internal/runtime/resolver.go:203`), but pattern matching and equality compare `CtorName` only
  (`eval_patterns.go:187`, `eval_equality.go:68`). Bytecode ADT tables are keyed by bare name
  (`bytecode/compiler/collections.go:82`, `bridge_safety.go:121`).
- **Merges by bare name:** SMT `adtTypes`/`recordAliases`, first write wins (`internal/smt/verify.go:92-126`);
  REPL/WASM module registry (`internal/repl/module_registry_load.go:82, 241-256`).
- **Duplicates:** `checkDuplicateModuleBindings` (MOD007) checks let-vs-func and let-vs-let and assumes
  "funcs are declared, so unique by the parser's own rules" (`internal/elaborate/file.go:16-18`). They are not.

**Impact.** The pitch is that the checker makes AI-written code trustworthy. a/a′/a″ are a checker that
says ✓ and a program that dies with a type error. Agents reuse `Row`, `Item`, `Config`, `Error`, `Event`
constantly (the M-TYPE-NAME-SHADOW doc's words, and the registry agrees: `Event` is exported by three
packages, `Verdict` by `std/stream/bridge` and `motoko_ext_compose`).

## 2. Goals

**Primary goal:** every program `check` accepts can no longer fail at run time because two types or two
definitions shared a name; every type name in a signature resolves to exactly one declaration or is a
coded error at its own line.

**Success metrics:**
- All ten rows of the §1 table reject at `check` with a coded error (a‴ included), and `v15` still runs.
- Zero change in `check` verdicts across the in-repo corpus and the downstream gate, apart from the
  rows listed in §8 (measured blast radius).
- `interface_hash` of every published package unchanged (no republish forced by identity).
- One resolver: d1, d3 and a share one type-name scope table, not three.

## 3. High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1 Identity = (defining module, name), stored as `TCon.Module` | Representation used by unifier, interfaces, cache, codegen | human | design | high |
| D2 Display stays the bare name; diagnostics qualify only when two distinct types print the same | Error text and goldens are user-facing | agent | compile | low |
| D3 Registry hash projection keeps rendering bare names; the internal cache digest renders qualified | Decides whether every published package must republish | human | design | high |
| D4 Builtin-minted nominal types get a home module from one table (`Json` → `std/json`, …) | A missed entry is a false rejection of a valid program | agent | compile | med |
| D5 Auto-import channels stay; resolution order is explicit and ambiguity at use is an error | Removing them rejects about 500 annotation sites downstream | human | design | high |
| D6 Duplicates (func, type, constructor) in one module are MOD007; two explicit imports of one bare type/ctor name are MOD015 | Error-code registry is public | human | design | low |
| D7 Errors, not warnings, in dependency code too | Decides whether the `ailang_parse` package must republish | human | design | med |

### Design freeze

- [ ] D1 qualified identity (vs Alternative B, §5)
- [ ] D3 hash projection unchanged
- [ ] D5 auto-import channels kept, order fixed
- [ ] D6 MOD015 (not MOD016) for import-vs-import
- [ ] D7 hard error everywhere, republish `ailang_parse`

### Deferred decisions

- Spelling of the qualifier in diagnostics (`T (from q/a)` vs `q/a.T`) — agent, consistent with `TC_TYPE_SHADOW_001`.
- Whether `TaggedValue` matching also compares `ModulePath` (defence in depth) — agent; not required for soundness.
- Go-codegen type-name mangling for same-named ADTs — agent; `--emit-go` is not Stable.
- Levenshtein suggestion thresholds for `TC_UNBOUND_TYPE_001` — agent, per m-unbound-type-names D5.

## 4. Design

### 4.1 Representation

`types.TCon` gains `Module string`. `Equals` compares `(Module, Name)`. `String()` returns `Name`.
`Module == ""` is legal only for primitives (`int`, `string`, …) and builtin structural heads (`Array`,
`Map`, `List`, which already normalise to `TArray`/`TMap`/list). After elaboration a nominal TCon with an
empty module is an internal error: a test-only hook in `internal/types` panics on it, and release
builds report it as an internal compiler error. A forgotten minting site therefore fails loudly instead
of re-opening the hole. (No new environment variable: new ones must go through the `internal/config`
registry, and a test hook is enough.)

Keyed struct literals keep compiling (`TCon{Name: "x"}`), which is what makes the change incremental:
M0 adds the field, then each minting site is fixed under a counter that reports empty-module nominal
TCons reaching the unifier.

### 4.2 Minting sites

1. **Elaborator** — `astTypeToInternalType` resolves a bare name through a `TypeScope` (§4.3) and stamps
   the defining module. Type-application heads (`T[a]`) go through the same lookup.
2. **Interface builder** — ADT `ResultType`, constructor `FieldTypes` and export schemes carry the
   defining module (an extension of M-ALIAS-BODY-CLOSURE, which already closes alias bodies in their
   defining module). Interfaces are the only way a type crosses a module, so a qualified interface
   is enough for an importer that never names the type (row a).
3. **Builtins** — `types.Con(name)` (builder) looks the name up in one `nominalHome` table:
   `Option` → `std/option`, `Result` → `std/result`, `Json` → `std/json`, `XmlNode` → `std/xml`, and so on;
   names with no `.ail` declaration (`DOMEvent`, `StreamConn`, `ProcessHandle`, …) get `$builtin`. A unit
   test walks every registered builtin signature and fails on a nominal TCon without a home, and checks
   that each home module really declares the type.

### 4.3 Type-name resolution (one table for a, d1, d3)

At each annotation, in order:

1. a type parameter in scope;
2. a type declared in this module;
3. a type bound by an explicit selective import (`import M (T)`, `import M (T as U)` binds `U`);
4. the prelude (`Option`, `Result` from `internal/loader/prelude_imports.go`);
5. a type exported by a **direct** dependency through the auto-import channels (M-CTOR-AUTO, M-TYPE-ALIAS);
6. an alias reached transitively (M-TRANSITIVE-ALIAS-ENV-IMPORT).

The first rule that matches wins, which keeps today's local-wins and explicit-beats-auto behaviour
(`examples/alias_body_closure/main.ail` depends on it). If rule 5 or 6 matches more than one *distinct*
`(Module, Name)`, the use is `TC_AMBIGUOUS_TYPE_001`; nothing is ambiguous until it is used, so a module
that merely imports two packages exporting `Event` is unaffected. No rule matching is
`TC_UNBOUND_TYPE_001` (#1485), with the suggestion set from m-unbound-type-names D5.

Constructors resolve the same way in their own namespace (types and constructors stay separate:
`mk-main/src/eval/journal/scan.ail` imports record type `Interaction` from one module and constructor
`Interaction` from another, and that stays legal). `ConstructorPattern.TypeName` and the `$adt` factory
key become qualified (`make_<module>_<T>_<C>`, escaped), which removes the `ExternalTypes` overwrite.

### 4.4 Duplicates

MOD007 grows three cases in `checkDuplicateModuleBindings`: func declared twice, type declared twice,
and one constructor name declared by two local types. Two explicit imports binding the same bare type or
constructor name are MOD015 at the import site, as [m-import-vs-import-collision](../v0_54_20/m-import-vs-import-collision.md)
proposes (its registry text already says "or another import"). Values keep that doc's use-site rule.

### 4.5 Everything else that keys by name

- **Unifier diagnostics:** a mismatch whose two sides print identically becomes `TC_NOMINAL_001` and
  names both modules.
- **Derived Eq / dictionaries:** key `prelude::Eq::<module>::<name>`; duplicates become impossible rather
  than swallowed.
- **SMT:** `adtTypes`/`recordAliases` keyed by qualified name; first-write-wins merge removed.
- **REPL/WASM:** the registry loader builds its scope with the same `TypeScope`. `repl` is a Stable
  command, so the deferral noted in `LIMITATIONS.md:148` ends here.
- **Bytecode:** ADT tables keyed by qualified name. This also narrows the LIMITATIONS row "Cross-ADT
  constructor tag collisions in unresolved bytecode switches", since the scrutinee's static type now
  names one ADT.
- **Re-exports:** AILANG has none (`MOD003`). Identity is the *defining* module, so a future re-export
  (Rust `pub use`, OCaml `type t = M.t`) preserves identity with no further change.
- **Aliases:** stay structural and transparent. Two record aliases `R = {x: int}` from different
  modules remain the same type (closed issue #904 was the record variant and is unaffected). An alias
  to an ADT resolves to the ADT's qualified TCon.

### 4.6 Interfaces, cache and packages

| Artifact | Change | Consequence |
|---|---|---|
| iface JSON `type` strings, stdlib goldens, registry `interface_hash` (`sha256:ifacev2:`) | **none** (TCon renders `Name`) | no republish for identity; `make freeze-stdlib` unchanged |
| iface JSON constructors/types | new `module` field on nominal type refs, excluded from `HashProjection` | consumers can see identity without the hash changing |
| internal cache digest (`iface/digest.go`) | renders qualified | an upstream switching `a.T`→`b.T` invalidates importers |
| compile cache key | `cacheKeyVersion` `v10` → `v11` (`pipeline/cache_key.go:50`) | every entry misses once |

Not covered by D3, and recorded as a follow-up rather than done: the hash cannot tell a package that
changes which `T` an export returns. Fixing that is an `ifacev3` identity change and every publisher on
an older binary would hit `PUB005` (the reason `hash_projection.go:11-15` pins the projection schema).

## 5. Alternatives considered

**B. Closure-wide name uniqueness (reject, don't distinguish).** Keep `TCon{Name}`; reject any module
whose import closure contains two exported nominal types with the same name. Cheaper (no representation
change, about 3 days) and, on today's corpus, it rejects the same zero programs (§8). It loses because:
(1) it rejects legal programs like `v15` and any consumer of two packages that both export `Event`
(three registry packages do today) even when they never mix; (2) a package's *private* dependencies
would decide whether its users compile, which is hostile to the registry; (3) it does nothing for d1,
the cache digest or SMT/REPL merges, so most of §4 is still needed. It is, however, strictly more
restrictive than A, so relaxing B to A later would be additive. **Fallback:** if the M0 spike shows A
cannot land in the window, ship B for v0.58.x and A in 1.x, and say so in the release notes.

**C. Runtime tag check.** Compare `ModulePath` in pattern matching so a/a′ fail loudly at run time. It
turns a wrong answer into a crash but leaves `check` lying; rejected as the fix, kept as an optional
defence (deferred decision).

**D. Qualified names in `Name` (`"q/a.T"`).** Breaks every `Name == "Json"` test in 126 sites and changes
every rendered signature, hence every `interface_hash`. Rejected.

## 6. Diagnostics

Codes follow the existing scheme: namespaced `TC_*_NNN` for checker errors
(`TC_TYPE_SHADOW_001`, `TC_ARITY_001`), `MODnnn` for module namespace errors. Each gets a page under
`docs/docs/reference/errors/`.

```
TC_NOMINAL_001: expected T (defined in q/b), but this value has type T (defined in q/a).
  They are different types that share a name.
  Fix: use the T from one module on both sides — import q/a (T) — or convert between them explicitly.

TC_UNBOUND_TYPE_001: type 'Nope' is not in scope (at v9/main.ail:2:23).
  Fix: declare it, or import it by name: import <module> (Nope).
  Types in scope here: Option, Result, W.

TC_AMBIGUOUS_TYPE_001: type 'Leg' is ambiguous here: q/a and q/b both export a type Leg and
  neither is imported by name. Fix: import q/a (Leg), or import q/b (Leg as BLeg).

MOD007: func 'f' is declared twice (at 2:1 and 3:1) — a module-scope name may have only one binding.
  Fix: remove or rename the duplicate.
MOD007: type 'T' is declared twice (at 2:1 and 3:1).
MOD007: constructor 'A' is declared by both type U (at 2:10) and type V (at 3:10).
MOD015: 'Leg' is imported twice (from q/a at 2:1 and from q/b at 3:1). Fix: import q/b (Leg as BLeg).
```

The `MOD007` registry description changes from "(let/func same name)" to "Duplicate module-scope
declaration"; v3's current message ("V's constructors are: " with an empty list) disappears because
the constructor table is qualified.

## 7. Conflict surface

**Positions touched:** `TCon.Equals` (every unification); `astTypeToInternalType` default and
`TypeApp` arms; `resolveSelectiveImports`/`resolveConstructorImport`; `ctorTypes`/`adtTypeParams`;
`iface.Builder.Build`; `computeDigest`; `RegisterDerivedEq`; SMT/REPL merges; bytecode ADT tables.

**What else lives there and must still work** (each becomes a fixture in M2/M3):

| Program | Why it is at risk |
|---|---|
| `examples/alias_body_closure/main.ail` | explicit `(Item)` must beat the auto-import channel (rule 3 before 5) |
| `examples/runnable/visited_set_bfs.ail` | `import std/set as S` + `import std/set (Set)`; `Set` wraps the builtin `Map` head |
| `examples/runnable/nested_match_simple.ail` | local `type Option` shadows the prelude's (rule 2 before 4) |
| `mk-main/src/eval/journal/scan.ail` | type `Interaction` and constructor `Interaction` from two modules |
| registry `crew_core` 0.6.0 `session.ail` | same, in a published package (`Applied`) |
| `stapledons-godot/sim/protocol_worlds.ail` | `Planet` plus `Planet as ObservedPlanet` |
| `docparse/tests/verify_attribution.ail` | writes `-> Json` without importing the name (rule 5) |
| `v15` repro | two same-named ADTs used disjointly, evaluator and VM |
| `std/net.ail` `urlEncodeForm(params: List[…])` | `List[...]` head must stay builtin |

**Disambiguation:** the six-rule order in §4.3 is today's de-facto precedence made explicit; the only
new outcomes are errors where today one candidate wins silently.

**What deliberately changes:** rows a–d5 of §1 become errors; a‴ (structural twins crossing a module
boundary) is rejected even though it runs today.

## 8. Blast radius, measured

Method: a regex sweep (`sweep.py`, `unbound2.py` in the session scratchpad; not the real parser),
resolving `module`/`import` lines, relative imports and `pkg/` imports against `ailang-packages`, the
`ailang-parse` checkout and the latest version of each of the 61 packages in this machine's local
registry cache (`~/.ailang/cache/registry/sunholo`, not the registry itself). Corpus: this repo
(`std`, `examples`, `benchmarks`, `tests`, `internal` fixtures; `.claude/` worktrees excluded), `daneel`,
`mk-main`, `ailang-packages`, `ailang-parse`, `docparse`, `email-parse`, `ailang-world`,
`stapledons-godot` (caches and builds excluded), `stapledons_voyage`: **2,073 files**, plus **561 files**
in the 61 cached packages. Import resolution after the package mapping: at least 99.9% in every
downstream root, 1,327 / 1,350 in this repo (the misses are experimental examples). Positive controls: the sweep
flags every §1 repro. Local checkouts can be behind their remotes.

| Class | Measured | Real after reading each hit |
|---|---|---|
| Two exported same-named ADTs in one import closure (a, a′, a‴) | 0 | **0** |
| Local ADT named like a reachable imported ADT (a″) | 2 (`nested_match_*` local `Option`) | **0 expected to break**: neither mixes values with `std/option`; confirm in M2 |
| Duplicate top-level funcs (d2) | 5 pairs in 4 files | **5, all real, all identical bodies** (one differs only in a binder name): `daneel ext/writer/register.ail` (`slen`, `cut`); `mk-main scripts/verify_mot136_dagr_producer.ail` (`sfield`), `packages/motoko-ext-herdr/types.ail` (`length_of`); `ailang-parse docparse/services/zip_extract.ail` (`readDocxStyles`) — **and every cached `ailang_parse` version from 0.39.2 to 0.55.5 (22)**, which `docparse` (pins 0.55.5) and `email-parse` (pins 0.50.0) depend on |
| Same type name explicitly imported twice (d3) | 8 | **0**: 4 are `as` renames, 4 are type-vs-constructor namespace pairs (legal; one is in the published `crew_core` 0.6.0) |
| Undeclared type names (d1), known set = §4.3 rules | 25 | **≈1**: `DOMEvent` in `std/dom.ail` (builtin, needs a home entry); the rest are `[T]` parameters, `List[..]`, fixtures in `tests/` that already fail, or a regex miss |
| Annotation sites relying on the auto-import channel (would break if D5 went the other way) | ≈500 in ≈60 files | `Json`, `XmlNode` written without importing the name (docparse, ailang-parse, packages, email-parse) |

**Summary:** the soundness change itself breaks no program found; the duplicate-func rule breaks one
published package that two flagship downstreams depend on, fixed by deleting one duplicated function and republishing.
**Not measured:** non-exported ADTs leaking through exported signatures; programs that match a builtin's
`Option` result against a local `Option`; anything outside these checkouts. The real number comes from
running the M2 binary through the downstream gate, which is an acceptance criterion below.

## 9. Milestones

Ordered so the riskiest unknown — how many places mint an unqualified nominal TCon — is measured first.

**M0 Spike: count the minting sites (1.5 d).** Add `TCon.Module`, stamp it in the elaborator and iface
builder only, treat `""` as a wildcard, and count wildcard matches by `(name, call site)` through
the test hook across `make test`, `make verify-examples` and the downstream gate.
*Accept:* a ranked list of sites; rows a, a′, a″, a‴ already rejected; `v15` passes; go/no-go on
Alternative B recorded in this doc.

**M1 Duplicates (0.5–1 d, independent of M0).** MOD007 func/type/constructor cases.
*Tests:* `v8`, `v12`, `v13` rejected with both positions; MOD007 let cases unchanged; a func defined
once in each of two modules is fine. *Mutation:* delete the func-vs-func loop, `TestMOD007_DuplicateFunc`
must fail. *Downstream:* PRs (by their owners) removing the five duplicates; `ailang_parse` republished
and the docparse/email-parse pins moved before this lands in a tag.

**M2 Nominal identity (3–4 d).** `Equals` on `(Module, Name)`; `nominalHome` table and its completeness
test; qualified `$adt` factory keys, `ctorTypes`, `adtTypeParams`, `ConstructorPattern.TypeName`, derived
Eq keys; `TC_NOMINAL_001`; qualified cache digest; `cacheKeyVersion` v11; wildcard removed.
*Tests:* every §1 a-row rejected with `TC_NOMINAL_001` naming both modules; `v15` on evaluator and
`--strict-bytecode`; two modules deriving Eq on `Row` both work; an upstream changing which `T` it
returns invalidates the importer's cache (`TestCacheKey_NominalOriginEditInvalidates`); every
`interface_hash` in `.stdlib-golden/` and of the 61 cached packages byte-identical before/after.
*Mutation:* revert `Equals` to name-only, the q repro test must fail.

**M3 One resolver (2 d).** `TypeScope` with the §4.3 order; `TC_UNBOUND_TYPE_001`;
`TC_AMBIGUOUS_TYPE_001`; MOD015 for import-vs-import types/constructors.
*Tests:* `v9`, `v10` rejected; every §7 fixture passes; auto-imported `Json` resolves; an `as`-renamed
type is unbound under its original name (#1485's second case). *Mutation:* make rule 5 return the
first candidate, the ambiguity test must fail; drop the unbound error, `v9` must pass `check` again
(proving the test sees the change).

**M4 Back ends and merges (1.5–2 d).** SMT and REPL/WASM keyed qualified; bytecode ADT tables keyed
qualified. *Tests:* a contract over `a.T` verified against a's constructors when `b.T` is loaded; a REPL
session importing both modules; the VM tag-collision fixture from the LIMITATIONS row.

**M5 Promise, docs, gate (1 d).** Stability clause (§10); error pages; teaching prompt; LIMITATIONS;
changelog. *Accept:* downstream gate green with moved pins; `make test`, `make verify-examples`,
`make freeze-stdlib` produces no diff.

Testing notes: `ailang check` checks only its first file argument, so multi-module tests use Go
pipeline tests or `check` on the entry module; every new error code gets a positive and a negative test.

## 10. The soundness clause for `stability.md`

Proposed text, to go under "What Stable means in 1.x":

> **Soundness fixes are not breaking changes.** A Stable program is one the documented rules accept. If
> `ailang check` accepts a program those rules reject, that is a bug, whether it shows as a run-time type
> error, a name that resolves to the wrong declaration, or an effect the checker missed. Fixing it may
> make a previously accepted program fail to check, and the fix may ship in any 1.x release. Such a fix:
> - only rejects programs; it never changes what an accepted program computes;
> - is listed in the CHANGELOG under `Fixed — … (soundness)`, with its error code and the migration;
> - is run against the downstream gate first, and the number of affected programs is published;
> - ships as a warning for at least one minor before it becomes an error, where that is feasible —
>   but immediately as an error when the hole lets a program reach behaviour its types forbid.
>
> Experimental items need no carve-out; they may change in a minor anyway. Internal items carry no promise.

Sources adapted: Rust's 1.0 promise ("we reserve the right to fix compiler bugs, patch safety holes, and
change type inference in ways that may occasionally require new type annotations") and RFC 1122's
process (crater run, future-incompatibility warning where possible); Go 1's carve-out for bugs that
violate the specification ("a program that depends on the buggy behavior may break if the bug is
fixed"). Quoted from memory, not re-fetched. AILANG's version is narrower than Rust's: it does not
reserve inference changes, only rejections of programs the rules already forbid.

## 11. Migration

**What a downstream author changes:** delete duplicate definitions (MOD007); import the type you use
by name when two direct dependencies export it (`TC_AMBIGUOUS_TYPE_001`); stop passing one module's `T`
to another module's `T` (`TC_NOMINAL_001`) — convert explicitly or use one type; declare or import
typo'd type names (`TC_UNBOUND_TYPE_001`). No source change is needed for programs that only use
same-named types disjointly. No republish is needed for identity; `ailang_parse` needs one for d2.

**Changelog fragment draft** (`changelogs/unreleased/YYYY-MM-DD-nominal-type-identity.md`):

```markdown
### Fixed — Types are identified by name and module; two `T`s from different modules no longer unify (soundness, BREAKING)

`check` accepted programs that passed one module's `type T` where another module's `type T` was
expected, and they then failed at run time (`_str_upper: arg 0 - expected string, got int`). A
nominal type is now its defining module plus its name. Mixing them is `TC_NOMINAL_001`; an
undeclared type name is `TC_UNBOUND_TYPE_001` (#1485); a bare name two dependencies export is
`TC_AMBIGUOUS_TYPE_001`; the same type imported twice is `MOD015` (#1980); a func, type or
constructor declared twice in one module is `MOD007` (#1278). Interface hashes are unchanged.
Migration: design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md §11.
```

**Docs:** `LIMITATIONS.md` — delete "Same-named types across imports" (lines 139-148) and narrow the
bytecode tag-collision row; the canonical `docs/docs/reference/limitations.md` has no entry today, so
nothing to remove there. Teaching prompt "Module Import Rules": add two lines — a type belongs to the
module that declares it, and import a type by name if two imports export it — and remove nothing (the
prompt never taught the unsound behaviour). Version the prompt through prompt-manager. Error pages for
the three new TC codes; update `mod007.md` and `mod015.md`. `stability.md`: §10 clause; drop the
bare-name note if one is added meanwhile.

## 12. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| A minting site left unqualified → false rejection | High in M2 | M0 counter; debug-build panic; `nominalHome` completeness test; downstream gate |
| Cache serves a pre-change verdict | Low | `cacheKeyVersion` v11 |
| `interface_hash` drifts and publishing breaks (`PUB005`) | Med | M2 byte-identity test over goldens and 61 packages |
| Interacts with #2146 exhaustiveness work landing in parallel | Med | exhaustiveness reads constructor sets by qualified ADT; land M2 first or agree the key |
| Window slips | Med | Alternative B fallback named in M0 |

**Non-goals:** module-qualified type *syntax* (`S.Set[int]`, the LIMITATIONS parser papercut — additive,
1.x); re-export syntax; changing alias structural equality; removing the auto-import channels;
`ifacev3`; Go codegen correctness beyond name mangling.

## 13. Axiom compliance

A1 +1 (verdicts no longer depend on import order); A2 0; A3 0; A4 +1 (a name means what its module
says, not what is nearby); A5 +1 (a module's typing depends only on its own declarations and imports);
A6 0; A7 +1 (a coded error at the cause instead of a distant run-time crash); A8 0 (no syntax); A9 0;
A10 +1 (two packages can each export `Event`); A11 +1 (four coded errors replace silent behaviour);
A12 +1 (identity checked at the module boundary). **Net +7.** No hard violation on A1/A3/A4/A7.

## 14. Decisions for Mark

1. **Qualified identity (A) over closure-wide uniqueness (B)?** Recommend A, with B as the named
   fallback if M0 says A will not fit the window.
2. **Duplicates in dependency code: error, or warning for `pkg/` modules during 0.58.x?** Recommend
   error everywhere and republish `ailang_parse` (delete one duplicated function) before the tag; a capped-lint
   mode would be a second promise to keep.
3. **MOD015 or MOD016 for import-vs-import?** Recommend MOD015; the registry has promised it since v0.52.0.
4. **Keep `interface_hash` on display names (no republish), with identity in a non-hashed field?**
   Recommend yes; `ifacev3` is a 1.x follow-up.
5. **Adopt the §10 clause, including warn-first-where-feasible?** Recommend yes.
6. **Fold the #1485 and #1980 type-namespace docs into this one?** Recommend yes; the value-namespace
   half of #1980 stays in its own doc.

## 15. Findings that differ from the review

- Row a is not new: M4 of m-type-name-shadow-and-cache (2026-09-26, audit row 8) recorded it, at P2.
- The review lists three silent acceptances; there are five (d4 duplicate types, d5 duplicate
  constructors), plus a″, which `LIMITATIONS.md` describes as loud.
- #1485 and #1980 already have planned docs, and #1980 has two that disagree on the error code.
- The measured break from the soundness fix is zero; the measured break from #1278 falls on a published
  package two flagship downstreams pin. The review's "rejects programs that pass today" is true for
  d2, not (on this corpus) for a.
