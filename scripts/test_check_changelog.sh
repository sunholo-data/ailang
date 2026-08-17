#!/bin/bash
# test_check_changelog.sh — the root-CHANGELOG index gate must refuse EVERY heading shape.
#
# WHY THIS EXISTS (2026-08-17, V1 mission iteration 217).
# `scripts/check_changelog.sh` is a refusal gate with no coverage at all: it shipped, went red
# once, and nobody had ever checked WHICH release-note shapes it can see. A gate's coverage is a
# property of its enumerator, and this one enumerated lexically — named Keep-a-Changelog verbs
# (`### Added|Fixed|Changed|Removed|Deprecated|Security`) plus bracketed version headings
# (`## [Unreleased]`). Measured on dev at 0002c9b0b: root CHANGELOG.md held FIVE stranded
# sections spanning 169 lines and the gate flagged exactly ONE. The other four —
# `## v0.32.0 (Unreleased)` (no brackets), `### Docs — staleness/fluff audit`,
# `### Eval cost accuracy — …` (x2) and `### Mission infrastructure …` (no verb) — were invisible
# by construction, i.e. release notes that release-manager would silently drop, sitting inside the
# very file whose guard was printing a failure about a different line.
#
# So the arms below are not decoration: arms 3–5 are the ones the PREVIOUS gate passed. They are
# the reason this file exists, and each is a mutation that must red.
set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GATE="$REPO_ROOT/scripts/check_changelog.sh"
FAILED=0
ARMS_RUN=0
ARMS_EXPECTED=13

pass() { echo "  ok   — $1"; ARMS_RUN=$((ARMS_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; ARMS_RUN=$((ARMS_RUN + 1)); }

echo "changelog index gate:"

if [ ! -f "$GATE" ]; then
  echo "  FAIL — $GATE not found; instrument failure, refusing to report a green"
  exit 1
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# A realistic index fixture. The clean arm is a control that CAN fail: if the gate ever became
# unconditionally red, arm 1 catches it; if it became unconditionally green, arms 2-9 catch it.
make_fixture() {
  _dir="$1"
  mkdir -p "$_dir/changelogs"
  printf '%s\n' \
    '# AILANG Changelog' \
    '' \
    'For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).' \
    '' \
    '## Changelog Archives' \
    '' \
    'The full changelog has been split into themed files.' \
    '' \
    '| File | Versions | Theme |' \
    '|------|----------|-------|' \
    '| [v0.18-current.md](changelogs/v0.18-current.md) | v0.18.0+ | Current |' \
    > "$_dir/CHANGELOG.md"
  printf '%s\n' '# Current changelog' '' '## [Unreleased]' '' '### Added — a real entry' \
    > "$_dir/changelogs/v0.18-current.md"
}

# Runs the gate with cwd = fixture (the gate reads CHANGELOG.md and changelogs/ relatively),
# capturing rc and output separately. Exit code is read directly, never through a pipe.
OUT=""; RC=0
# NOTE: this sets globals rather than echoing rc, and that is load-bearing. An earlier draft was
# called as `expect ... "$(run_gate "$D")"`, and command substitution runs in a SUBSHELL — so the
# captured output never reached the assertion and every message check compared against an empty
# string. rc-only assertions had hidden the same class one layer down; this one hid the fix.
run_gate() {
  OUT=$( cd "$1" && bash "$GATE" 2>&1 )
  RC=$?
}

# Every arm asserts rc AND a message fragment unique to the branch it targets.
#
# rc alone is NOT discriminating and this was measured, not assumed: with rc-only assertions, a
# mutation drill (`if false && ...` on each refusal branch) left TWO branches alive — neutering
# the missing-file check still produced rc=1 (the archive-count branch refused instead), and
# neutering the active-link check left its arm green (the fixture had tripped a different branch).
# Both arms were passing for the wrong mechanism. An exit code is a three-value observable shared
# by every refusal; the message is not.
expect() {
  _name="$1"; _want="$2"; _got="$3"; _frag="$4"
  if [ "$_got" -ne "$_want" ]; then
    fail "$_name — expected rc=$_want, got rc=$_got"
  elif ! printf '%s' "$OUT" | grep -qF "$_frag"; then
    fail "$_name — rc=$_got as expected, but output does not mention '$_frag' (wrong branch fired)"
  else
    pass "$_name (rc=$_got)"
  fi
}

# --- arm 1: a clean index passes -------------------------------------------------------------
D="$WORK/clean"; make_fixture "$D"
run_gate "$D"
expect "clean index passes" 0 "$RC" "is index-only"

# --- arm 2: the shape the OLD gate already caught (regression guard) --------------------------
D="$WORK/verb"; make_fixture "$D"
printf '\n%s\n' '### Added `some-model` — a release note' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses '### Added …' (verb heading)" 1 "$RC" "contains release-note content"

# --- arm 3: NEW — a section heading with no Keep-a-Changelog verb -----------------------------
# This is one of the four that shipped invisibly on dev.
D="$WORK/noverb"; make_fixture "$D"
printf '\n%s\n' '### Docs — staleness/fluff audit (2026-08-17)' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses '### Docs — …' (no verb; OLD gate passed this)" 1 "$RC" "contains release-note content"

# --- arm 4: NEW — an unbracketed version heading ----------------------------------------------
D="$WORK/unbracketed"; make_fixture "$D"
printf '\n%s\n' '## v0.32.0 (Unreleased)' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses '## v0.32.0 (Unreleased)' (no brackets; OLD gate passed this)" 1 "$RC" "contains release-note content"

# --- arm 5: NEW — any heading depth ------------------------------------------------------------
D="$WORK/deep"; make_fixture "$D"
printf '\n%s\n' '#### Some deeper release note' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses a level-4 heading" 1 "$RC" "contains release-note content"

# --- arm 6: anti-vacuity floor — no archive heading --------------------------------------------
# Without it the file has no recognisable index shape, so "no offenders" would be vacuous.
D="$WORK/noarchive"; make_fixture "$D"
grep -v '^## Changelog Archives$' "$D/CHANGELOG.md" > "$D/tmp" && mv "$D/tmp" "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses a file with no '## Changelog Archives' heading" 1 "$RC" "expected exactly 1"

# --- arm 7: two archive headings are also unrecognisable ----------------------------------------
D="$WORK/dupearchive"; make_fixture "$D"
printf '\n%s\n' '## Changelog Archives' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses duplicated '## Changelog Archives'" 1 "$RC" "expected exactly 1"

# --- arm 8: the index must link the active changelog --------------------------------------------
# The fixture keeps EXACTLY ONE archive heading and no other heading, so the only branch that can
# refuse it is the link check. An earlier draft appended a second archive heading here and was
# refused by the archive-count branch instead — green for the wrong mechanism.
D="$WORK/nolink"; make_fixture "$D"
grep -v 'changelogs/v0.18-current.md' "$D/CHANGELOG.md" > "$D/tmp" && mv "$D/tmp" "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses an index that does not link the active changelog" 1 "$RC" "does not link the active changelog"

# --- arm 9: a missing CHANGELOG.md is a refusal, not a pass --------------------------------------
D="$WORK/missing"; mkdir -p "$D/changelogs"
printf '%s\n' '# Current' > "$D/changelogs/v0.18-current.md"
run_gate "$D"
expect "refuses a missing CHANGELOG.md" 1 "$RC" "not found (run from repo root)"

# --- arms 10-11: NEW — bracketed headings, named explicitly ------------------------------------
# The structural rule ("any heading but the archive table") already refuses these, and arms 3-5
# above cover the shapes that used to be invisible. These two pin the shapes the OLD lexical gate
# DID catch by name, so a future rewrite cannot regress them silently while arms 3-5 stay green.
# Carried over from motoko's #758, which was closed as superseded on 2026-08-17 under the
# repo-ownership rule; its self-test had these and the landed one did not.
D="$WORK/bracketed-unreleased"; make_fixture "$D"
printf '\n%s\n' '## [Unreleased]' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses '## [Unreleased]' (bracketed)" 1 "$RC" "contains release-note content"

D="$WORK/bracketed-version"; make_fixture "$D"
printf '\n%s\n' '## [v0.32.0]' >> "$D/CHANGELOG.md"
run_gate "$D"
expect "refuses '## [v0.32.0]' (bracketed version)" 1 "$RC" "contains release-note content"

# --- arms 12-13: NEW — the index must point somewhere -------------------------------------------
# These target the second anti-vacuity floor. Before it, BOTH of these fixtures exited 0 with
# `✓ CHANGELOG.md is index-only and links changelogs/` — a blank filename in a success message,
# certifying a link to a file that does not exist. Measured on dev at a99083ae5.
# Two input shapes, one branch: the directory is gone, and the directory survives with no
# *current* file in it (a rename or an archive rollover).
D="$WORK/nochangelogsdir"; make_fixture "$D"
rm -rf "$D/changelogs"
run_gate "$D"
expect "refuses a missing changelogs/ directory" 1 "$RC" "the index points nowhere"

D="$WORK/nocurrentfile"; make_fixture "$D"
rm -f "$D/changelogs/v0.18-current.md"; printf '%s\n' '# Old' > "$D/changelogs/v0.17-old.md"
run_gate "$D"
expect "refuses a changelogs/ with no *current* file" 1 "$RC" "the index points nowhere"

# --- anti-vacuity: every arm must have run ------------------------------------------------------
if [ "$ARMS_RUN" -ne "$ARMS_EXPECTED" ]; then
  echo "  FAIL — $ARMS_RUN of $ARMS_EXPECTED arms ran; a short arm set is vacuously green"
  FAILED=1
fi

if [ "$FAILED" -eq 0 ]; then
  echo "changelog index gate: OK ($ARMS_RUN arms)"
  exit 0
fi
echo "changelog index gate: FAILED"
exit 1
