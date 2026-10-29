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

ARM64_PAT='FMADDD|FMSUBD|FNMADDD|FNMSUBD|FMADDS|FMSUBS|FNMADDS|FNMSUBS'
AMD64_PAT='VFMADD|VFMSUB|VFNMADD|VFNMSUB'
npkgs=$(echo $PKGS | wc -w | tr -d ' ')

# One `go test -c` per arch builds all four packages (shared deps compile once),
# and the two arches build side by side: 33.6s -> 8.9s on a many-core Mac, no
# change on CI's 4-core runner (93-121s before, 124s after; dev, 2026-10-27).
build() {
	local arch="$1" amd64="$2"
	mkdir -p "$TMP/$arch"
	GOOS=linux GOARCH="$arch" GOAMD64="$amd64" CGO_ENABLED=0 go test -c -o "$TMP/$arch/" $PKGS
}
build arm64 v1 & arm_pid=$!
build amd64 v3 & amd_pid=$!
build_rc=0
wait "$arm_pid" || build_rc=1
wait "$amd_pid" || build_rc=1
[ "$build_rc" -eq 0 ] || { echo "FAIL: cross-compiling the runtime test binaries failed"; exit 1; }

status=0
short=0
pids=""
for arch in arm64 amd64; do
	if [ "$arch" = arm64 ]; then pattern="$ARM64_PAT"; else pattern="$AMD64_PAT"; fi
	n=0
	for bin in "$TMP/$arch"/*.test; do
		[ -f "$bin" ] || continue
		n=$((n + 1))
		go tool objdump "$bin" | awk -v scope="$SCOPE" -v pat="$pattern" -v arch="$arch" '
			/^TEXT / { fn = $2; file = $3; inscope = (fn ~ scope) && (file !~ /_test\.go$/); next }
			inscope && $0 ~ pat { hits[fn]++ }
			END { for (f in hits) { printf "  %s: %d fused op(s) in %s\n", arch, hits[f], f; bad = 1 } exit bad }
		' > "$bin.hits" &
		pids="$pids $!"
	done
	# A package that stopped producing a test binary must not shrink the scan silently.
	if [ "$n" -ne "$npkgs" ]; then
		echo "FAIL: $arch built $n test binaries, expected $npkgs ($PKGS)"
		short=1
	fi
done
for pid in $pids; do wait "$pid" || status=1; done
cat "$TMP"/*/*.hits
[ "$short" -eq 0 ] || exit 1

if [ "$status" -ne 0 ]; then
	echo "FAIL: fused multiply-add in AILANG runtime code (#1465)."
	echo "      Wrap the product in float64(...) so every architecture rounds it."
	exit 1
fi
echo "OK: no fused multiply-add in AILANG runtime code (arm64, amd64/v3)"
