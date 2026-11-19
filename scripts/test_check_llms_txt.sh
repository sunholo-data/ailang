#!/bin/bash
# test_check_llms_txt.sh — self-test for scripts/check_llms_txt.sh (bash 3.2).
#
# The gate must pass exactly when the three llms.txt copies are what
# tools/generate-llms-txt.sh produces for the ACTIVE prompt, and fail when any
# copy embeds another prompt, is stale against an edited active prompt, or is
# missing. Each arm runs the real generator and the real gate in a scratch tree.
# (A review mutation that made the gate compare the wrong file survived until
# this existed.)
set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GATE="$REPO_ROOT/scripts/check_llms_txt.sh"
GEN="$REPO_ROOT/tools/generate-llms-txt.sh"
FAILED=0
ARMS_RUN=0
ARMS_EXPECTED=6

pass() { echo "  ok   — $1"; ARMS_RUN=$((ARMS_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; ARMS_RUN=$((ARMS_RUN + 1)); }

WORK="$(mktemp -d "${TMPDIR:-/tmp}/llms-gate.XXXXXX")"
trap 'rm -rf -- "${WORK:?}"' EXIT

setup() {
    rm -rf -- "${WORK:?}/root"
    mkdir -p "$WORK/root/tools" "$WORK/root/prompts" "$WORK/root/docs/static"
    cp "$GEN" "$WORK/root/tools/generate-llms-txt.sh"
    echo "# README" > "$WORK/root/README.md"
    printf '# AILANG v9.9.1 - prompt A\nalpha body\n' > "$WORK/root/prompts/v9.9.1.md"
    printf '# AILANG v9.9.2 - prompt B\nbeta body\n' > "$WORK/root/prompts/v9.9.2.md"
    cat > "$WORK/root/prompts/versions.json" <<'JSON'
{"versions": {"v9.9.1": {"file": "prompts/v9.9.1.md"}, "v9.9.2": {"file": "prompts/v9.9.2.md"}}, "active": "v9.9.2"}
JSON
}
generate() { /bin/bash "$WORK/root/tools/generate-llms-txt.sh" >/dev/null 2>&1; }
gate() { LLMS_TXT_ROOT="$WORK/root" /bin/bash "$GATE" >/dev/null 2>&1; }
set_active() { sed -i.bak "s/\"active\": \"[^\"]*\"/\"active\": \"$1\"/" "$WORK/root/prompts/versions.json"; }

echo "llms.txt gate:"

setup; generate
if gate; then pass "generator output for the active prompt passes"; else fail "generator output for the active prompt was refused"; fi

setup; generate; set_active v9.9.1
if gate; then fail "a stale copy (generated for v9.9.2, active v9.9.1) passed"; else pass "copies generated for another prompt version fail"; fi

setup; generate; printf 'an edit to the active prompt\n' >> "$WORK/root/prompts/v9.9.2.md"
if gate; then fail "an edited active prompt with stale copies passed"; else pass "an active prompt edited after generation fails"; fi

setup; set_active v9.9.1; generate; cp "$WORK/root/llms.txt" "$WORK/root/old.txt"
set_active v9.9.2; generate; cp "$WORK/root/old.txt" "$WORK/root/docs/static/llms.txt"
if gate; then fail "one copy (docs/static) embedding the inactive prompt passed"; else pass "a single stale copy fails"; fi

setup; generate; rm -f "$WORK/root/docs/llms.txt"
if gate; then fail "a missing copy passed"; else pass "a missing copy fails"; fi

setup; generate; printf 'an edit\n' >> "$WORK/root/prompts/v9.9.2.md"; generate
if gate; then pass "regenerating after the edit passes again"; else fail "regenerated copies were refused"; fi

if [ "$ARMS_RUN" -ne "$ARMS_EXPECTED" ]; then
    echo "  FAIL — ran $ARMS_RUN arms, expected $ARMS_EXPECTED"
    FAILED=1
fi
if [ "$FAILED" -ne 0 ]; then
    echo "✗ llms.txt gate self-test failed"
    exit 1
fi
echo "✓ llms.txt gate self-test: $ARMS_RUN arms"
