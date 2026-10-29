# Accepted M0 repair evidence — v0.56.0 UNRELEASED

Root independently accepted **PASS 95/100, round 3**, at
`4fb3b2b7efdbb77f7a6a7d77f92c1ba74652a314`.
[PR #2019](https://github.com/sunholo-data/ailang/pull/2019) normally merged as
`0db7902fd7430e408f3aa30ae811718f036cd5c1` on 2026-10-29 at 22:19:32 UTC.
Issues #1960/#1961 closed at 22:19:33/22:19:34 UTC. Actual final-head CI has
27 terminal contexts: 20 SUCCESS / 7 SKIPPED. Actual Sonar new-code coverage is
84.4%, passing the unchanged 80% threshold.

- [Final root report](independent/eval_M-SMT-PREFLIGHT-M0-REPAIR_round_3.md) and
  [unchanged JSON](independent/eval_M-SMT-PREFLIGHT-M0-REPAIR_round_3.json).
- Historical [round 1 FAIL 73](independent/eval_M-SMT-PREFLIGHT-M0-REPAIR_round_1.md),
  [round 2 FAIL 73](independent/eval_M-SMT-PREFLIGHT-M0-REPAIR_round_2.md) and
  [round 1 Sonar corrigendum](independent/round1-sonar-corrigendum.md) remain intact.
- [Actual Sonar response](independent/sonar-qualitygate-final.json),
  [final CI monitor](monitor/final-monitor-report.json),
  [job identities](monitor/jobs-final.json), [one-retry ledger](monitor/retry-ledger.json)
  and [verified actual merge](independent/pr-actual-merge.json).

The [multiphase parent](../../../../planned/m-smt-well-sorted-preflight.md),
[repair plan](../../../../planned/v0_56_0/sprint-plan-m-smt-preflight-m0-repair.md)
and [validation](../../../../planned/v0_56_0/m-smt-preflight-m0-repair-validation.md)
remain **planned** under the multiphase archival exception. Only M0-F1/F4/Z is
accepted; parent M1–M4/F2/F5/#1950/#1953 remain deferred. Both external BLOCKED
records and the frozen M0-REVIEW section remain byte-identical. Human ratification
and subsequent implementation acceptance do not become an external PROCEED.
The outer authorization/execution fields in the frozen external JSON are dated
snapshots; current acceptance lives in the parent preface and sprint JSON.

The corpus preserves 8,634,077 original bytes with SHA256
`4bb89da9fb147b2d7c9d9ceb02c69d041ad14e3274e5bd701570aed6d2416315`
in a 621,804-byte wrapper. Its 466,009 compressed bytes have SHA256
`693d00c77f5e866201eb663c66db816ac0058040656cb926e924b2973a1c6b23`.
All nonpassing corpus rows and source/binary qualifications remain in the raw
payload and summaries. Packaging is not a claim of universal corpus success.
Historical local runtime/corpus evidence is pinned to source 60a7d7b and the
captured binary identity. The accepted 4fb checkpoint preserves production Go and
all AIL bytes while adding the 69-line SMT behavior test/validation repair;
actual final-head CI supplies the current broad-gate proof.

This namespace contains **75 logical artifacts: 69 verbatim and 6 losslessly
encoded**. Final and historical report pairs are verbatim.

## Identities, history and representation

[manifest.json](manifest.json) maps each original logical artifact to its current
relative stored path, original byte count/SHA256, encoding and stored identity.
Absolute source paths are capture provenance; they are not portable links or a
requirement for consuming the archive. Original `m0/` and `1900/` logical prefixes
are retained while current stored paths are relative to this namespace.

The frozen `staging-*` records describe the earlier preparation-only snapshot;
their pending/no-acceptance wording, old path prefixes and references to scratch
scripts are historical. This README and canonical manifest record the accepted
archive and current paths. Payloads and evaluation reports are never normalized
or rewritten. Wrapped records preserve original bytes through deterministic gzip
(mtime 0, empty filename, level 9) and base64; wrappers also authenticate compressed
bytes. Report JSON schemas are retained exactly.

[External CI log references](staging-external-ci-log-references.json) retain URLs,
local capture paths, byte counts and SHA256 for omitted raw failures, including
4.6–5 MB historical logs. A reference is not a claim that a failed run passed.
The original failures remain distinct from later actual successful jobs. This
archive adds no retry, waiver, threshold, workflow or product change.

The [root merged-source comparison](independent/three-sprint-merged-source-verification.json)
qualifies the named-path equality at combined dev b9c15d4. It does not imply that
historical dirty/local binaries were clean builds of a later commit. No tests,
builds, provider calls or CLI program executions were rerun to package this data.

## Recovering a lossless wrapper

Run from this evidence directory and choose an unused output path. This uses only
Python's standard library and refuses to overwrite a file. Verify the wrapper's
stored size/SHA against `manifest.json` before decoding; this example then checks
both compressed and original identities:

```sh
python3 - execution/corpus-final-raw.json.gzip-base64.json OUTPUT_PATH <<'PYCODE'
import base64, gzip, hashlib, json, pathlib, sys
wrapper = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
assert wrapper["encoding"] == "gzip+base64"
compressed = base64.b64decode(wrapper["payload"], validate=True)
assert len(compressed) == wrapper["compressed_bytes"]
assert hashlib.sha256(compressed).hexdigest() == wrapper["compressed_sha256"]
original = gzip.decompress(compressed)
assert len(original) == wrapper["original_bytes"]
assert hashlib.sha256(original).hexdigest() == wrapper["original_sha256"]
with pathlib.Path(sys.argv[2]).open("xb") as output:
    output.write(original)
print(wrapper["logical_name"], len(original), wrapper["original_sha256"])
PYCODE
```

No preparer, verifier or execution helper is installed as a repository tool.
The original publishing harness is archived as encoded evidence data.
