#!/usr/bin/env bash
#
# install_z3.sh — put a pinned z3 on PATH for CI (contract-verification tests).
#
# `apt-get update && apt-get install z3` took 161-248s on 3 of 12 dev runs
# (2026-10-27; 6-19s on the rest), silent the whole time under -qq. This takes
# the same z3 version Ubuntu ships (4.13.3) as an official release asset from
# GitHub, checked against a pinned sha256, so no Ubuntu mirror is involved.
# internal/smt finds the solver with exec.LookPath("z3").

set -euo pipefail

VERSION=4.13.3
ASSET="z3-$VERSION-x64-glibc-2.35"
SHA256=32c7377026733c9d7b33c21cd77a68f50ba682367207b031a6bfd80140a8722f
URL="https://github.com/Z3Prover/z3/releases/download/z3-$VERSION/$ASSET.zip"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/ailang-z3.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL --retry 3 --retry-delay 5 --max-time 120 -o "$TMP/z3.zip" "$URL"
echo "$SHA256  $TMP/z3.zip" | sha256sum -c -
unzip -q -j "$TMP/z3.zip" "$ASSET/bin/z3" -d "$TMP"
sudo install -m 0755 "$TMP/z3" /usr/local/bin/z3

got="$(z3 --version)"
echo "$got"
case "$got" in
	"Z3 version $VERSION "*) ;;
	*) echo "::error::expected Z3 version $VERSION on PATH, got: $got"; exit 1 ;;
esac
