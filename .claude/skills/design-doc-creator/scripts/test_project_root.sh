#!/usr/bin/env bash
set -euo pipefail
skill=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
scratch=$(mktemp -d /tmp/design-skill-root.XXXXXX)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/plugin/skills" "$scratch/project/nested" "$scratch/bin" "$scratch/not-a-repo"
cp -R "$skill" "$scratch/plugin/skills/design-doc-creator"
git init -q "$scratch/project"
printf '#!/bin/sh\necho "1. design_docs/reference.md (0.10)"\n' > "$scratch/bin/ailang"
chmod +x "$scratch/bin/ailang"
export PATH="$scratch/bin:$PATH"
shared="$scratch/plugin/skills/design-doc-creator/scripts"
(cd "$scratch/project/nested" && bash "$shared/create_planned_doc.sh" m-shared-root > "$scratch/create.log")
if [ ! -f "$scratch/project/design_docs/planned/m-shared-root.md" ]; then
  cat "$scratch/create.log"
  echo 'FAIL: shared skill wrote outside the target Git worktree' >&2
  exit 1
fi
(cd "$scratch/project/nested" && bash "$shared/move_to_implemented.sh" m-shared-root v0_37_1 > "$scratch/move.log")
[ -f "$scratch/project/design_docs/implemented/v0_37_1/m-shared-root.md" ]
# Positive control: local layout and empty search results still create a document.
printf '#!/bin/sh\nexit 0\n' > "$scratch/bin/ailang"
mkdir -p "$scratch/project/.claude/skills"
cp -R "$skill" "$scratch/project/.claude/skills/design-doc-creator"
(cd "$scratch/project" && bash .claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-local-root > "$scratch/local.log")
[ -f "$scratch/project/design_docs/planned/m-local-root.md" ]
# Missing target context must fail, not write relative to the plugin installation.
if (cd "$scratch/not-a-repo" && bash "$shared/create_planned_doc.sh" m-refused > "$scratch/refusal.log" 2>&1); then
  echo 'FAIL: accepted a non-Git target' >&2; exit 1
fi
grep -q 'target Git worktree' "$scratch/refusal.log"
[ ! -d "$scratch/design_docs" ]
[ ! -d "$scratch/plugin/design_docs" ]
[ ! -d "$scratch/not-a-repo/design_docs" ]
echo 'PASS: shared and local skill roots, nested cwd, move, and missing-target refusal'
