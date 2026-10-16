#!/usr/bin/env bash
set -euo pipefail
# Resolve the executable before changing to the repository root.
binary=$(command -v "${1:-ailang}")
binary=$(realpath "$binary")
runs=${2:-3}
[[ "$runs" =~ ^[1-9][0-9]*$ ]] || { echo 'runs must be positive' >&2; exit 1; }
cd "$(dirname "$0")/../.."
python3 - "$binary" "$runs" <<'PY'
import subprocess
import sys
import time

binary, runs = sys.argv[1], int(sys.argv[2])

def call(args, expected=None):
    result = subprocess.run([binary, *args], capture_output=True, text=True)
    if result.returncode:
        sys.stderr.write(result.stderr)
        raise SystemExit(f"command failed ({result.returncode}): {args}")
    if expected is not None and result.stdout != expected:
        sys.stderr.write(result.stderr)
        raise SystemExit(f"incorrect output: {args}: {result.stdout!r}, expected {expected!r}")
    return result.stdout

print(f"binary: {binary}", flush=True)
print(call(["--version"]).strip(), flush=True)
print(f"runs: {runs}; monotonic wall seconds include startup/compilation/input construction", flush=True)

def execute(module, n, flags):
    start = time.perf_counter()
    expected = f"{n if module == 'build' else max(n - 1, 0)}\n"
    output = call(["run", "--quiet", *flags, "--caps", "IO", "--args-json", str(n),
                   f"bench/vm_tail_walk/{module}.ail"], expected)
    elapsed = time.perf_counter() - start
    return output, elapsed

for module in ("walk", "foldl", "build"):
    call(["check", f"bench/vm_tail_walk/{module}.ail"])
    for n in (0, 1, 2):
        baseline, _ = execute(module, n, [])
        vm, _ = execute(module, n, ["--bytecode"])
        assert vm == baseline
    execute(module, 2, ["--strict-bytecode"])
    sizes = (10000, 20000, 40000) if module == "build" else (40000, 80000, 160000)
    for n in sizes:
        samples = {}
        baseline = None
        for engine, flags in (("interpreter", []), ("vm", ["--bytecode"])):
            samples[engine] = []
            for _ in range(runs):
                output, elapsed = execute(module, n, flags)
                if baseline is None:
                    baseline = output
                if output != baseline:
                    raise SystemExit("engine output mismatch")
                samples[engine].append(elapsed)
            print(f"{module} n={n} {engine}: min={min(samples[engine]):.6f} samples={samples[engine]}", flush=True)
        print(f"VM/interpreter={min(samples['vm']) / min(samples['interpreter']):.4f}", flush=True)
PY
