# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Iteration 261 · 2026-08-23 · dev green (16 checks, 0 not-green @ `98ec079ca`)**

## In flight
- **`#764` protocol-only `serveapi` module — UNPARKED AND PLANNED.** Mark answered `D-35` with
  **(a) PLAIN PACKAGE** (19:01Z on `#745`). Design doc frozen on that ruling the same iteration
  (`9ce91ce50`); sprint plan + JSON landed (`1db206fe5`): 4 milestones, 875 est. LOC, 21 baseline
  rows taken on the pristine tree. **Next iteration executes M1.**
- Planner found **7 discrepancies** in the doc. One is blocking and was confirmed first-party:
  the round-2 reviewer's *verbatim* CI-self-test fix (`zz_intruder_test.go`) is **vacuous** —
  `go list -deps` does not enumerate test-only imports (measured 1 dep / 0 hits vs control
  `-test` 209 / 1). Plan uses a non-test intruder plus an arm that pins the blind spot.

## Next
- **262: execute `M1_EXTRACT_PROTOCOL_PACKAGE`.** Then M2–M4, then the `#764` reply.
- `D-34` pre-authorizes the **v0.34.0 release ask** once `#764` is green on `dev` — that tag is
  the delivery Ailang World actually consumes (it pins releases, so merging alone does not reach it).

## Parked on Mark (3 open decisions)
- **`D-30`** — harness↔`ai-check` version coupling before the `not_applicable` split: (a) versioned
  JSON schema, (b) bind to `os.Executable()`, (c) accept the residual. Skew is live on this rig.
- **`D-31`** — split the designer rotation into authoring vs review lanes, or widen it: 2 of 3
  entries cannot author for structural reasons, so it collapses onto Fable every time (instance 7).
- **`D-32`** — should an `inconclusive` verification obligation be exempted from the KPI arm, as
  your `D-29` ruling exempts `not_applicable`?

## Loop health
- Routing: controller opus · planner **opus** (`fail-closed:planner-lane-field-missing`) · designer
  rotation collapsed on Fable (see `D-31`) · executor codex · evaluator sonnet.
- **Running-skill drift fired and was REPAIRED** under the ratified `D-16`: main checkout was 1
  behind, so the loop was executing a rulebook missing one commit. ff-only, controls measured.
- Two instrument findings recorded at instance 1 (below the ≥2 bar for a skill edit): quorum
  artifacts are written to a **CWD-relative, gitignored** path, so Gate 2's own quorum check reads
  zero for a four-times-reviewed doc; and a *verbatim* reviewer fix proved wrong under the carve-out.
- Cost: metered **$0.00** of $5 this iteration. Quota: opus ×2.
