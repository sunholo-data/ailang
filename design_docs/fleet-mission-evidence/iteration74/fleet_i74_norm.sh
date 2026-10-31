#!/bin/bash
norm() { tr 'A-Z' 'a-z' < "$1" | sed -e 's#\.agents/#.claude/#g' -e 's/claude code/X/g' -e 's/claude-code/X/g' -e 's/claude/X/g' -e 's/codex/X/g'; }
mech=0; real=0
for n in $(cut -d' ' -f1 /tmp/fleet_i74_drift.txt | grep -v '^enumerated'); do
  r=0
  for f in $( (cd .claude/skills/$n; find . -type f; cd ../../../.agents/skills/$n; find . -type f) | sort -u); do
    a=.claude/skills/$n/$f; b=.agents/skills/$n/$f
    if [ ! -f "$a" ] || [ ! -f "$b" ]; then r=$((r+1)); continue; fi
    cmp -s "$a" "$b" && continue
    d=$(diff <(norm "$a") <(norm "$b") | grep -c '^[<>]'); r=$((r+d))
  done
  if [ $r -eq 0 ]; then mech=$((mech+1)); echo "RENAME-ONLY $n"; else real=$((real+1)); echo "CONTENT $n residual=$r"; fi
done
echo "rename-only=$mech content=$real"
