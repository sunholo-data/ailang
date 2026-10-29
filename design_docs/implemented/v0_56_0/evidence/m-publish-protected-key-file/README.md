# Accepted protected publisher key-file evidence — v0.56.0 UNRELEASED

The independent evaluator accepted **PASS 95/100, round 2**, at
`1f6c2ea68a5f45917f91935cdf2470e3833094e1`; root then accepted and normally
merged [PR #2021](https://github.com/sunholo-data/ailang/pull/2021) as
`b9c15d431ee25fb28c7bc2729f199cb80c8cdada` on 2026-10-29 at 22:20:12 UTC.
Issue #1900 closed at 22:20:13 UTC. Actual final-head CI has 27 terminal contexts:
20 SUCCESS / 7 SKIPPED. Actual Sonar new-code coverage is 90.0%, passing the
unchanged 80% threshold.

- [Final independent report](independent/eval_M-PUBLISH-PROTECTED-KEY-FILE_round_2.md)
  and [unchanged JSON](independent/eval_M-PUBLISH-PROTECTED-KEY-FILE_round_2.json).
  The JSON schema is `result: "pass"`, `total_score: 95`, not `verdict`/`score`.
- [Original round 1 FAIL 73](independent/eval_M-PUBLISH-PROTECTED-KEY-FILE_round_1.md),
  [original failing CI state](independent/pr-796b-round1-terminal-failure.json) and
  [root audit-script schema correction](independent/root-audit-comment-correction.json)
  remain intact. Root reviewed the Markdown report and actual terminal CI before
  merge; the later script correction does not rewrite the evaluator's report.
- [Actual Sonar response](independent/sonar-gate-1f6.json),
  [terminal CI](independent/pr-1f6-terminal.json),
  [root acceptance](independent/root-acceptance-comment.md) and
  [actual merge](independent/pr-root-merge.json).

The [parent](../../m-publish-protected-key-file.md),
[plan](../../m-publish-protected-key-file-sprint-plan.md) and
[validation](../../m-publish-protected-key-file-validation.md) move together.
This is accepted implementation, unreleased; no tag or version bump is implied.
Historical local execution used the captured f953-dirty binary and is qualified
accordingly. Final actual 1f6 CI supplies clean-head broad proof; production Go
was unchanged by its final coverage-test repair. Unix protected-file acceptance,
Windows fail-closed unsupported behavior and same-UID limitations are retained;
there is no Windows ACL or same-UID isolation claim.

This namespace contains **61 logical artifacts: 58 verbatim and 3 losslessly
encoded**. Final and historical report pairs are verbatim. One encoded payload is
the original execution harness, retained as data rather than an installed tool.

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
python3 - execution/M1-green.log.gzip-base64.json OUTPUT_PATH <<'PYCODE'
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
