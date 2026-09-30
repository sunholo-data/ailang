#!/bin/bash
# Copy the mission-loop skills from .claude/skills (the source of truth) to
# .agents/skills, verbatim. pi and codex read .agents (AGENTS.md: "Skills live in
# .agents/skills/"), so a stale copy there is what those harnesses actually run.
#
# Measured 2026-09-29: .agents/skills/mission-control had NO resources/ directory
# (no gate files at all) and a SKILL.md 989 diff lines behind, last synced 2026-08-28;
# the sprint-* copies still carried the 2026-07-28 "claude -> Codex" rename, which
# points at .Codex/skills/... paths that do not exist; mission-brief and
# mission-loop-change were missing. Verbatim is correct: the .claude/... paths
# resolve for any agent reading this repo.
#
# Scope is the fleet's (ticket skills:agents-copies-stale): mission-* and sprint-*.
# Other skills carry .agents-only edits (model-manager, design-doc-creator) and need
# a reconciliation, not an overwrite. test_agents_skills_sync.sh enforces this list.
#
#   tools/launchd/sync-agents-skills.sh           copy
#   tools/launchd/sync-agents-skills.sh --check   exit 1 if any copy differs
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SKILLS="mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator"
rc=0
for s in $SKILLS; do
  src="$ROOT/.claude/skills/$s"; dst="$ROOT/.agents/skills/$s"
  [ -d "$src" ] || { echo "sync-agents-skills: missing source $src" >&2; rc=1; continue; }
  if [ "${1:-}" = "--check" ]; then
    diff -rq "$src" "$dst" >/dev/null 2>&1 || { echo "STALE: .agents/skills/$s differs from .claude/skills/$s"; rc=1; }
  else
    mkdir -p "$dst" && rsync -a --delete "$src/" "$dst/" || rc=1
  fi
done
[ "${1:-}" = "--check" ] && [ "$rc" -eq 0 ] && echo "agents skills in sync ($SKILLS)"
exit $rc
