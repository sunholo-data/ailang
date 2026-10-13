#!/bin/bash
# Phase 3a measurement only. Bash 3.2 + BSD userland; Python 3 parses JSON, never raw-token grep.
set -eu
umask 077
pin='' v1='' only=all reps=3 fence=on dry=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --pin|--v1|--only|--reps|--fence)
      [ "$#" -ge 2 ] || { echo "Missing value: $1" >&2; exit 2; }
      case "$1" in --pin) pin=$2;; --v1) v1=$2;; --only) only=$2;; --reps) reps=$2;; --fence) fence=$2;; esac; shift 2;;
    --dry-run) dry=1; shift;; *) echo "Unknown option: $1" >&2; exit 2;;
  esac
done
case "$only:$fence" in claude:on|claude:off|codex:on|codex:off|pi:on|pi:off|all:on|all:off) :;; *) exit 2;; esac
case "$reps" in ''|*[!0-9]*|0) echo 'reps must be positive' >&2; exit 2;; esac
case "$pin:$v1" in /*:/*) :;; *) echo 'Absolute --pin and --v1 required' >&2; exit 2;; esac
pin=$(cd "$pin" && pwd -P)
pin_marker_root=$pin; user_root=$HOME; world_parent=/private/tmp; ku=${CODEX_HOME:-$HOME/.codex}
if [ -n "${SPIKE_TEST_ROOT:-}" ]; then
  [ "$dry" -eq 1 ] || { echo 'SPIKE_TEST_ROOT requires --dry-run' >&2; exit 2; }
  test_root=$(cd "$SPIKE_TEST_ROOT" && pwd -P)
  case "$test_root/" in "$pin/"*) :;; *) echo 'Test root must be inside pin' >&2; exit 2;; esac
  pin_marker_root=$test_root/pin; user_root=$test_root/user; v1=$test_root/v1; world_parent=$test_root/tmp; ku=$user_root/.codex
fi
v1=$(cd "$v1" && pwd -P)
[ "$pin" != "$v1" ] || { echo 'v1 must differ from pin' >&2; exit 2; }
for dependency in openssl python3 shasum ps; do command -v "$dependency" >/dev/null || exit 2; done
runid=$(date -u +%s)-$(openssl rand -hex 12)
base=$pin/design_docs/fleet-mission-evidence/iteration34
base=$(cd "$base" && pwd -P); case "$base/" in "$pin/"*) :;; *) echo 'Evidence root escapes pin' >&2; exit 2;; esac
[ ! -L "$base/runs" ] || { echo 'Evidence runs parent must not be a symlink' >&2; exit 2; }
mkdir -p "$base/runs"
run=$base/runs/$runid; mkdir "$run"
mkdir "$run/audit" "$run/calls" "$run/runtime"
ledger=(); parents=(); audited=(); markers=(); locs=()
arms=0; verdicts=0; controls_ok=0; controls_total=0; failed=0; safety=OK; completed=0; active='' watch=''
claude-sub() { env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN claude "$@"; }
export -f claude-sub
snapshot() {
  local p=$1 i
  for ((i=0;i<${#parents[@]};i++)); do [ "${parents[i]}" != "$p" ] || return 0; done
  i=${#parents[@]}; parents[i]=$p; audited[i]=0
  printf '%s\n' "$p" > "$run/audit/$i.parent"
  if [ -d "$p" ]; then
    (cd "$p" && LC_ALL=C ls -1A) > "$run/audit/$i.before"
    { ls -ld "$p"; for entry in "$p"/* "$p"/.[!.]* "$p"/..?*; do
      if [ -L "$entry" ]; then ls -ld "$entry"; readlink "$entry"; fi
    done; } > "$run/audit/$i.symlinks"
    shasum "$run/audit/$i.before" > "$run/audit/$i.before.sha"; wc -l < "$run/audit/$i.before" > "$run/audit/$i.before.count"
  else printf 'MISSING\n' > "$run/audit/$i.before"; fi
}
audit() {
  local i=$1 p=${parents[$1]} result=OK
  [ "${audited[i]}" -eq 0 ] || return 0
  if [ -d "$p" ]; then (cd "$p" && LC_ALL=C ls -1A) > "$run/audit/$i.after"
  else printf 'MISSING\n' > "$run/audit/$i.after"; fi
  shasum "$run/audit/$i.after" > "$run/audit/$i.after.sha"; wc -l < "$run/audit/$i.after" > "$run/audit/$i.after.count"
  if ! cmp -s "$run/audit/$i.before" "$run/audit/$i.after"; then
    # controller i34: a shared parent (/private/tmp) changes under concurrent writers; only a delta naming THIS run is ours
    diff "$run/audit/$i.before" "$run/audit/$i.after" | grep '^[<>]' > "$run/audit/$i.delta" || :
    # judge N2 (i34): a REMOVED pre-existing entry is a safety failure unless it is another spike run's
    # marker, or the parent is the shared world parent where foreign temp files come and go.
    if grep -Fq "$runid" "$run/audit/$i.delta"; then result=SAFETY_FAILURE; safety=SAFETY_FAILURE; failed=1
    elif [ "$p" != "$world_parent" ] && grep '^<' "$run/audit/$i.delta" | grep -vq '^< zz-fleet-spike-'; then
      result=SAFETY_FAILURE; safety=SAFETY_FAILURE; failed=1
    else result=CONCURRENT_CHANGE; fi
  fi
  audited[i]=1
  printf 'AUDIT parent=%s status=%s before_sha=%s after_sha=%s before_lines=%s after_lines=%s\n' "$p" "$result" \
    "$(shasum < "$run/audit/$i.before" | awk '{print $1}')" "$(shasum < "$run/audit/$i.after" | awk '{print $1}')" \
    "$(wc -l < "$run/audit/$i.before" | tr -d ' ')" "$(wc -l < "$run/audit/$i.after" | tr -d ' ')" | tee -a "$run/audit.log"
}
cleanup() {
  local rc=$? i j p
  trap - EXIT INT TERM; set +e
  [ "$completed" -eq 1 ] || failed=1
  if [ -n "$active" ]; then kill -TERM -- "-$active" 2>/dev/null; kill -KILL -- "-$active" 2>/dev/null; wait "$active" 2>/dev/null; fi
  if [ -n "$watch" ]; then kill "$watch" 2>/dev/null; wait "$watch" 2>/dev/null; fi
  for ((i=${#ledger[@]}-1;i>=0;i--)); do
    p=${ledger[$i]}; [ -n "$p" ] || continue
    for ((j=0;j<${#parents[@]};j++)); do
      case "${parents[$j]}/" in "$p/"*) audit "$j";; esac
    done
    rm -rf -- "$p" # MUTATION_DRILL: owned-directory cleanup
    [ ! -e "$p" ] && [ ! -L "$p" ] || { safety=SAFETY_FAILURE; failed=1; }
  done
  for ((j=0;j<${#parents[@]};j++)); do audit "$j"; done
  printf 'SUMMARY runid=%s arms=%s verdicts=%s controls_ok=%s/%s safety=%s\n' "$runid" "$arms" "$verdicts" "$controls_ok" "$controls_total" "$safety" | tee "$run/summary.txt"
  [ "$failed" -eq 0 ] && [ "$rc" -eq 0 ] || exit 1
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
own() { snapshot "$(dirname "$1")"; mkdir "$1"; ledger[${#ledger[@]}]=$1; printf '%s\n' "$1" >> "$run/ownership.txt"; }
cu=$user_root/.claude
for parent in "$cu/skills" "$user_root/.agents/skills" "$user_root/.codex/skills" "$user_root/.pi/agent/skills" \
  "$ku/skills" "$v1" "$v1/.claude/skills" "$pin/.claude/skills" "$pin/.agents/skills" "$pin_marker_root/.claude/skills" "$pin_marker_root/.agents/skills" "$world_parent"; do snapshot "$parent"; done
scratch=$run/runtime/zz-fleet-spike-$runid-scratch; own "$scratch"
cs=$scratch/CS; ks=$scratch/KS; psroot=$scratch/PS; xp=$scratch/XP
mkdir -p "$cs/skills" "$ks/skills" "$psroot/skills" "$xp/.claude/skills" "$scratch/tmp" "$scratch/cache"
# controller i34: pi needs its provider table; link (never copy/write) the real models.json/auth.json into the owned PS root
for f in models.json auth.json; do [ ! -f "$user_root/.pi/agent/$f" ] || ln -s "$user_root/.pi/agent/$f" "$psroot/$f"; done
world=$world_parent/zz-fleet-spike-$runid-world; own "$world"
if [ -z "${SPIKE_TEST_ROOT:-}" ]; then
  ancestor=$world
  while :; do
    [ ! -e "$ancestor/.git" ] && [ ! -L "$ancestor/.git" ] || { echo "World inside checkout ancestry: $ancestor"; exit 1; }
    [ ! -e "$ancestor/.claude/skills" ] && [ ! -L "$ancestor/.claude/skills" ] || { echo "World ancestry has skills: $ancestor"; exit 1; }
    [ "$ancestor" != / ] || break; ancestor=$(dirname "$ancestor")
  done
fi
{ printf 'pin=%s\nv1=%s\nworld=%s\nfence=%s\ndry=%s\n' "$pin" "$v1" "$world" "$fence" "$dry"
  python3 -c 'import os; print("env_names="+",".join(sorted(os.environ)))'
  for checkout in "$pin" "$v1"; do
    printf 'checkout=%s\n' "$checkout"
    env GIT_OPTIONAL_LOCKS=0 git -C "$checkout" rev-parse HEAD; env GIT_OPTIONAL_LOCKS=0 git -C "$checkout" diff --stat; env GIT_OPTIONAL_LOCKS=0 git -C "$checkout" status --short
  done
} > "$run/metadata.txt" 2> "$run/metadata.stderr" || :
# The deny fixture is new and inside scratch; the profile allows only its sibling runtime subtree.
profile=$run/fence.sb
{ printf '(version 1)\n(allow default)\n(deny file-write*)\n'
  for writable in "$scratch" "$run/calls" "$world"; do printf '(allow file-write* (subpath "%s"))\n' "$writable"; done
  printf '(deny file-write* (subpath "%s/deny-fixture"))\n(allow file-write* (literal "/dev/null"))\n' "$scratch"
} > "$profile"
fence_ok=1
if [ "$dry" -eq 0 ] && [ "$fence" = on ]; then
  # shellcheck disable=SC2016 # The fenced child expands its own positional argument.
  if ! /usr/bin/sandbox-exec -f "$profile" /bin/bash -c ': > "$1"' bash "$scratch/allow-fixture" 2> "$run/fence.stderr" || \
    /usr/bin/sandbox-exec -f "$profile" /bin/bash -c ': > "$1"' bash "$scratch/deny-fixture" 2>> "$run/fence.stderr"; then
    fence_ok=0; failed=1; echo 'INSTRUMENT_FAILURE fence self-test' >&2
  fi
fi
# A stopped leader must not leave a child writer alive before marker cleanup.
bounded() {
  local out=$1 err=$2 start deadline pid pgid runner_pgid rc end
  shift 2; start=$(date +%s); deadline=$((start+120)); runner_pgid=$(ps -o pgid= -p $$ | tr -d ' ')
  set -m
  (while [ ! -e "$out.start" ]; do sleep 0.05; done; exec "$@") < /dev/null > "$out" 2> "$err" & pid=$!
  pgid=$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ') || pgid=
  if [ "$pgid" != "$pid" ] || [ "$pgid" = "$runner_pgid" ]; then
    kill "$pid" 2>/dev/null || :; wait "$pid" 2>/dev/null || :; set +m; call_rc=125
    printf 'start=%s deadline=%s end=%s rc=125 pgid=UNSAFE\n' "$start" "$deadline" "$(date +%s)" > "$out.timing"; return
  fi
  active=$pgid; : > "$out.start"
  (while kill -0 -- "-$pgid" 2>/dev/null; do
    now=$(date +%s)
    if [ "$now" -ge "$((deadline-5))" ]; then echo TIMEOUT > "$out.timeout"; kill -TERM -- "-$pgid" 2>/dev/null || :; fi
    if [ "$now" -ge "$deadline" ]; then kill -KILL -- "-$pgid" 2>/dev/null || :; break; fi
    sleep 1
  done) < /dev/null & watch=$!
  if wait "$pid"; then rc=0; else rc=$?; fi
  if [ -e "$out.timeout" ]; then while [ "$(date +%s)" -lt "$deadline" ]; do sleep 1; done; fi
  kill -TERM -- "-$pgid" 2>/dev/null || :; kill -KILL -- "-$pgid" 2>/dev/null || :
  kill "$watch" 2>/dev/null || :; wait "$watch" 2>/dev/null || :
  while kill -0 -- "-$pgid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then echo 'Process group not reaped' >&2; exit 1; fi; sleep 1
  done
  active=; watch=; set +m; end=$(date +%s)
  printf 'start=%s deadline=%s end=%s rc=%s pgid=%s\n' "$start" "$deadline" "$end" "$rc" "$pgid" > "$out.timing"
  call_rc=$rc; [ ! -e "$out.timeout" ] || call_rc=124
}
for cli in claude codex pi; do
  command -v "$cli" > "$run/$cli.executable" 2>/dev/null || :
  [ "$dry" -eq 0 ] && [ "$fence_ok" -eq 1 ] || continue
  pre=(); [ "$fence" = off ] || pre=(/usr/bin/sandbox-exec -f "$profile")
  cmd=("$cli"); [ "$cli" != claude ] || cmd=(/bin/bash -c 'claude-sub "$@"' bash)
  for flag in --version --help; do
    bounded "$run/calls/$cli.$flag.stdout" "$run/calls/$cli.$flag.stderr" ${pre[@]+"${pre[@]}"} "${cmd[@]}" "$flag"
    nl -ba "$run/calls/$cli.$flag.stdout" > "$run/$cli.$flag.numbered"
  done
  if [ "$cli" = codex ]; then
    bounded "$run/calls/codex.exec-help.stdout" "$run/calls/codex.exec-help.stderr" ${pre[@]+"${pre[@]}"} codex exec --help
    nl -ba "$run/calls/codex.exec-help.stdout" > "$run/codex.exec-help.numbered"
  fi
done
plant() {
  local location=$1 parent=$2 leaf token resource
  leaf=$parent/$name; locs[${#locs[@]}]=$location; markers[${#markers[@]}]=$leaf
  [ -d "$parent" ] || { available=0; printf 'MISSING location=%s parent=%s\n' "$location" "$parent" >> "$run/findings.txt"; return; }
  own "$leaf"; mkdir "$leaf/resources"
  token=$(openssl rand -hex 24); resource=$(openssl rand -hex 24)
  if grep -Fq "$token" "$run/tokens.tsv" || grep -Fq "$resource" "$run/tokens.tsv" || [ "$token" = "$resource" ]; then exit 1; fi
  printf '%s\t%s\t%s\t%s\n' "$location" "$token" "$resource" "$leaf" >> "$run/tokens.tsv"
  printf '%s\t%s\t%s\t%s\n' "$location" "$token" "$resource" "$leaf" >> "$run/current.tsv"
  cat > "$leaf/SKILL.md" <<EOF
---
name: $name
description: Use only when explicitly requested by the unique name $name.
---
Your skill_token is $token. Read resources/probe.md relative to THIS SKILL.md directory.
Return ONLY JSON with keys skill_token and resource_token containing the two exact tokens. No operational actions.
EOF
  printf 'Your resource_token is %s. Return it with the skill token as instructed.\n' "$resource" > "$leaf/resources/probe.md"
  shasum "$leaf/SKILL.md" "$leaf/resources/probe.md" >> "$run/marker-hashes.txt"
}
: > "$run/tokens.tsv"
# Final-answer and provenance parser. Raw stdout/stderr stay unchanged; no prompt/tool-result scoring.
score() {
  python3 - "$cli" "$armdir" "$run/current.tsv" "$run/tokens.tsv" "$kind" "$expected" "$name" "$cwd" <<'PY'
import json, pathlib, sys, shlex, re
cli, directory, current, inventory, kind, expected, name, cwd = sys.argv[1:]
p = pathlib.Path(directory); events = []
for line in (p/'stdout').read_text().splitlines():
    try: events.append(json.loads(line))
    except ValueError: pass
final = ''; valid = False; billing = ''; calls = []
if cli == 'C':
    results = [e for e in events if e.get('type') == 'result']
    if results:
        e = results[-1]; final = e.get('result', ''); valid = not e.get('is_error', False) and e.get('subtype') == 'success'
        billing = next((x.get('apiKeySource', '') for x in events if x.get('type') == 'system' and x.get('subtype') == 'init'), '')  # controller i34: hook events carry no apiKeySource
        (p/'auth-observation.json').write_text(json.dumps({'apiKeySource':billing, 'usage':e.get('usage', {})}))
    for e in events:
        if e.get('type') == 'assistant':
            calls += [(x.get('name',''), x.get('input',{})) for x in e.get('message',{}).get('content',[]) if x.get('type') == 'tool_use']
elif cli == 'K':
    if (p/'final.txt').is_file(): final = (p/'final.txt').read_text(); valid = any(e.get('type') == 'turn.completed' for e in events)
    for e in events:
        x = e.get('item', {})
        if e.get('type') == 'item.completed' and x.get('type') == 'command_execution': calls.append(('command', x.get('command','')))
        elif e.get('type') == 'item.completed' and x.get('type') in ('mcp_tool_call','web_search'): calls.append(('forbidden',x))
else:
    terminal = [e.get('message',{}) for e in events if e.get('type') == 'message_end' and e.get('message',{}).get('role') == 'assistant']
    if terminal:
        e = terminal[-1]; final = ''.join(x.get('text','') for x in e.get('content',[]) if x.get('type') == 'text')
        ends = [x for x in events if x.get('type') == 'agent_end']  # controller i34: pi 0.85.1 emits a trailing agent_settled
        valid = e.get('stopReason') == 'stop' and bool(ends) and not ends[-1].get('willRetry', False)
    calls = [(e.get('toolName',''), e.get('args',{})) for e in events if e.get('type') == 'tool_execution_start']
(p/'final.txt').write_text(final)
rows = [l.split('\t') for l in pathlib.Path(current).read_text().splitlines()]
allowed = {str(pathlib.Path(r[3])/f) for r in rows for f in ('SKILL.md','resources/probe.md')}
# controller i34: repo context files (AGENTS.md tells codex to read CLAUDE.md) are not skill search and cannot carry run tokens
allowed |= {str((pathlib.Path(cwd)/f).resolve()) for f in ('CLAUDE.md','AGENTS.md')}
leak = False
for tool, args in calls:
    if tool.lower() == 'skill':
        requested = args.get('skill','').lstrip('/')
        prefix = name[:len('zz-fleet-spike-') + 35]  # zz-fleet-spike-<epoch>-<24 hex>
        # controller i34: a negative control asking the Skill tool for its own unplanted -n<k> name is correct behaviour
        leak |= requested != name and not (kind == 'negative' and requested.startswith(prefix + '-n'))
    elif tool.lower() in ('read',):
        target = args.get('file_path', args.get('path','')); leak |= target not in allowed and target != 'resources/probe.md'
    elif tool == 'command':
        try:
            words = shlex.split(args)
            if len(words) == 3 and words[0] in ('/bin/bash','/bin/zsh') and words[1] in ('-lc','-c'): words = shlex.split(words[2])
            pieces, segment = [], []
            for word in words + [';']:
                if word in ('&&',';'): pieces.append(segment); segment = []
                else: segment.append(word)
            read_cwd = cwd
            for words in pieces:
                if len(words) == 2 and words[0] == 'cd' and words[1] in {r[3] for r in rows}: read_cwd = words[1]; continue
                targets = []
                if words and words[0] == 'cat': targets = words[1:]
                elif len(words) == 4 and words[:2] == ['sed','-n'] and re.fullmatch(r'[0-9]+(?:,[0-9]+)?p', words[2]): targets = words[3:]
                elif len(words) == 4 and words[:2] == ['head','-n'] and words[2].isdigit(): targets = words[3:]
                leak |= not targets or any(str((pathlib.Path(read_cwd)/t).resolve()) not in allowed for t in targets)
        except (ValueError, IndexError): leak = True
    else: leak = True
(p/'provenance.json').write_text(json.dumps({'calls':calls, 'forbidden_search_or_path':leak}))
status, origin = 'INCONCLUSIVE', '-'
if kind == 'auth' and re.search(r'not logged in|not authenticated|login required|please (?:run .*)?log in', final, re.I): status = 'NOT_LOGGED_IN'
elif not valid: status = 'INSTRUMENT_FAILURE'
elif kind == 'auth':  # controller i34: tolerate <=5 output tokens (the API counts 3 for a one-character answer)
    status = 'AUTH_OK' if final.strip() == '1' and results[-1].get('usage',{}).get('output_tokens', 99) <= 5 and  billing in ('none','oauth','subscription') else 'INVALID_PROBE'
elif leak: status = 'INVALID_DISCOVERY'
elif final.strip() == 'ABSENT':
    tokens = [t for l in pathlib.Path(inventory).read_text().splitlines() for t in l.split('\t')[1:3]]
    status = 'ABSENT' if not any(t in final for t in tokens) else 'INCONCLUSIVE'
else:
    body = re.sub(r'^\s*```(?:json)?\s*|\s*```\s*$', '', final)  # controller i34: tolerate one markdown fence
    try: answer = json.loads(body)
    except ValueError: answer = {}
    if isinstance(answer, dict) and set(answer) == {'skill_token','resource_token'}:
        skills = [r[0] for r in rows if r[1] == answer['skill_token']]; resources = [r[0] for r in rows if r[2] == answer['resource_token']]
        if len(skills) == len(resources) == 1:
            status = 'MATCH' if skills == resources else 'RESOURCE_MISMATCH'; origin = skills[0]+'/'+resources[0]
if kind == 'positive' and (status != 'MATCH' or origin != expected+'/'+expected): status = 'INSTRUMENT_FAILURE'
if kind == 'negative' and status != 'ABSENT': status = 'INSTRUMENT_FAILURE'
print(status, origin)
PY
}
verdict() {
  local id=$1 status=$2 origin=$3 controls=$4 rc=$5
  [ "$dry" -eq 0 ] || controls=NOT_RUN
  printf 'VERDICT runid=%s arm=%s status=%s origin=%s controls=%s rc=%s fence=%s evidence=%s\n' \
    "$runid" "$id" "$status" "$origin" "$controls" "$rc" "$fence" "runs/$runid/calls/$id" | tee "$run/calls/$id/verdict.txt"
  verdicts=$((verdicts+1)); printf '%s\t%s\t%s\n' "$id" "$status" "$origin" >> "$run/verdicts.tsv"
}
call() {
  local id=$1 kind=$2 expected=$3 prompt=$4; shift 4
  armdir=$run/calls/$id; mkdir "$armdir"; arms=$((arms+1)); call_rc=0; status=NOT_RUN; origin=-
  argv=(); child_env=(env "TMPDIR=$scratch/tmp" "XDG_CACHE_HOME=$scratch/cache")
  case "$cli" in
    C) argv=(/bin/bash -c 'claude-sub "$@"' bash -p --model claude-haiku-5-5 --no-session-persistence --output-format stream-json --verbose)
       if [ "$kind" = auth ]; then argv=("${argv[@]}" --tools ''); else argv=("${argv[@]}" --tools 'Read,Skill' --allowedTools 'Read,Skill'); fi
       argv=("${argv[@]}" "$@" --permission-mode dontAsk --permission-prompts none --debug-file "$armdir/debug.log" "$prompt");  # controller i34: variadic --add-dir swallowed the prompt
       if [ "$root" = "$cu" ]; then child_env=(env -u CLAUDE_CONFIG_DIR "${child_env[@]:1}"); else child_env=("${child_env[@]}" "CLAUDE_CONFIG_DIR=$root"); fi;; # controller i34: real-config arms run with CLAUDE_CONFIG_DIR UNSET, as the driver does
    K) argv=(codex --no-daemon --ask-for-approval never exec --model gpt-6.1-sol --sandbox read-only --ephemeral --skip-git-repo-check --json --cd "$cwd" --output-last-message "$armdir/final.txt" "$prompt"); child_env=("${child_env[@]}" "CODEX_HOME=$root");;
    I) argv=(pi --print --model ollama/kimi-k3:cloud --mode json --no-session --tools 'read,grep,find,ls' --no-extensions --no-context-files --approve "$@" "$prompt"); child_env=("${child_env[@]}" "PI_CODING_AGENT_DIR=$psroot");;
  esac
  pre=(); [ "$fence" = off ] || pre=(/usr/bin/sandbox-exec -f "$profile")
  # shellcheck disable=SC2016 # Positional cwd/argv are interpreted inside the fenced child.
  execute=(${pre[@]+"${pre[@]}"} /bin/bash -c 'cd "$1" || exit; shift; exec "$@"' bash "$cwd" "${child_env[@]}" "${argv[@]}")
  printf 'cwd=%q\n' "$cwd" > "$armdir/argv.txt"; printf '%q ' "${execute[@]}" >> "$armdir/argv.txt"; printf '\n' >> "$armdir/argv.txt"
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$id" "$cli" "$cwd" "$root" "$kind" "$locations" | tee -a "$run/arm-table.tsv"
  : > "$armdir/stdout"; : > "$armdir/stderr"; : > "$armdir/final.txt"
  if [ "$dry" -eq 0 ]; then
    if [ "$available" -eq 0 ] || [ "$fence_ok" -eq 0 ]; then status=INSTRUMENT_FAILURE; call_rc=125
    else
      bounded "$armdir/stdout" "$armdir/stderr" "${execute[@]}"
      if score_line=$(score 2> "$armdir/parser.stderr"); then read -r status origin <<< "$score_line"; else status=INSTRUMENT_FAILURE; origin=-; fi
      if [ "$call_rc" -ne 0 ] && { [ "$kind" != auth ] || [ "$status" != NOT_LOGGED_IN ]; }; then status=INSTRUMENT_FAILURE; fi
    fi
    if [ "$status" = INSTRUMENT_FAILURE ] && [ "$fence" = on ]; then printf 'INSTRUMENT_FAILURE fence=on; inspect denied-operation evidence in stderr/debug/raw streams\n' > "$armdir/failure.txt"; fi
  fi
  if [ -e "$armdir/stdout.timing" ]; then cp "$armdir/stdout.timing" "$armdir/timing.txt"
  else printf 'start=NOT_RUN deadline=NOT_RUN end=NOT_RUN\n' > "$armdir/timing.txt"; fi
  printf '%s\n' "$call_rc" > "$armdir/rc.txt"
}
discovery() { printf 'For this measurement as mission controller, use the available skill %s. Follow its relative resource instruction and return only its JSON. If unavailable return ABSENT. Use only the skill catalog/tool and its declared files; do not search directories, enumerate other skills, or infer paths.' "$1"; }
group() {
  local id=$1 cli=$2 root=$3 cwd=$4 locations=$5 mode=$6; shift 6
  local location parent i phase control_status control_id negative_id group_ok=1 subject_status subject_origin subject_rc subject_id=$id subject_kind=subject subject_expected=-
  name=zz-fleet-spike-$runid-$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]'); negative=zz-fleet-spike-$runid-n$arms; available=1; markers=(); locs=(); : > "$run/current.tsv"
  # controller i34: K:H is the EFFECTIVE CODEX_HOME's skills dir (mission env: ~/.codex-missions), not ~/.codex
  for location in $locations; do
    case "$cli:$location" in C:P|K:L) parent=$pin_marker_root/.claude/skills;; C:U) parent=$cu/skills;; C:V) parent=$v1/.claude/skills;; C:X) parent=$xp/.claude/skills;; C:S) parent=$cs/skills;; K:A|I:A) parent=$pin_marker_root/.agents/skills;; K:B|I:B) parent=$user_root/.agents/skills;; K:H) parent=$ku/skills;; K:T) parent=$ks/skills;; I:T) parent=$psroot/skills;; esac
    plant "$location" "$parent"
  done
  if [ "$mode" = baseline ]; then subject_id=$id+; subject_kind=positive; subject_expected=${locations%% *}; fi
  control_extras=("$@")
  if [ "$mode" = baseline ] && [ "$cli" = I ]; then control_extras=(--skill "${markers[0]}/SKILL.md"); fi
  if [ "$mode" = explicit ] || [ "$mode" = no-skills ]; then
    [ "$mode" != no-skills ] || control_extras=(${control_extras[@]+"${control_extras[@]}"} --no-skills)
    control_extras=(${control_extras[@]+"${control_extras[@]}"} --skill "$pin_marker_root/.agents/skills/$name/SKILL.md")
  fi
  for phase in pre post; do
    for ((i=0;i<${#markers[@]};i++)); do
      control_id=$id-$phase-${locs[$i]}+
      call "$control_id" positive "${locs[$i]}" "Read ${markers[$i]}/SKILL.md, follow its relative resource instruction, return only its JSON." ${control_extras[@]+"${control_extras[@]}"}
      controls_total=$((controls_total+1)); control_status=$status
      if [ "$status" = MATCH ]; then controls_ok=$((controls_ok+1)); elif [ "$dry" -eq 0 ]; then group_ok=0; failed=1; fi
      verdict "$control_id" "$control_status" "$origin" control "$call_rc"
    done
    negative_id=$id-$phase-negative
    if [ "$mode" = baseline ] && [ "$phase" = pre ]; then negative_id=$id-; fi
    call "$negative_id" negative - "$(discovery "$negative")" ${control_extras[@]+"${control_extras[@]}"}
    controls_total=$((controls_total+1)); if [ "$status" = ABSENT ]; then controls_ok=$((controls_ok+1)); elif [ "$dry" -eq 0 ]; then group_ok=0; failed=1; fi
    verdict "$negative_id" "$status" "$origin" control "$call_rc"
    if [ "$phase" = pre ]; then
      prompt=$(discovery "$name"); extras=(${control_extras[@]+"${control_extras[@]}"})
      if [ "$mode" = baseline ] && [ "$cli" != I ]; then prompt="Read ${markers[0]}/SKILL.md, follow its relative resource instruction, return only its JSON."; fi
      if [ "$mode" = E ] && [ "${#markers[@]}" -gt 0 ]; then prompt="Read ${markers[0]}/SKILL.md, follow its relative resource instruction, return only its JSON."; fi
      call "$subject_id" "$subject_kind" "$subject_expected" "$prompt" ${extras[@]+"${extras[@]}"}
      if [ "$mode" = baseline ]; then controls_total=$((controls_total+1)); if [ "$status" = MATCH ]; then controls_ok=$((controls_ok+1)); elif [ "$dry" -eq 0 ]; then failed=1; group_ok=0; fi; fi
      subject_status=$status; subject_origin=$origin; subject_rc=$call_rc
    fi
  done
  [ "$group_ok" -eq 1 ] || subject_status=INSTRUMENT_FAILURE
  verdict "$subject_id" "$subject_status" "$subject_origin" "$group_ok" "$subject_rc"
  case "$subject_status" in NOT_RUN|MATCH|ABSENT) :;; *) failed=1;; esac
  for marker in ${markers[@]+"${markers[@]}"}; do
    owned=0; for ((i=0;i<${#ledger[@]};i++)); do [ "${ledger[$i]}" != "$marker" ] || owned=1; done
    [ "$owned" -eq 1 ] || continue
    rm -rf -- "$marker"
    [ ! -e "$marker" ] && [ ! -L "$marker" ] || { safety=SAFETY_FAILURE; failed=1; exit 1; }
    for ((i=0;i<${#ledger[@]};i++)); do [ "${ledger[$i]}" != "$marker" ] || ledger[i]=''; done
  done
}
if [ "$only" = all ] || [ "$only" = claude ]; then
  group C C "$cu" "$pin" 'P U' baseline
  group C-project C "$cu" "$pin" P D
  group C-user C "$cu" "$world" U D
  for ((rep=1;rep<=reps;rep++)); do
    group C-collision-$rep C "$cu" "$pin" 'P U' D
    group C-world-$rep C "$cu" "$world" 'U X' D --add-dir "$xp"
    group C-v1-$rep C "$cu" "$v1" 'P U V' D
  done
  group C-add-only C "$cu" "$world" X D --add-dir "$xp"
  group C-settings C "$cu" "$pin" 'P U' D --setting-sources ''
  cli=C; cwd=$world; locations=none; available=1; name=auth; : > "$run/current.tsv"
  baseline=INSTRUMENT_FAILURE; isolated=INSTRUMENT_FAILURE
  for auth in base isolated; do
    root=$cu; [ "$auth" != isolated ] || root=$cs
    call "C-auth-$auth" auth - 'Return only 1.'
    if [ "$auth" = isolated ] && [ "$baseline" != AUTH_OK ] && [ "$dry" -eq 0 ]; then status=INSTRUMENT_FAILURE; fi
    if [ "$auth" = base ]; then baseline=$status; else isolated=$status; fi
    verdict "C-auth-$auth" "$status" "$origin" auth "$call_rc"
    [ "$dry" -eq 1 ] || { [ "$status" = AUTH_OK ] || { [ "$auth" = isolated ] && [ "$baseline" = AUTH_OK ] && [ "$status" = NOT_LOGGED_IN ]; }; } || failed=1
  done
  if [ "$dry" -eq 1 ] || { [ "$baseline" = AUTH_OK ] && [ "$isolated" = AUTH_OK ]; }; then group C-synthetic C "$cs" "$pin" 'P S' D
  else
    # Plan every optional arm, but never perform inference following failed isolated auth.
    saved_dry=$dry; dry=1; group C-synthetic C "$cs" "$pin" 'P S' D; dry=$saved_dry
  fi
fi
if [ "$only" = all ] || [ "$only" = codex ]; then
  group K K "$ku" "$pin" A baseline
  group K-claude K "$ku" "$pin" L D
  group K-project K "$ku" "$pin" A D
  group K-agents-user K "$ku" "$world" B D
  group K-legacy-user K "$ku" "$world" H D
  group K-collision K "$ku" "$pin" 'A B H' D
  group K-by-path K "$ku" "$pin" 'A B H' E
  group K-home K "$ks" "$pin" 'A T' D
fi
if [ "$only" = all ] || [ "$only" = pi ]; then
  group I I "$psroot" "$pin" 'A B T' baseline
  group I-project I "$psroot" "$pin" A D
  group I-collision I "$psroot" "$pin" 'A B T' D
  group I-explicit I "$psroot" "$pin" 'A B T' explicit
  group I-world I "$psroot" "$world" 'A B T' explicit
  group I-no-skills I "$psroot" "$pin" 'A B T' no-skills
fi
printf 'COUNTS runid=%s (observations, no universal precedence claim)\n' "$runid"
LC_ALL=C sort "$run/verdicts.tsv" | awk -F '\t' '$1 ~ /-(pre|post)-/ || $1 ~ /^[CIK][+-]$/ {next} {a=$1; sub(/-[0-9]+$/,"",a); n[a" status="$2" origin="$3]++} END {for (k in n) print k, n[k]}' | tee "$run/counts.txt"
completed=1
