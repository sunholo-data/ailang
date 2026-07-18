# M-DIAG-PRIMITIVE-FIELD-SUGGESTIONS: Symbol-Specific Enrichment of the Primitive-Field-Access Diagnostic

**Status**: BACKLOG (extension-lane stub) — severed from M-PROMPT-FOOTGUNS on 2026-07-18
**Lane**: Extension (per PROGRAM.md "default bias: if it can be an extension, it is an extension")
**Priority**: P3 (Low — ~2% of recent compile failures)
**Target**: TBD (post-v0.30.0)
**Dependencies**: designing a diagnostic-enrichment point OUTSIDE `internal/types` (see below)

## Provenance

This doc captures the **Part C / Phase 3** that was severed from
[`m-prompt-footguns-to-diagnostics.md`](m-prompt-footguns-to-diagnostics.md) when Mark ratified the
PARK-NOTE recommendation on 2026-07-18 (ship the unanimously-accepted Part A + Part B; drop Part C to
the extension lane). The severance had two independent triggers, both recorded in that doc:

1. **gpt5-6-sol's blocking objection (Part C only):** the primitive-detection premise (matching
   `TCon` names `string`/`int`/`float`/`bool`) is unverified against user ADTs/aliases with
   primitive-like names; the reviewer's own remedy was "defer Phase 3".
2. **Frozen-core routing (PROGRAM.md):** the symbol-specific repair (`.split` →
   `split(receiver, ...)` + `import std/string (split)`) requires a std-symbol → call-form catalog.
   Hosting that catalog inside `internal/types` would couple the type-system core to library policy
   and grow with the stdlib — a frozen-core violation. It belongs to the extension lane.

## Problem

`content.split("\n")` (Python/JS-style dot-notation for string methods) parses as record field
access and fails in the unifier with the raw, unhelpful:

```
type unification failed at [field access at file.ail:5:22]:
cannot unify type constructor string with *types.TRecordOpen
```

No hint that primitives have no fields, that AILANG uses functions rather than methods, or that the
concrete fix is `split(content, "\n")`.

## Scope of THIS backlog item (the enrichment ONLY)

The **generic, library-agnostic** half of the diagnostic (a coded
`TC_PRIMITIVE_FIELD_ACCESS_001` stating "type 'string' has no fields — AILANG uses functions rather
than methods; tried to access '.split' at <pos>") is the CORE half and is **not** owned here — it was
part of the dropped Phase 3 and can be shipped independently in `internal/types` when re-scoped
(it carries only the receiver-primitive name, requested field, and source path — all recoverable
from the types/constraint already at the unifier, mirroring the v0.20.0 tagged-union precedent).

THIS doc owns only the **symbol-specific enrichment**: turning that generic error into a
call-form-carrying repair (`.split` → `split(receiver, ...)` + the matching `import std/string
(split)`), populated from verified stdlib metadata, and extending to std/list method-shapes
(`xs.map(f)` → `map(f, xs)`).

## The blocker: no enrichment hook exists at HEAD

A live audit at HEAD (M-PROMPT-FOOTGUNS verification rows 21–24) found **no existing
post-unification enrichment seam** to attach a stdlib suggestion catalog to, without adding new core
surface:

- `swapTraps` / `DetectArgOrderWarnings` (`internal/pipeline/warn_split_args.go`) is a
  **non-blocking warning** pass over successfully-elaborated Core. A unification FAILURE aborts
  compilation before it runs, and it never sees error values — it cannot decorate a type error.
- Type errors cross the types→pipeline boundary as **unstructured `fmt.Errorf` chains** (even the
  tagged-union error is `fmt.Errorf` + a `[record_access_on_tagged_union]` string tag). There is
  **zero** `errors.As`/`errors.Is` on type errors anywhere in `internal/pipeline` — no decoration
  seam to reuse.
- `internal/diag` is a footgun coverage **table + CI fixture contract**, not a runtime enrichment
  mechanism.
- The #327 `SetModuleFuncNames`/`localResolutionHint` pattern injects *program facts* pre-inference;
  reusing its shape for a stdlib catalog would still require a NEW types-side setter + formatter =
  new core surface.

## Candidate enrichment routes (to be decided when this is scheduled)

Both recorded verbatim from the M-PROMPT-FOOTGUNS "Frozen-Core Routing" section; neither chosen here:

- **Route (a) — structured error + `errors.As` decorator.** Give the unifier's
  primitive-field-access failure a structured error TYPE (not `fmt.Errorf`), and add an
  `errors.As`-based decorator in BOTH pipeline paths (`pipeline_single.go`, `pipeline_module.go`)
  that, when it recognizes the structured error, looks up `field` in a stdlib-metadata catalog
  (hosted in the pipeline/extension layer, NOT `internal/types`) and appends the call-form +
  import suggestion.
- **Route (b) — #327-style data-injection setter.** A pipeline-owned setter feeds a
  field→call-form table into the checker pre-inference; the checker consults it only to enrich the
  message. (Downside: still touches `internal/types` surface — weaker fit for frozen-core.)

Route (a) is the current front-runner (keeps `internal/types` library-agnostic) but the structured
error type is a small design of its own — hence this is a backlog item, not a Phase.

## Non-Goals

- The generic core diagnostic itself (severed-Phase-3 core half; re-scope separately).
- Any stdlib symbol catalog or import advice inside `internal/types` (frozen-core violation).
- Dot-notation method-call SYNTAX support (contradicts A8/minimal-syntax; AILANG uses functions).

## Related

- [`m-prompt-footguns-to-diagnostics.md`](m-prompt-footguns-to-diagnostics.md) — parent doc; see its
  "Frozen-Core Routing (Part C)" and "Future Work" sections and verification rows 20–24.
- M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0) — the unifier prescriptive-error precedent the core
  half would extend.
- m-dx-split-argument-warning (`swapTraps`) — the pipeline-level trap-table shape the enrichment
  catalog should mirror (it is why the catalog does NOT belong in `internal/types`).

---

**Document created**: 2026-07-18 (severed from M-PROMPT-FOOTGUNS Part C at sprint execution time)
