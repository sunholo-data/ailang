#!/bin/bash
# stdlib-iface-lib.sh — shared plumbing for freeze-stdlib.sh / verify-stdlib.sh
#
# Sourced, never executed. Both scripts MUST agree on binary, module list and
# JSON generation, or the gate compares apples to oranges and reports drift that
# is really a difference between the two scripts.

STDLIB_DIR="std"
GOLDEN_DIR=".stdlib-golden"

# resolve_ailang — prefer the freshly built binary over whatever is on PATH.
#
# The old scripts called bare `ailang`, i.e. they verified the SYSTEM binary's
# idea of the interfaces while you were editing the working tree. Same
# convention as tools/verify_examples.sh.
resolve_ailang() {
    AILANG="./bin/ailang"
    if [ ! -x "$AILANG" ]; then
        AILANG="$(command -v ailang || true)"
    fi
    if [ -z "$AILANG" ]; then
        echo "error: ailang binary not found (run 'make build')" >&2
        exit 1
    fi
}

# stdlib_modules — every module under std/, derived from the filesystem.
#
# The old scripts hardcoded MODULES="io list option result string": 5 of 45.
# A freeze gate covering 11% of the surface is worse than none, because it reads
# as coverage. Sorted for a deterministic run order.
stdlib_modules() {
    find "$STDLIB_DIR" -maxdepth 1 -name '*.ail' -exec basename {} .ail \; | LC_ALL=C sort
}

# iface_json <module> <outfile> — write the module's normalized interface JSON.
#
# Returns non-zero and prints the real error on failure. The old scripts piped
# `ailang iface ... 2>/dev/null | grep -A 10000 '^{'`, which (a) discarded the
# diagnostic, (b) truncated any interface over 10000 lines, and (c) under
# `set -e` killed the script via grep's exit status with NO message at all —
# which is exactly how this gate sat broken without anyone seeing why.
# stdout is already clean JSON; the version-mismatch warning goes to stderr.
iface_json() {
    local module="$1" outfile="$2" errfile
    errfile="$(mktemp)"
    if ! "$AILANG" iface "std/$module" >"$outfile" 2>"$errfile"; then
        echo "  ✗ $module: 'ailang iface std/$module' failed" >&2
        sed 's/^/      /' "$errfile" >&2
        rm -f "$errfile"
        return 1
    fi
    if [ ! -s "$outfile" ]; then
        echo "  ✗ $module: 'ailang iface std/$module' produced no output" >&2
        rm -f "$errfile"
        return 1
    fi
    rm -f "$errfile"
    return 0
}

# sha256_of <file>
sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        echo "error: no SHA256 tool found (sha256sum or shasum)" >&2
        exit 1
    fi
}
