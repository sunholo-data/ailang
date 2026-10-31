#!/bin/bash
L="agent-inbox ailang-packages benchmark-manager benchmark-runner builtin-developer cli-doc-maintainer cloud-cascade-debug collaboration-hub coordinator-helper design-doc-creator docs-sync eval-analyzer eval-gap-finder gemini-cli-helper github-issue-triage headless-runner local-model-onboarding local-ollama-eval model-manager motoko-analyzer parser-developer post-release pr-monitor prompt-manager release-manager trace-debugger use-ailang"
n_=0
for n in $L; do n_=$((n_+1))
 refs=$(grep -rlE "skills/$n\b|/skill:$n\b|\`$n\` skill|$n skill" .claude/skills/mission-* .claude/skills/sprint-* tools/launchd design_docs/*-mission.md missions 2>/dev/null | wc -l | tr -d ' ')
 fa=$(cd .agents/skills/$n && find . -type f | sort); fc=$(cd .claude/skills/$n && find . -type f | sort)
 onlyA=$(comm -23 <(echo "$fa") <(echo "$fc") | grep -c .); onlyC=$(comm -13 <(echo "$fa") <(echo "$fc") | grep -c .)
 dA=$(diff -r .claude/skills/$n .agents/skills/$n 2>/dev/null | grep -c '^>'); dC=$(diff -r .claude/skills/$n .agents/skills/$n 2>/dev/null | grep -c '^<')
 echo "$n refs=$refs onlyAgentsFiles=$onlyA onlyClaudeFiles=$onlyC agentsOnlyLines=$dA claudeOnlyLines=$dC lastC=$(git log -1 --format=%cs -- .claude/skills/$n) lastA=$(git log -1 --format=%cs -- .agents/skills/$n)"
done
echo "enumerated=$n_"
