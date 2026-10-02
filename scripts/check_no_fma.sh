#!/usr/bin/env bash
#
# check_no_fma.sh — cross-architecture float determinism gate (#1465).
#
# The Go spec lets the compiler fuse x*y+z into one FMA instruction on arm64
# (and on amd64 with GOAMD64>=v3), "possibly across statements". A fused
# result can differ from the separately-rounded one in the last bit, so any
# fusion in code that computes AILANG values makes a pure program print
# different numbers on different CPUs. The remedy is an explicit float64(...)
# conversion around the product, which the spec says prevents fusion.
#
# This gate cross-compiles the language runtime packages for arm64 and
# amd64/v3 and fails if any FMA-family instruction appears in non-test code
# of those packages. It needs only the Go toolchain (no arm64 hardware).
#
# Exit 0 = clean. Exit 1 = fused instruction found (function names printed).

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/ailang-nofma.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

# Test binaries link each package together with its dependencies
# (eval, effects, bytecode, runtime, ...), so these cover the runtime surface.
PKGS="./internal/mathx ./internal/builtins ./internal/vm ./internal/simhash"
# Functions whose results reach AILANG programs.
SCOPE='github\.com/sunholo-data/ailang/internal/(mathx|builtins|vm|eval|effects|bytecode|runtime|simhash)\.'

status=0
check() {
	local arch="$1" amd64="$2" pattern="$3"
	local pkg name bin
	for pkg in $PKGS; do
		name="$(basename "$pkg")"
		bin="$TMP/$name.$arch.test"
		GOOS=linux GOARCH="$arch" GOAMD64="$amd64" CGO_ENABLED=0 go test -c -o "$bin" "$pkg"
		go tool objdump "$bin" | awk -v scope="$SCOPE" -v pat="$pattern" -v arch="$arch" '
			/^TEXT / { fn = $2; file = $3; inscope = (fn ~ scope) && (file !~ /_test\.go$/); next }
			inscope && $0 ~ pat { hits[fn]++ }
			END { for (f in hits) { printf "  %s: %d fused op(s) in %s\n", arch, hits[f], f; bad = 1 } exit bad }
		' || status=1
	done
}

check arm64 v1 'FMADDD|FMSUBD|FNMADDD|FNMSUBD|FMADDS|FMSUBS|FNMADDS|FNMSUBS'
check amd64 v3 'VFMADD|VFMSUB|VFNMADD|VFNMSUB'

if [ "$status" -ne 0 ]; then
	echo "FAIL: fused multiply-add in AILANG runtime code (#1465)."
	echo "      Wrap the product in float64(...) so every architecture rounds it."
	exit 1
fi
echo "OK: no fused multiply-add in AILANG runtime code (arm64, amd64/v3)"
