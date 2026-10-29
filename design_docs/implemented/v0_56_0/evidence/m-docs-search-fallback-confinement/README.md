# #1965 — root-accepted documentation fallback confinement evidence

Root independently accepted [PR2017](https://github.com/sunholo-data/ailang/pull/2017)
**PASS95/100** at `50ca373aeacd6b74ffce573d1892cf2dd4211e32`. Root normally
merged `1cfaedbf2170e5c3260c5207c7c82823bd51032f` at21:16:33UTC on2026-10-29;
issue1965 closed21:16:34UTC. Actual final27CI contexts are20SUCCESS/7SKIPPED,
zero failed/cancelled/pending. Target **v0.56.0 is UNRELEASED**; no version/tag bump.

The [parent](../../m-docs-search-fallback-confinement.md),
[plan](../../m-docs-search-fallback-confinement-sprint-plan.md) and
[validation map](../../m-docs-search-fallback-confinement-validation.md) travel
together. This archival was performed by the implementation author **after
root's separate independent acceptance**. It adds no self-evaluation or new
product/test execution, and does not alter the frozen root judgment.

## Frozen independent judgment

- [Root report JSON](independent/eval_M-DOCS-SEARCH-FALLBACK-CONFINEMENT_round_1.json)
  and [Markdown](independent/eval_M-DOCS-SEARCH-FALLBACK-CONFINEMENT_round_1.md)
  remain byte-identical to the original independent reports.
- [All21criteria](independent/acceptance-matrix-final.json),
  [AST inventory](independent/functions-over50.json),
  [actual final CI snapshot](independent/pr2017-final.json) and
  [CI-backed automation](independent/automated-evaluation.json) preserve root's
  concrete proof. The physical function-length cap−5 is the sole deduction;
  no compiler/IR/performance bonus or invented language example was required.
- Root independently verified18source and27raw-log identities, guard order,
  real library/CLI/tool children, active helper/proxy positives and full state
  snapshots. Snapshots show no writes; before-read refusal also rests on actual
  guard order, not a fabricated cache-read spy. Explicit neural/embedding and
  cache-management behavior remains separately scoped.

## Executor originals and historical qualifications

`executor/` contains the frozen final handoff and its27original log identities,
including real backend/CLI/tool REDs, preliminary macOS path-spelling comparison,
actual generated-reference drift failure, corrected generated-page gate,
full affected/core proof, final test-only atomic counter checks and local
structure/build evidence. No failure was rewritten as a success. The original
reference failure was repaired through the authoritative generator and rerun.

The preserved counterfactual `executor/manifest.json` and `overlay.json` identify
only three exact pre-guard files restored from dev6d0d3a9da in a scratch build;
they do not claim whole-old-binary equivalence. Source/preimage revisions and
hashes remain reconstructable through Git; the local scratch binary is not
committed or claimed portable. Earlier parent approval/pending-gate paths in
frozen records are historical to their checkpoint, not the current archive state.
Original absolute `/tmp/...` references identify session originals; their durable
logical copies are mapped by [the archive manifest](manifest.json).

## Lossless storage and portable decode

All43logical artifacts retain exact original bytes/hash: **40verbatim files and
3transparent lossless base64 JSON wrappers**. Only whitespace-bearing originals
`1965-M2-red.log`, `1965-M3-red.log` and `1965-M4-test-changed.log` use the logical
filename plus `.base64.json`. Every wrapper declares logical filename/path,
original bytes/SHA256, encoding and payload; manifest entries also authenticate
the stored wrapper. No whitespace policy, exclusion or evidence normalization
was introduced. Both independent report files remain plain/verbatim.

This Python3 standard-library code is portable across operating systems. Run
from this evidence directory, choosing the input/output paths. No product or
tool script is added:

```python
import base64
import hashlib
import json
from pathlib import Path

wrapper = json.loads(
    Path("executor/1965-M4-test-changed.log.base64.json").read_text()
)
assert wrapper["encoding"] == "base64"
data = base64.b64decode(wrapper["payload"], validate=True)
assert len(data) == wrapper["original_bytes"]
assert hashlib.sha256(data).hexdigest() == wrapper["original_sha256"]
Path("recovered-1965-M4-test-changed.log").write_bytes(data)
```

All43logical identities, including decoded wrappers, were checked against
original local bytes and hashes. This records preservation, not a new verdict.
