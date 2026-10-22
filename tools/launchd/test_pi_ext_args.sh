#!/bin/bash
# test_pi_ext_args.sh — lib/pi-ext-args.sh: each pi extension loads exactly once.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/lib/pi-ext-args.sh"
PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; }
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT

mkdir -p "$T/plain"
[ -z "$(_mc_pi_ext_args "$T/plain")" ] && ok "no .pi/extensions -> no flags (discovery unchanged: world, stapledon)" \
  || bad "no .pi/extensions must print nothing"

mkdir -p "$T/repo/.pi/extensions"; : > "$T/repo/.pi/extensions/a.ts"; : > "$T/repo/.pi/extensions/b.ts"; : > "$T/repo/.pi/extensions/README.md"
out=$(_mc_pi_ext_args "$T/repo")
[ "$(printf '%s\n' "$out" | head -1)" = "--no-extensions" ] && ok "repo extensions -> discovery off" || bad "first flag must be --no-extensions"
[ "$(printf '%s\n' "$out" | grep -c '^-e$')" = 2 ] && ok "one -e per .ts" || bad "expected 2 -e flags: $out"
printf '%s\n' "$out" | grep -q 'README' && bad "non-.ts files must not load" || ok "README not loaded"

mkdir -p "$T/empty/.pi/extensions"
[ "$(_mc_pi_ext_args "$T/empty")" = "--no-extensions" ] && ok "empty dir -> discovery off, nothing named" || bad "empty dir"

# Wiring: every pi launch site uses it (or, for the probe, disables discovery outright).
grep -q 'pi --mode json --no-session --no-tools --no-extensions' "$HERE/lib/lane-probe.sh" && ok "probe disables discovery" || bad "probe not wired"
grep -q '_mc_pi_ext_args "\$REPO"' "$HERE/mission-control.sh" && ok "pi controller wired" || bad "pi controller not wired"
grep -q '_mc_pi_ext_args "\$WORKDIR"' "$HERE/../../scripts/mission_pi_run.sh" && ok "role runner wired" || bad "role runner not wired"

ROOT=$(cd "$HERE/../.." && pwd)
skill_roots=${PI_EXT_ARGS_SKILL_ROOTS-"$ROOT/.claude/skills/mission-control
$ROOT/.agents/skills/mission-control"}
sweep_ok=1; instrument_ok=1; root_count=0
while IFS= read -r skill_root || [ -n "$skill_root" ]; do
  root_count=$((root_count+1))
  route="$skill_root/resources/gate-3-route.md"
  if [ -r "$route" ] && grep -Fq 'pi --mode json --no-session --no-tools --no-extensions' "$route"; then
    ok "$skill_root: probe disables discovery"
  else
    bad "$skill_root: probe must include --no-tools --no-extensions ($route)"
  fi
  # Validate one Bash fence, with initialization before the launch and both -e flags after it.
  if [ -r "$route" ] && awk '
    function finish() {
      if (source_line && init > source_line && launch > init && sandbox > launch && fence > sandbox) valid=1
      source_line=init=launch=sandbox=fence=0
    }
    /^[[:space:]]*```/ {
      finish()
      bash_block=(!in_block && $0 ~ /^[[:space:]]*```bash[[:space:]]*$/)
      in_block=!in_block
      next
    }
    bash_block {
      if (index($0, "source \"$REPO/tools/launchd/lib/pi-ext-args.sh\"")) source_line=FNR
      if (index($0, "PI_EXT_ARGS=(); while IFS= read -r _l; do PI_EXT_ARGS+=(\"$_l\"); done < <(_mc_pi_ext_args \"$WT\")")) init=FNR
      if (index($0, "pi --mode json --no-session ${PI_EXT_ARGS[@]+\"${PI_EXT_ARGS[@]}\"}")) launch=FNR
      if (index($0, "-e \"$REPO/tools/pi-extensions/sandbox/index.ts\"")) sandbox=FNR
      if (index($0, "-e \"$REPO/tools/pi-extensions/worktree-fence.ts\"")) fence=FNR
    }
    END { finish(); exit !valid }
  ' "$route"; then
    ok "$skill_root: raw sandbox snippet wired"
  else
    bad "$skill_root: raw sandbox snippet needs helper initialization and launch splice before both -e flags ($route)"
  fi

  # Count independently of safety; missing inputs cannot silently make the sweep vacuous.
  files=(); inputs_ok=1
  if [ ! -d "$skill_root" ] || [ ! -r "$skill_root" ] || [ ! -x "$skill_root" ]; then
    echo "UNREADABLE OR MISSING ROOT: $skill_root"
    inputs_ok=0
  fi
  for file in "$skill_root/SKILL.md" "$skill_root"/resources/*.md; do
    if [ ! -f "$file" ] || [ ! -r "$file" ]; then
      echo "UNREADABLE OR MISSING FILE: $file"
      inputs_ok=0
    else
      files+=("$file")
    fi
  done
  if [ "$inputs_ok" -eq 0 ]; then sweep_ok=0; instrument_ok=0; fi
  launches=0
  if [ "${#files[@]}" -gt 0 ]; then
    # A launch is `pi`, any run of blanks, `--mode`, blanks, `json`. Each launch LINE must itself
    # disable discovery or carry the PI_EXT_ARGS splice; a splice elsewhere in the fence does not count.
    launches=$(awk '/pi[[:space:]]+--mode[[:space:]]+json/ { n++ } END { print n+0 }' ${files[@]+"${files[@]}"})
    if ! awk '
      /pi[[:space:]]+--mode[[:space:]]+json/ &&
        !index($0, "--no-extensions") && !index($0, "${PI_EXT_ARGS[@]+\"${PI_EXT_ARGS[@]}\"}") {
        printf "%s:%d: %s\n", FILENAME, FNR, $0
        unsafe=1
      }
      END { exit unsafe }
    ' ${files[@]+"${files[@]}"}; then sweep_ok=0; fi
  fi
  echo "LAUNCH COUNT: $skill_root: $launches launch lines"
  if [ "$launches" -lt 2 ]; then
    echo "INSTRUMENT FAILURE: $skill_root: $launches launch lines (minimum 2)"
    instrument_ok=0
  fi
done <<< "$skill_roots"
if [ "$root_count" -eq 0 ]; then
  echo "INSTRUMENT FAILURE: no skill roots selected"
  sweep_ok=0; instrument_ok=0
fi
[ "$sweep_ok" -eq 1 ] && ok "skill launch sweep safety" || bad "skill launch sweep safety"
[ "$instrument_ok" -eq 1 ] && ok "skill launch count instrument" || bad "skill launch count instrument"
# Match exact helper bytes and rc separately from the child completion marker.
zsh_pi_result_ok() {
  local child_rc="$1" out="$2" expected="$3"
  [ "$child_rc" = 0 ] || return 1
  [ "$(grep -cx 'AFTER_CALL' "$out" || true)" = 1 ] || return 1
  sed '/^AFTER_CALL$/d' "$out" | cmp -s "$expected" -
}
zsh_pi_arm() {
  local arm="$1" dir="$T/$1 with spaces" child_rc=0 name
  if [ ! -x /bin/zsh ]; then
    if [ "$(uname -s)" = Darwin ]; then bad "$arm (no /bin/zsh)"; else echo "SKIP (no /bin/zsh): $arm"; fi
    return
  fi
  mkdir -p "$dir/.pi/extensions"
  if [ "$arm" = zsh_pi_two_ts ]; then
    : > "$dir/.pi/extensions/a.ts"; : > "$dir/.pi/extensions/b.ts"; : > "$dir/.pi/extensions/README.md"
  fi
  (
    unset AILANG_AGENT_POLICY
    for name in $(compgen -e | grep '^MISSION_' || true); do unset "$name"; done
    /bin/zsh -f -c '
      setopt nomatch; unsetopt null_glob
      . "$1" || exit 80
      _mc_pi_ext_args "$2" > "$3"
      fn_rc=$?
      printf "%s\n" AFTER_CALL
      if [ "$4" = zsh_pi_options ]; then
        [[ -o nomatch && ! -o null_glob ]] || { print -u2 OPTIONS_LEAKED; exit 81; }
      fi
      cat "$3"
      printf "RC=%s\n" "$fn_rc"
    ' zsh "$HERE/lib/pi-ext-args.sh" "$dir" "$dir/helper-out" "$arm"
  ) > "$dir/zsh-out" 2> "$dir/zsh-err" || child_rc=$?
  {
    printf '%s\n' --no-extensions
    if [ "$arm" = zsh_pi_two_ts ]; then printf '%s\n' -e "$dir/.pi/extensions/a.ts" -e "$dir/.pi/extensions/b.ts"; fi
    printf 'RC=0\n'
  } > "$dir/expected"
  if ! zsh_pi_result_ok "$child_rc" "$dir/zsh-out" "$dir/expected"; then
    bad "$arm: child rc=$child_rc, result/AFTER_CALL mismatch: $(cat "$dir/zsh-out" "$dir/zsh-err")"
    return
  fi
  # Feed actual successful output without ONLY its marker into the same checker.
  sed '/^AFTER_CALL$/d' "$dir/zsh-out" > "$dir/no-marker"
  if zsh_pi_result_ok "$child_rc" "$dir/no-marker" "$dir/expected"; then
    bad "$arm: missing-marker negative control accepted"
    return
  fi
  ok "$arm"
}
for arm in zsh_pi_empty zsh_pi_two_ts zsh_pi_options; do zsh_pi_arm "$arm"; done

echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
