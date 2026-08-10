#!/usr/bin/env bash
# Guard: every .claude/skills/*/SKILL.md carries usable YAML frontmatter.
#
# A SKILL.md with no `---` frontmatter block still LOADS -- the loader falls back
# to the H1 heading as the skill's description. Since the description is what
# drives skill triggering, the skill silently under-triggers instead of failing.
# Found 2026-08-10: motoko-analyzer (the mission diagnostic loop) and
# ailang-packages had both been advertising their titles as their descriptions.
#
# Deliberately NOT a full YAML validity check: that needs a parser this repo does
# not carry, and two live skills use an unquoted ": " inside their description
# (`Use when: ...`, `(default: the V1 mission)`) which strict YAML rejects but the
# real loader accepts. This asserts what actually broke.
set -uo pipefail

SKILL_DIR=".claude/skills"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'

if [ ! -d "$SKILL_DIR" ]; then
	echo -e "${RED}✗ $SKILL_DIR not found (run from repo root)${RESET}"
	exit 1
fi

FOUND=0
COUNT=0

for f in "$SKILL_DIR"/*/SKILL.md; do
	[ -f "$f" ] || continue
	COUNT=$((COUNT + 1))
	slug=$(basename "$(dirname "$f")")

	if [ "$(head -1 "$f")" != "---" ]; then
		echo -e "${RED}✗ $f: no YAML frontmatter (must start with ---)${RESET}"
		echo "    Without it the loader uses the H1 heading as the description,"
		echo "    so this skill under-triggers instead of failing loudly."
		FOUND=1
		continue
	fi

	# Frontmatter must be CLOSED by a bare --- before we read anything from it;
	# without this the body would be scanned for name:/description: too, and an
	# unterminated block would pass the gate.
	fm_end=$(awk 'NR>1 && /^---[[:space:]]*$/{print NR; exit}' "$f")
	if [ -z "$fm_end" ]; then
		echo -e "${RED}✗ $f: frontmatter is unterminated (no closing ---)${RESET}"
		FOUND=1
		continue
	fi

	# Tested BEFORE the sed: a "2,1p" range clamps to printing line 2 rather than
	# nothing, so an empty block would otherwise be misreported as a missing name.
	if [ "$fm_end" -le 2 ]; then
		echo -e "${RED}✗ $f: frontmatter block is empty${RESET}"
		FOUND=1
		continue
	fi

	fm=$(sed -n "2,$((fm_end - 1))p" "$f")

	name=$(printf '%s\n' "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -1)
	desc=$(printf '%s\n' "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -1)

	if [ -z "$name" ]; then
		echo -e "${RED}✗ $f: frontmatter has no non-empty 'name:'${RESET}"
		FOUND=1
	elif [ "$name" != "$slug" ]; then
		echo -e "${RED}✗ $f: name '$name' does not match directory '$slug'${RESET}"
		FOUND=1
	fi

	if [ -z "$desc" ]; then
		echo -e "${RED}✗ $f: frontmatter has no non-empty 'description:'${RESET}"
		echo "    The description is the trigger text -- an empty one means the skill"
		echo "    is effectively unreachable."
		FOUND=1
	fi
done

if [ "$COUNT" -eq 0 ]; then
	echo -e "${RED}✗ no SKILL.md files found under $SKILL_DIR -- the gate is vacuous${RESET}"
	exit 1
fi

if [ "$FOUND" -eq 1 ]; then
	echo ""
	echo -e "${YELLOW}Fix: give each SKILL.md a --- delimited frontmatter block with"
	echo -e "'name:' (kebab-case, matching its directory) and a 'description:' that"
	echo -e "says what the skill does AND when to use it.${RESET}"
	exit 1
fi

echo -e "${GREEN}✓ all $COUNT skills have frontmatter with a matching name and a description${RESET}"
