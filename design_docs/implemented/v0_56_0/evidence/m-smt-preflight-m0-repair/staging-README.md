# Reversible evidence staging — no acceptance claim

This directory is preparation only. No repository, design/plan status, archive,
merge, release, gate or final-evaluation claim is changed by its creation. M0
round1/2 and publishing round1 are immutable historical reports, including their
failures. Final M0/root and publishing/independent reports were not staged; the
assigned judges own them. Existing pending qualifications inside copied source
artifacts remain byte-identical, rather than being rewritten as current status.

`manifest.json` maps each original logical name to its original byte count/SHA256,
source path, staged path, encoding and staged byte count/SHA256. Seventeen original
M0 raw-artifact entries are checked against the committed implementation evidence
record. Read identity and scope qualifications in those artifacts before using
corpus evidence for a later source head. Coverage percentages labelled projected
are still projections; packaging does not turn them into actual Sonar results.

Ordinary small files are copied verbatim. The load-bearing M0 corpus is a JSON
wrapper containing deterministic gzip (mtime0, empty original filename, level9)
then standard base64. Logs with carriage returns or trailing whitespace use the
same lossless wrapper so an eventual text-hygiene gate cannot silently alter raw
evidence. The wrapper records original and compressed sizes/hashes, encoding,
compression parameters and payload. Its `logical_name` is the original artifact
name; no JSON payload normalization is applied before compression.

The corpus wrapper is:
`m0/execution/corpus-final-raw.json.gzip-base64.json`.
Its original8,634,077bytes have SHA256
`4bb89da9fb147b2d7c9d9ceb02c69d041ad14e3274e5bd701570aed6d2416315`.

## Portable standard-library decode example

Choose an unused output path explicitly; this example refuses to overwrite it.
Run from this staging directory (or change the wrapper argument):

```sh
python3 - m0/execution/corpus-final-raw.json.gzip-base64.json OUTPUT_PATH <<'PY'
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
PY
```

`verify_staging.py` independently reads every staged entry, validates both levels
of hashes/lengths, decodes wrappers, and compares full bytes to the original source
files. It also checks gzip mtime/filename and repeats corpus compression to verify
byte-deterministic output on this host. It writes only `decode-checks.json`; it does
not write a decoded8MB corpus into the staging payload or invoke product tools.
`prepare_staging.py` is the transparent staging recipe, not a repository tool.

`external-ci-log-references.json` retains actual CI URLs, local raw-log byte counts
and hashes without copying failure payloads into the eventual repository. The
historical1773 4.6MB logs remain owned by their existing archive. There is no
claim here that a failed job is resolved, retried or accepted.

Routing after actual independent PASS is a separate authorized operation: M0's
multi-phase parent design and companion remain planned; publishing's parent,
plan and validation move only after actual PASS. Do not copy this directory into
an implemented archive as a substitute for that decision. No source checkout was
modified by staging, and no new provider/tool/credential action was performed.
