# M-PKG-SMOKE-STAGED-LOCK — accepted, unreleased evidence

Root accepted independent round2 **PASS95/100** and normally merged
[PR2013](https://github.com/sunholo-data/ailang/pull/2013) at
`96d69f05b6e84bd6666e8fa91ff948f803b67d92` on2026-10-29 at20:53:24UTC.
Issue1773 closed20:53:25UTC. Target **v0.56.0 is UNRELEASED**; no tag/version bump
is implied. [Parent](../../m-pkg-smoke-staged-lock.md),
[plan](../../m-pkg-smoke-staged-lock-sprint-plan.md) and
[validation map](../../m-pkg-smoke-staged-lock-validation.md) are archived together.

## Frozen evaluations

|Round|Exact evaluated head|Verdict|Durable report|
|---|---|---|---|
|1|0853c9fc15f3daf5f9e304e6e451e78c21fd899c|**FAIL75/100**, aggregate CI hard failure|[JSON](round-1/eval_M-PKG-SMOKE-STAGED-LOCK_round_1.json), [Markdown](round-1/eval_M-PKG-SMOKE-STAGED-LOCK_round_1.md)|
|2|165f87a12ceb0e20c04216c472004c11685e417b|**PASS95/100**, all27actual contexts completed20SUCCESS/7SKIPPED|[JSON](round-2/eval_M-PKG-SMOKE-STAGED-LOCK_round_2.json), [Markdown](round-2/eval_M-PKG-SMOKE-STAGED-LOCK_round_2.md)|

Both report pairs remain **verbatim frozen bytes**. The47logical evidence
artifacts are stored as44verbatim files and3transparent lossless base64 JSON
wrappers; no original evidence data is normalized or rewritten.
[Manifest](manifest.json) records original local paths, byte counts and SHA256s.
Then-current pending/planned paths, absolute `/tmp` paths and recommendations in
the frozen reports remain historical, not assertions about current repository
layout or unresolved acceptance. This README and the live sprint JSON provide
the final disposition. No verdict or failure identity was renamed/reclassified.

Round2 independently verified all33owned paths: all21Go/AIL plus every static
asset/TOML fixture equal0853, only3own metadata updates. Accepted565incoming
changes were checked separately; its runner test is the sole global Go/AIL delta.
Actual new-head CI supplies broad proof, not a whole-binary equivalence claim.
The [21-row map](round-2/acceptance-matrix-final.json),
[source identity](round-2/source-identity.json),
[AST inventory](round-2/functions-over50.json),
[automated CI evaluator](round-2/automated-evaluation.json) and
[final check snapshot](round-2/pr2013-final.json) preserve that distinction.
The sole deduction is the literal>50line function cap−5, including tests; base100,
no compiler/IR/performance bonus. No new archival-time tests were run.

## Supporting proof and path mapping

- `round-1/` retains its source/acceptance/function inventories, local-log manifest,
  original failed CI snapshot/classification/attempt records and signal-only
  independent control README+RED/GREEN logs. The extracted control deliberately
  stubbed staging; its README does not claim M1 proof. Actual full-stage owner
  signal RED/GREEN is separately retained below.
- `owner-local/` preserves all15small original owner logs identified by the frozen
  round1 local-evidence manifest: corrected M1 RED/checkpoint, the earlier fixture
  syntax failure, M2 diagnostic/signal RED/GREEN, M3 warning RED/GREEN, strict
  registry-byte CLI discriminator, human/JSON and final scoped/core/structure proof.
  The manifest's original bare basenames map to this directory;12are verbatim
  and3are losslessly encoded as described below. All decoded bytes/hashes remain
  identical to the originals.
- `accepted-integration/` contains original cc2build/focused/identity proof after
  normal integration of accepted565. Its actual selective package/CLI/runner run
  is distinct from the earlier37cfull changed/core proof and new165broadCI.
- `round-2/` contains the current frozen reports and all referenced portable
  relative JSON/log companions. Parser1machine/20prose criteria were resolved by
  the independent complete source/test/log/CI map, not generator `passes` flags.

The original report's absolute `/tmp/ailang-backlog-20261029/...` paths identify
session-local originals. They are not portable links. Durable copies are mapped
above and in the manifest; original source revisions/URLs remain available in
Git/GitHub. Historical issue-triage snapshots elsewhere in the repo may retain
then-current `planned/` paths and owner/status observations; those snapshots were
not rewritten into claims about the final state.

## Lossless wrappers for three original RED logs

Only `1773-M1-fixture-red.log`, `1773-M1-red.log` and `1773-M3-red.log`
retain original trailing whitespace in their frozen data. Their stored filenames
are the logical name plus `.base64.json`; the plain `.log` files are not also
committed. Each wrapper declares the original logical filename/path, original
byte count and SHA256, encoding `base64`, and its payload. The manifest records
logical identities plus the stored wrapper identities. This keeps full
`git diff --check` clean without normalizing evidence or changing any policy.

The following **Python3 standard-library** code is portable across operating
systems. Run it from this evidence directory in a Python3 interpreter, choosing
the wrapper and output paths; no repository product/tool script is added:

```python
import base64
import hashlib
import json
from pathlib import Path

wrapper = json.loads(
    Path("owner-local/1773-M1-red.log.base64.json").read_text()
)
assert wrapper["encoding"] == "base64"
data = base64.b64decode(wrapper["payload"], validate=True)
assert len(data) == wrapper["original_bytes"]
assert hashlib.sha256(data).hexdigest() == wrapper["original_sha256"]
Path("recovered-1773-M1-red.log").write_bytes(data)
```

All47logical identities, including the three decoded wrappers, were checked
against the original local evidence bytes and hashes during archival. Both
frozen evaluation report pairs remain byte-identical to the originals.

## Two actual failed runs, retained as failures

The 4.59MB raw logs are intentionally **not committed**. Their exact run/job URLs,
local original paths, byte counts and hashes are frozen in
[the failure manifest](round-1/ci-two-failures-manifest.json),
[round1 report](round-1/eval_M-PKG-SMOKE-STAGED-LOCK_round_1.json) and
[this archive manifest](manifest.json). Raw originals were hash-verified again
by round2 before archival. They may remain accessible only in the original
workspace or GitHub retention window; this archive does not claim otherwise.

|Attempt|Run/job|Bytes|SHA256|
|---|---|---:|---|
|1|[40883138509 /123479693310](https://github.com/sunholo-data/ailang/actions/runs/40883138509/job/123479693310)|4590430|fda09a3970adb9537caf4e2adfd5bfd1782164bb646aa49c1faf974709915e08|
|2|[40883138509 /123488317858](https://github.com/sunholo-data/ailang/actions/runs/40883138509/job/123488317858)|4594234|5e5bd48a695725c52577d83463752c9de79563adc8388f855760844b443f9f60|

Both failed the unchanged worker `eval/sigterm` fixture's empty PID read. The
later separately accepted fixture/new-head CI cleared the current gate and did
not retroactively turn either old execution into a success. No workflow retry,
operator/card action, live publication or merge occurred during this archival.
