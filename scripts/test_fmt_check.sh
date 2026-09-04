#!/usr/bin/env bash
# Mutation-sensitive self-test for make fmt-check.
set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd) || exit 1
W=$(mktemp -d "$ROOT/.tmp-iter327.fmt-check.XXXXXX") || exit 1
trap 'rm -rf "$W"' EXIT
trap 'rm -rf "$W"; exit 1' HUP INT TERM

pass=0
fail=0
ck() {
    if [ "$2" = "$3" ]; then
        echo "  PASS $1"
        pass=$((pass + 1))
    else
        echo "  FAIL $1 (got '$2' want '$3')"
        fail=$((fail + 1))
    fi
}

fixture() {
    name="$1"
    mkdir -p "$W/$name" || exit 1
    printf 'include %s/make/code-health.mk\n' "$ROOT" > "$W/$name/Makefile" || exit 1
}

fixture formatted
printf 'package formatted\n\nfunc main() {}\n' > "$W/formatted/main.go"
out=$(make -s -C "$W/formatted" fmt-check 2>&1); rc=$?
ck "fmt-check formatted fixture: accepts" "$rc" "0"

fixture empty
: > "$W/empty/empty.go"
out=$(make -s -C "$W/empty" fmt-check 2>&1); rc=$?
ck "fmt-check empty Go: refuses formatter error" "$([ "$rc" -ne 0 ] && echo refused):$(printf '%s' "$out" | grep -c 'gofmt failed')" "refused:1"

fixture malformed
printf 'package main\nfunc(' > "$W/malformed/bad.go"
out=$(make -s -C "$W/malformed" fmt-check 2>&1); rc=$?
ck "fmt-check malformed Go: refuses formatter error" "$([ "$rc" -ne 0 ] && echo refused):$(printf '%s' "$out" | grep -c 'gofmt failed')" "refused:1"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ]
