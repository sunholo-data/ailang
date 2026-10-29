#!/usr/bin/env python3
"""Isolated controls for the manifest sync and CI validator (no repository writes)."""
import json
import pathlib
import shutil
import subprocess
import tempfile


def run(command, diagnostic=None):
    result = subprocess.run(command, capture_output=True, text=True)
    output = result.stdout + result.stderr
    if diagnostic is None:
        assert result.returncode == 0, output
    else:
        assert result.returncode != 0, output
        assert diagnostic in output, output
    return output


with tempfile.TemporaryDirectory(prefix="manifest-selftest-") as temporary:
    root = pathlib.Path(temporary)
    validator = root / "validator"
    writer = root / "writer"
    run(["go", "build", "-o", str(validator), "./scripts/validate_manifest.go"])
    run(["go", "build", "-o", str(writer), "./scripts/backfill_manifest_modules.go"])
    examples = root / "examples"
    runnable = examples / "runnable"
    runnable.mkdir(parents=True)
    source = pathlib.Path("examples/runnable/stdlib_gzip.ail")
    shutil.copyfile(source, runnable / source.name)
    manifest = root / "manifest.json"
    baseline = json.loads(pathlib.Path("examples/manifest.json").read_text())
    baseline["examples"] = [e for e in baseline["examples"] if e["path"] == "runnable/stdlib_gzip.ail"]
    assert len(baseline["examples"]) == 1
    baseline["future_authored_field"] = {"keep": "unchanged"}
    manifest.write_text(json.dumps(baseline))
    sync = [str(writer), "--manifest", str(manifest), "--dir", str(examples)]
    validate = [str(validator), "--manifest", str(manifest), "--dir", str(examples), "--ci"]
    original = manifest.read_bytes()
    run(sync + ["--check"], "statistics drift: true")
    assert manifest.read_bytes() == original, "--check wrote the manifest"
    run(sync)
    repaired = json.loads(manifest.read_text())
    assert list(repaired) == list(baseline), "top-level key order changed"
    for key in baseline:
        if key not in {"examples", "statistics"}:
            assert repaired[key] == baseline[key], key
    for before, after in zip(baseline["examples"], repaired["examples"]):
        assert {k: v for k, v in before.items() if k != "modules"} == {
            k: v for k, v in after.items() if k != "modules"
        }, "authored entry fields changed"
    run(validate)
    run(sync + ["--check"])
    valid = manifest.read_bytes()
    run(sync)
    assert manifest.read_bytes() == valid, "sync is not idempotent"

    stale = json.loads(valid)
    stale["statistics"]["total"] += 1
    manifest.write_text(json.dumps(stale))
    output = run(validate, "statistics mismatch")
    assert "make sync-examples-manifest" in output, output
    run(sync)
    run(validate)

    drifted = json.loads(valid)
    drifted["examples"][0]["modules"] = ["std/DRIFT"]
    manifest.write_text(json.dumps(drifted))
    run(sync + ["--check"], "modules 1")
    run(validate, "modules drift")
    run(sync)
    run(validate)

    # Nested files are gated; underscore scratch files and grandfathered paths pass.
    nested = runnable / "nested"
    nested.mkdir()
    unknown = nested / "unlisted.ail"
    shutil.copyfile(source, unknown)
    run(validate, "unlisted runnable example: runnable/nested/unlisted.ail")
    unknown.unlink()
    shutil.copyfile(source, runnable / "_scratch.ail")
    shutil.copyfile(source, runnable / "ai_call_result.ail")
    run(validate)

    # Legacy bare paths resolve to runnable/ and must not be reported as unlisted.
    legacy = json.loads(valid)
    legacy["examples"][0]["path"] = source.name
    manifest.write_text(json.dumps(legacy))
    run(validate)

    # Empty manifests have finite, zero coverage and all six counters.
    empty = json.loads(valid)
    empty["examples"] = []
    manifest.write_text(json.dumps(empty))
    run(sync)
    stats = json.loads(manifest.read_text())["statistics"]
    assert len(stats) == 6 and all(value == 0 for value in stats.values()), stats
    run(sync + ["--check"])
    print("Manifest self-test passed: modules, statistics, coverage, allowlist, scratch, legacy paths, idempotence")
