#!/bin/bash
# D-FLEET-12: findings are report flags, not a record-commit blocker.
set -u
ALLOWED_OPENAI='gpt-6.1-sol gpt-6-luna'
file=''
entry=''
usage() { echo "usage: $0 --file <mission-log.md> [--entry N]" >&2; exit 2; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --file)
      [ "$#" -ge 2 ] && [ -z "$file" ] && [ -n "$2" ] || usage
      file="$2"; shift 2 ;;
    --entry)
      [ "$#" -ge 2 ] && [ -z "$entry" ] || usage
      case "$2" in ''|*[!0-9]*) usage ;; esac
      entry="$2"; shift 2 ;;
    *) usage ;;
  esac
done
[ -n "$file" ] || usage
if [ ! -f "$file" ] || [ ! -r "$file" ]; then
  echo "instrument failure: missing/unreadable file: $file" >&2
  exit 2
fi

# Buffer only findings from the selected entry. With no --entry each heading
# replaces the selection, so the last entry in file order wins (not the max N).
awk -v wanted="$entry" -v allowed="$ALLOWED_OPENAI" '
function canonical(id) {
  id = tolower(id)
  sub(/^[[:alnum:]_]+:/, "", id)
  sub(/^openai\//, "", id)
  gsub(/[-._]/, "", id)
  return id
}
BEGIN {
  n = split(allowed, models, " ")
  for (i = 1; i <= n; i++) allow[canonical(models[i])] = 1
}
/^## [0-9]+ — / {
  active = (wanted == "" || $2 == wanted)
  if (active) {
    seenEntry = 1; seenBlock = 0; inBlock = 0; findings = ""; count = 0
  }
  next
}
active {
  if (!seenBlock && /^\*\*Routing evidence\*\*/) {
    seenBlock = 1; inBlock = 1
  } else if (/^\*\*/) {
    inBlock = 0
  }
  if (!inBlock) next
  rest = $0
  while (match(tolower(rest), /gpt[-0-9a-z.]*/)) {
    token = substr(rest, RSTART, RLENGTH)
    rest = substr(rest, RSTART + RLENGTH)
    sub(/[-.]+$/, "", token)
    key = canonical(token)
    if (!(key in allow)) {
      findings = findings FILENAME ":" FNR ": " token "\n"
      count++
    }
  }
}
END {
  if (!seenEntry) {
    print "instrument failure: missing entry " wanted > "/dev/stderr"
    exit 2
  }
  if (!seenBlock) {
    print "instrument failure: missing Routing evidence block" > "/dev/stderr"
    exit 2
  }
  printf "%s", findings
  exit (count > 0 ? 1 : 0)
}
' "$file"
