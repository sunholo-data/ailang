#!/usr/bin/env bash
# Guard: context documents obey progressive disclosure.
#
# "Context document" = anything the harness injects into a session's context
# without being asked: CLAUDE.md, .claude/rules/*.md, .claude/skills/*/SKILL.md.
# They are loaded WHOLESALE — there is no partial read — so every line in one is
# a line every future session pays for, whether or not it is relevant.
#
# The three ways they go wrong, all measured in this repo on 2026-09-03:
#
#   1. A rule with no `paths:` frontmatter loads in EVERY session, forever.
#      That is a legitimate choice for a handful of rules; it must be a stated
#      one, not an accident of a forgotten frontmatter block.
#   2. A `paths:` glob that matches nothing is a rule that never loads. Found
#      live: ailang-syntax.md was scoped to `stdlib/**` while the tree has
#      `std/` — the AILANG syntax rule had silently stopped loading for stdlib
#      edits, and nothing failed.
#   3. A SKILL.md grows without bound because appending is easier than filing.
#      Found live: mission-control/SKILL.md at 4201 lines (~96k tokens) with
#      zero reference files — a full quarter of a small context window spent
#      before the skill does anything. The fix is not brevity, it is layering:
#      keep the procedure in SKILL.md and move war stories, tables and worked
#      examples into sibling files the agent opens only when it needs them.
#
# Oversize docs that already exist are ratcheted through a baseline rather than
# fixed in one commit; the gate refuses to let them grow, and refuses to keep a
# stale entry once a doc comes back under the cap.
#
# bash 3.2 safe (the rig has no bash 4). No network.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

BASELINE=${CONTEXT_DOCS_BASELINE:-scripts/context_docs_baseline.txt}

# Caps are line counts, because tokens track lines far better than bytes for prose.
RULE_MAX_LINES=${RULE_MAX_LINES:-200}     # rules load alongside each other; keep them thin
SKILL_MAX_LINES=${SKILL_MAX_LINES:-500}   # Anthropic's guidance for a SKILL.md body
CLAUDE_MAX_LINES=${CLAUDE_MAX_LINES:-300} # the always-on root document

RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'
FAILED=0

# Arms E and the size ratchet run inside pipelines, whose subshells cannot set
# FAILED — they record to files instead.
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
TMP_SIZES="$TMP_DIR/sizes"; : > "$TMP_SIZES"
TMP_LINKS="$TMP_DIR/links"; : > "$TMP_LINKS"

fail() { printf "${RED}✗ %s${RESET}\n" "$1"; FAILED=1; }
note() { printf "    %s\n" "$1"; }

# ---------------------------------------------------------------- G0: vacuity
# Every arm below iterates a glob. If the globs match nothing the gate passes
# while asserting nothing, which is how a check quietly stops being a check.
[ -f CLAUDE.md ] || { fail "CLAUDE.md not found (run from repo root)"; exit 1; }
[ -d .claude/rules ] || { fail ".claude/rules not found — the gate would be vacuous"; exit 1; }
[ -d .claude/skills ] || { fail ".claude/skills not found — the gate would be vacuous"; exit 1; }
[ -f "$BASELINE" ] || { fail "missing baseline: $BASELINE"; exit 1; }

# Tracked-file list, computed once into a FILE: `grep -q` exits on first match
# and would SIGPIPE the producer of a pipeline, which on macOS prints a
# "write error: Broken pipe" per call and drowns the real output.
TRACKED="$TMP_DIR/tracked"
git ls-files > "$TRACKED" 2>/dev/null
[ -s "$TRACKED" ] || { fail "git ls-files returned nothing — cannot verify path globs"; exit 1; }

# glob_matches <glob> -> 0 if at least one tracked file matches, else 1.
#
# Translated to a regex rather than expanded by the shell: bash 3.2 has no
# globstar, so `internal/**` would otherwise expand to one directory level and
# under-report. Order matters — `.` is escaped first, and `**` is parked behind
# a printable sentinel so the single-`*` rule cannot eat it (BSD sed has no
# \xNN escape, so the sentinel cannot be a control character).
glob_matches() {
	local re
	re=$(printf '%s\n' "$1" \
		| sed -e 's/[.]/\\./g' \
		      -e 's/\*\*/@@GLOBSTAR@@/g' \
		      -e 's/\*/[^\/]*/g' \
		      -e 's/@@GLOBSTAR@@/.*/g' \
		      -e 's/?/[^\/]/g')
	grep -qE "^${re}$" "$TRACKED"
}

# ------------------------------------------------- A/B/C: .claude/rules/*.md
RULE_COUNT=0
for f in .claude/rules/*.md; do
	[ -f "$f" ] || continue
	RULE_COUNT=$((RULE_COUNT + 1))

	# --- A: scope is declared, one way or the other
	if [ "$(head -1 "$f")" = "---" ]; then
		fm_end=$(awk 'NR>1 && /^---[[:space:]]*$/{print NR; exit}' "$f")
		if [ -z "$fm_end" ]; then
			fail "$f: frontmatter is unterminated (no closing ---)"
			continue
		fi
		globs=$(awk -v e="$fm_end" 'NR>1 && NR<e' "$f" \
			| awk '/^paths:/{p=1;next} /^[^ ]/{p=0} p&&/^ *- /{gsub(/^ *- "?|"?$/,"");print}')
	else
		globs=""
	fi

	if [ -z "$globs" ]; then
		# No paths: means ALWAYS-ON. Allowed, but it has to be argued for in the
		# file itself, so the next reader knows it was chosen and why.
		if ! head -5 "$f" | grep -q '<!-- always-on:'; then
			fail "$f: no 'paths:' scope and no always-on marker"
			note "A rule without paths: loads in EVERY session, forever."
			note "Either scope it with a paths: frontmatter block, or declare the"
			note "cost deliberately in the first 5 lines:"
			note "  <!-- always-on: <why every session needs this> -->"
		fi
	else
		# --- B: each declared glob actually matches something
		for g in $globs; do
			if ! glob_matches "$g"; then
				# A glob may legitimately point at a real but gitignored tree.
				# Accept only if the static prefix exists on disk; a typo like
				# `stdlib/**` has no such prefix and still fails.
				prefix=$(printf '%s\n' "$g" | sed 's#/\*.*##; s#\*.*##')
				if [ -n "$prefix" ] && [ -e "$prefix" ]; then
					continue
				fi
				fail "$f: paths glob matches no tracked file: $g"
				note "This rule never loads for that path. Check the tree —"
				note "directories get renamed and the rule does not fail, it goes quiet."
			fi
		done
	fi

	# --- C: size (checked against the baseline ratchet below)
	lines=$(wc -l < "$f" | tr -d ' ')
	echo "$f $lines $RULE_MAX_LINES" >> "$TMP_SIZES"
done

[ "$RULE_COUNT" -gt 0 ] || fail "no rules found under .claude/rules — the gate is vacuous"

# ------------------------------------------------------ D: skills + CLAUDE.md
SKILL_COUNT=0
for f in .claude/skills/*/SKILL.md; do
	[ -f "$f" ] || continue
	SKILL_COUNT=$((SKILL_COUNT + 1))
	lines=$(wc -l < "$f" | tr -d ' ')
	echo "$f $lines $SKILL_MAX_LINES" >> "$TMP_SIZES"
done
[ "$SKILL_COUNT" -gt 0 ] || fail "no SKILL.md found under .claude/skills — the gate is vacuous"

echo "CLAUDE.md $(wc -l < CLAUDE.md | tr -d ' ') $CLAUDE_MAX_LINES" >> "$TMP_SIZES"

# ------------------------------------------------------- size ratchet + caps
while read -r path lines cap; do
	[ -n "${path:-}" ] || continue
	base=$(awk -v p="$path" '$1==p{print $2; exit}' "$BASELINE")

	if [ "$lines" -le "$cap" ]; then
		# Under the cap: it must not still be claiming an exemption.
		if [ -n "$base" ]; then
			fail "$path: $lines lines is under the $cap cap but is still in $BASELINE"
			note "Delete its line from the baseline — an exemption nobody removes"
			note "is an exemption that silently re-authorises the next regression."
		fi
		continue
	fi

	if [ -z "$base" ]; then
		fail "$path: $lines lines exceeds the $cap-line cap for this document class"
		note "Context docs load wholesale. Move the detail into sibling files and"
		note "link to them, so an agent opens them only when the task needs them."
		note "Deliberate exception: add '$path $lines' to $BASELINE with a why."
		continue
	fi

	if [ "$lines" -gt "$base" ]; then
		fail "$path: grew to $lines lines (baseline $base)"
		note "Baselined docs may shrink, never grow. Split before you append."
	fi
done < "$TMP_SIZES"

# ----------------------------------------------- E: relative links must resolve
# A pointer is the whole progressive-disclosure mechanism. A dead one turns
# "read the detail when you need it" into "the detail does not exist".
for f in CLAUDE.md .claude/rules/*.md; do
	[ -f "$f" ] || continue
	dir=$(dirname "$f")
	grep -o '](\.\{0,2\}[^):]*\.md[^)]*)' "$f" 2>/dev/null \
		| sed 's/^](//; s/)$//; s/#.*//' \
		| while read -r link; do
			[ -n "$link" ] || continue
			case "$link" in http*|"") continue;; esac
			if [ ! -e "$dir/$link" ] && [ ! -e "$link" ]; then
				printf "${RED}✗ %s: broken link: %s${RESET}\n" "$f" "$link"
				echo BROKEN >> "$TMP_LINKS"
			fi
		done
done
[ -s "$TMP_LINKS" ] && FAILED=1

# ------------------------------------------------------------------- verdict
if [ "$FAILED" -ne 0 ]; then
	printf "\n${RED}✗ context-doc gate failed${RESET}\n"
	printf "  Convention: %s\n" ".claude/rules/context-docs.md"
	exit 1
fi
printf "${GREEN}✓ context docs: %s rules, %s skills, CLAUDE.md — scoped, linked, within budget${RESET}\n" \
	"$RULE_COUNT" "$SKILL_COUNT"
exit 0
