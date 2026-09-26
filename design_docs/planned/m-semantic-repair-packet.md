# M-SEMANTIC-REPAIR-PACKET: Bounded, Identity-Bound Context for Code Repair

**Status:** Proposed design — not implementation-approved; quorum not run.
**Date / author:** 2026-09-05 / Astra, requested by Mark.
**Target:** Experimental tooling on the v1.x path; not a new v1.0 release gate.
**Priority:** P1 candidate.
**Estimated:** 4–6 engineering days, provisional until sprint planning; model experiments separate.
**Parent:** [Semantic context R2–R6](v0_29_0/m-ailang-semantic-context.md).
**Traces to:** [Astra vision](m-astra-vision.md), [PROGRAM](../PROGRAM.md).

## Problem and coverage decision

The semantic-context parent proposes typed context, semantic diffs and effect deltas. This child specifies ONE deliverable from that menu: a portable read-only repair packet, usable by a resident extension without importing World's state model into the compiler. It does not independently implement the parent's context engine, semantic cache or arbitrary AST slicing.

The reviewed `internal/iface/iface.go` and `json.go` expose typed exports, constructors, aliases and effects. `cmd/ailang/check_output.go` exposes diagnostic code, message, location and suggestion. Those are useful ingredients, but a consumer needs to know which source revision they describe, which facts were unavailable, and what it may safely edit. This design does not assert that any particular benchmark failure was caused by missing context; that hypothesis is tested by the [lifecycle pilot](m-lifecycle-eval-pilot.md).

## Goals and non-goals

Deliver a bounded machine-readable context artifact that is honest about provenance and completeness. It must support a compile-failing candidate without representing old inferred types as facts about that candidate.

Non-goals: automatic patch application; proving a natural-language specification; new syntax, typed holes or type inference rules; arbitrary incomplete-program semantics; World goals/approvals; a new whole-repository call-graph engine; model-specific heuristics in the compiler.

## Proposed interface

Proposed new read-only CLI family: `ailang repair-context --manifest <file> --json` (name subject to CLI review). The manifest supplies an allowed workspace root, a candidate module, optional named target export, optional last-known-good source snapshot, bounded dependency files and the evidence to include. Explicitly identify it as untrusted caller input.

Initial limits proposed for review: 256 files, 1 MiB total input text, 128 KiB output, 64 diagnostics and a 30-second wall-clock backstop. These are ceilings, not performance promises. Requesting a smaller limit is allowed; increasing the production maximum is a config decision, not something an input file can grant itself.

| Packet component | Contract |
|---|---|
| `schema`, packet identity | Versioned `ailang/repair-context/v1`; hash of a canonical payload excluding its own ID; ordered arrays, no wall-clock generation timestamp in identity |
| Candidate snapshot | Relative paths and raw-byte SHA-256 for every captured source/dependency; compiler executable digest; command options; dependency lock identity when applicable |
| Target | Module/export identity or explicit unresolved reason; file-wide editing is a permitted first-version fallback, explicitly labeled |
| Current diagnostics | Existing structured representation, original check outcome and compiler identity; missing codes/spans remain unavailable |
| Prior facts | Optional typed interface from a checked baseline, labeled with BASELINE identity and role; never substituted into current facts |
| Context | Full candidate module first; selected dependency interface/source objects with identity and origin; deterministic ordering |
| Completeness | Capture status, included/omitted paths, omission reasons, byte counts and remaining retrieval references; no claim of minimal sufficient context |
| Edit request | Caller-requested files/exports and desired obligation; advisory instructions, not authorization or proof |
| Evidence | References to check/test reports and their subject identity; arbitrary attached text stays untrusted |

Results distinguish `ready`, `incomplete` and `unavailable` as packet states; exit success means the packet was produced, never that the candidate passed checking. Compiler outcome is a separate required field. Malformed manifests, path escapes and inconsistent snapshots return a structured refusal and nonzero exit. Named labels are packet-local states, not newly allocated compiler error codes.

## Snapshot and compilation rules

1. Resolve and validate paths beneath the allowed root, reject symlink escapes, duplicate canonical paths and non-regular sources. The caller cannot broaden the permitted root through the manifest.
2. Capture source bytes and hashes before analysis. Compile against those captured bytes through an isolated source provider or temporary snapshot. The snapshot must include all dependency bytes actually read. If the current compiler cannot expose the required reads safely, first version accepts only an explicit local-module closure and reports unsupported package resolution. Never infer closure completeness from the manifest alone.
3. Use existing check/iface output. A candidate that fails type checking has diagnostics; it may have no current interface. A checked baseline supplies separately labeled expectations. No inferred type recovery is introduced by this sprint.
4. Selection uses explicit targets and deterministic order. Do not prune inside function bodies or claim higher-order dynamic dependencies are solved. If required context exceeds the output budget, emit `incomplete` with omission identities; consumers may retrieve more or choose full-context repair.
5. The packet is a snapshot, not a lease on mutable files. Before any patch application, the resident extension checks the current source/dependency identities against the packet. Any mismatch requires a new packet. Patch application is outside this command.

Interface identity alone is insufficient for snapshot freshness: implementation bodies can change without changing public types. `internal/iface/hash_projection.go` also deliberately excludes alias bodies from that specific hash projection. Use source-byte identities in addition to interface representations; do not repurpose registry compatibility hashes as source hashes.

## Example workflow

A module type-checks at S0. A requested edit produces S1 with a type error. The packet contains S1 diagnostics, S0's target interface clearly labeled prior context, and the captured dependency identities. A small model proposes a patch. The resident checks that S1 is still current, applies only its permitted patch through existing tools, then runs the full required checks. If the public contract must change, that is returned as a proposal to the requirement owner rather than silently weakening the check.

No AILANG snippets are introduced by this design; schema and CLI spellings above are proposed surfaces.

## Implementation plan and ownership

| Phase | Responsibility / likely files | Approximate scope |
|---|---|---|
| M1 | New `internal/repaircontext/`: manifest, canonical packet, identity and bounded capture | 350–550 production LOC plus tests |
| M2 | Adapter to pipeline/check/iface; new CLI entry and `cmd/ailang/help.go` | 250–400 LOC plus fixtures |
| M3 | Resident-facing usage guide, deterministic fixtures, A/B handoff | 100–200 LOC/docs; no model calls required |

The source-provider hook is a design-freeze dependency: inspect the pipeline resolver before planning M2 and choose the existing injection seam or the explicit local-closure restriction. New compiler behavior must route separately if neither is feasible. A World consumer goes through its extension/host boundary, never direct imports from dashboard packages into compiler internals.

## Acceptance criteria and tests

| ID | Observable acceptance | Failure control |
|---|---|---|
| RP1 | Same captured inputs/config produce byte-identical packet twice, including randomized map insertion order | Inject ordering dependence; identity test fails |
| RP2 | Change dependency BODY without changing exports: snapshot identity changes; stale application precondition rejects | Omit dependency source hash; control becomes falsely acceptable |
| RP3 | Broken candidate + valid baseline: candidate error remains visible; baseline facts retain their own identity | Relabel prior facts as current; schema/consumer test rejects |
| RP4 | Input/output limits exercised just below and above; output reports omissions, malformed/escaping paths refuse | Remove limit/path check; over-limit/escape test fails |
| RP5 | Existing iface constructor/alias/effect information preserved; unknown fields are not invented | Drop constructor fields or effect annotation; fixture comparison fails |
| RP6 | No AI/effect handler executes while constructing a packet | Fake handler counter stays zero with a known callable positive control elsewhere |
| RP7 | Concurrent file edit cannot produce a packet mixing compilation of one byte sequence with hashes of another | Race-controlled snapshot mutation fixture refuses or uses the captured version consistently |

Use `internal/iface/compact_adt_fields_test.go` (record and positional constructor tests inspected) and `nested_effects_test.go` (nested callback effect restoration inspected) as regression neighbors, not assumptions about a whole-program proof. Run focused package tests, CLI integration tests, `make check-boundaries`, and required repository checks in the eventual sprint. No implementation tests were added or run for this document.

## High-impact decisions / design freeze

| Decision | Proposed choice | Owner / deadline |
|---|---|---|
| Smallest useful scope | Module context, not expression-hole inference | Mark at design approval |
| Snapshot compilation seam | Existing resolver injection, otherwise explicit local closure | Designer before sprint planning |
| Public CLI/schema and limits | As above, after CLI convention review | Mark at design approval |
| Parent scope ownership | This child owns packet assembly only; parent retains R3–R6 beyond this packet | Mark at design approval |

- [ ] Design approved; source-provider mechanism verified.
- [ ] Manifest/schema and limits frozen; dependency-resolution restrictions documented.
- [ ] No new semantics slipped into the adapter.

## Risks and deferred work

Large closures may make packets no better than full context; measure rather than infer utility from small byte counts. Diagnostics and source comments remain untrusted model inputs. Do not include credentials or arbitrary environment dumps. Fine-grained slices, automatic semantic patches and type-constrained decoding are deferred until the module-scale packet produces useful evidence.

## Verification log / related work

- Read `internal/iface/{iface,json,hash_projection}.go`: available interface facts and alias exclusion recorded above.
- Read `cmd/ailang/{check,check_output}.go`: JSON check outcome and existing iface rendering; no claim of recovery from failed inference.
- Read named iface test bodies, not only filenames.
- Related-doc search: scaffold SimHash/neural search attempted; earlier direct neural control in this session family reported fallback-simhash, so scores are not treated as semantic duplicate evidence. Manual parent/topic review establishes this as a bounded child, not a replacement.
- [Semantic context](v0_29_0/m-ailang-semantic-context.md), [Astra/World allocation](m-astra-vision.md#ailang-world-review--routing-correction-2026-09-05), [lifecycle pilot](m-lifecycle-eval-pilot.md).


## Axiom compliance

Directional design assessment, not implementation approval. No hard violation proposed on A1/A3/A4/A7.

| Axiom | Score | Design constraint |
|---|---:|---|
| A1 Determinism | +1 | Identity-bound inputs; deterministic mechanical results |
| A2 Replayability | +1 | Preserve inputs and outcomes needed to reproduce the decision |
| A3 Effect legibility | 0 | Existing effect semantics unchanged |
| A4 Explicit authority | +1 | Metadata and model output never mint execution authority |
| A5 Bounded verification | +1 | Explicit limits and checks with named refusal outcomes |
| A6 Safe concurrency | 0 | Sequential first version; snapshot identity checked |
| A7 Machines first | +1 | Structured, versioned artifacts with explicit unavailable states |
| A8 Minimal syntax | +1 | No new language syntax |
| A9 Cost visibility | +1 | Record tool/model costs and failure overhead |
| A10 Composability | +1 | Reuse existing compiler, evidence and protocol boundaries |
| A11 Structured failure | +1 | Unknown, incomplete and stale cannot masquerade as success |
| A12 System boundary | +1 | Separate claims, verification, permission and action |

**Net +10.** Re-score if implementation scope changes.
