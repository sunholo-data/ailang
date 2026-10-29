#!/bin/bash
# Hermetic bash 3.2 arms. Default mode needs no Go; --writer takes a real Go
# test-executable wrapper supplied by TestQuorumArtifactShellSuite.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
WRITER=""; ARM=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --writer) WRITER="$2"; shift 2 ;;
    --arm) ARM="$2"; shift 2 ;;
    *) echo "bad argument: $1" >&2; exit 2 ;;
  esac
done
fixture=$(mktemp -d)
fixture=$(cd "$fixture" && pwd -P)
export HOME="$fixture/home"
mkdir -p "$HOME"
unset MISSION_QUORUM_ARTIFACT_DIR AILANG_AGENT_POLICY MISSION_DRIVER_ROOT MISSION_REPO MISSION_CONTROL_ACTIVE
# A mission controller exports the whole MISSION_* profile (MISSION_NAME selects
# which profile the extracted driver block sources), so scrub every one, not a list.
for _v in $(compgen -e | grep '^MISSION_' || true); do unset "$_v"; done
fail() { echo "FAIL ${CURRENT_ARM:-setup}: $*" >&2; exit 1; }
expect_rc() {
  local want="$1" got
  shift
  if "$@" >"$HOME/out" 2>"$HOME/err"; then got=0; else got=$?; fi
  [ "$got" = "$want" ] || fail "rc=$got want=$want: $(cat "$HOME/err")"
}
extract() {
  local file="$1" label="$2" out="$3" count
  count=$(grep -c "^# --- $label START ---$" "$file" || true)
  [ "$count" = 1 ] || fail "opening marker count=$count in $file"
  count=$(grep -c "^# --- $label END ---$" "$file" || true)
  [ "$count" = 1 ] || fail "closing marker count=$count in $file"
  awk -v start="# --- $label START ---" -v end="# --- $label END ---" '$0==start {inside=1;next} $0==end {inside=0} inside {print}' "$file" > "$out"
  [ -s "$out" ] || fail "empty extraction: $file"
}
setup() {
  export MISSION_REPO=sunholo-data/ailang
  unset MISSION_QUORUM_ARTIFACT_DIR AILANG_AGENT_POLICY MISSION_DRIVER_ROOT MISSION_CONTROL_ACTIVE
  mkdir -p "$HOME/repo/tools/launchd/lib" "$HOME/bin"
  cp "$ROOT/tools/launchd/lib/quorum-artifacts.sh" "$HOME/repo/tools/launchd/lib/" || fail helper
  # No git writes: the fixture seam answers only a read-only root query.
  cat > "$HOME/bin/git" <<'GIT'
#!/bin/bash
[ "${QUORUM_GIT_FAIL:-0}" = 0 ] || exit 1
printf '%s\n' "$QUORUM_FIXTURE_ROOT"
GIT
  chmod +x "$HOME/bin/git"
  export PATH="$HOME/bin:$PATH" QUORUM_FIXTURE_ROOT="$HOME/repo"
  . "$ROOT/tools/launchd/lib/quorum-artifacts.sh" || fail source
  cd "$HOME/repo"
  export MISSION_DRIVER_ROOT="$HOME/repo"
}
blocks() {
  local copy="$1"
  extract "$ROOT/$copy/skills/mission-control/resources/gate-2-pick.md" 'QUORUM READER' "$HOME/reader.sh"
  . "$HOME/reader.sh"
}
json_at() { mkdir -p "$1"; printf '{"doc":"doc.md","synthesis":{"verdict":"pass"},"reviewers":[]}\n' > "$1/doc-fixed.json"; }
local_dir() { printf '%s/.ailang/state/mission-quorum' "$HOME/repo"; }
repo_namespace_missions() {
  local mission
  for mission in v1 docs motoko fleet; do
    unset MISSION_QUORUM_ARTIFACT_DIR
    MISSION_NAME="$mission" mc_quorum_init || fail init
    [ "$MISSION_QUORUM_ARTIFACT_DIR" = "$HOME/.ailang/state/mission-quorum" ] || fail namespace
    [ ! -e "$MISSION_QUORUM_ARTIFACT_DIR" ] || fail "init mkdir"
  done
}
repo_namespace_isolation() {
  mc_quorum_init || fail init
  json_at "$MISSION_QUORUM_ARTIFACT_DIR"
  unset MISSION_QUORUM_ARTIFACT_DIR
  MISSION_REPO=sunholo-data/ailang-world mc_quorum_init || fail world
  [ "$MISSION_QUORUM_ARTIFACT_DIR" = "$HOME/.ailang/state/mission-quorum-world" ] || fail world-path
  mc_quorum_reader_dirs "$HOME/repo" || fail dirs
  expect_rc 1 mc_quorum_lookup doc
  unset MISSION_QUORUM_ARTIFACT_DIR
  MISSION_REPO=other/repo mc_quorum_init || fail other
  [ "$MISSION_QUORUM_ARTIFACT_DIR" = "$HOME/.ailang/state/mission-quorum-repos/other/repo" ] || fail other-path
  mc_quorum_reader_dirs "$HOME/repo" || fail dirs
  expect_rc 1 mc_quorum_lookup doc
}
invalid_repo_and_home() {
  local repo
  for repo in '' owner/ /repo ./. ../repo owner/.. owner/repo/extra owner/../repo 'bad name/repo'; do
    expect_rc 2 env MISSION_REPO="$repo" /bin/bash -c '. "$1"; mc_quorum_init' bash "$ROOT/tools/launchd/lib/quorum-artifacts.sh"
  done
  expect_rc 2 env HOME=relative MISSION_REPO=owner/repo /bin/bash -c '. "$1"; mc_quorum_init' bash "$ROOT/tools/launchd/lib/quorum-artifacts.sh"
}
invalid_and_explicit_override() {
  local override
  for override in relative '~/dir' '   '; do
    MISSION_QUORUM_ARTIFACT_DIR="$override" expect_rc 2 mc_quorum_init
  done
  MISSION_QUORUM_ARTIFACT_DIR="$HOME/absolute with spaces"
  MISSION_REPO=invalid HOME=relative mc_quorum_init || fail explicit
  [ "$MISSION_QUORUM_ARTIFACT_DIR" = "$HOME/absolute with spaces" ] || fail bytes
  [ ! -e "$MISSION_QUORUM_ARTIFACT_DIR" ] || fail mkdir
}
gate2_extracted_active_matrix() {
  local copy value
  for copy in .claude .agents; do
    blocks "$copy"
    export MISSION_CONTROL_ACTIVE=1
    unset MISSION_QUORUM_ARTIFACT_DIR
    expect_rc 2 mc_quorum_gate2 doc
    export MISSION_QUORUM_ARTIFACT_DIR=''
    expect_rc 2 mc_quorum_gate2 doc
    export MISSION_QUORUM_ARTIFACT_DIR="$HOME/shared"
    for value in '' relative "$HOME/missing"; do
      MISSION_DRIVER_ROOT="$value" expect_rc 2 mc_quorum_gate2 doc
    done
    mkdir -p "$HOME/no-helper"
    MISSION_DRIVER_ROOT="$HOME/no-helper" expect_rc 2 mc_quorum_gate2 doc
    chmod 000 "$HOME/repo"
    expect_rc 2 mc_quorum_gate2 doc
    chmod 700 "$HOME/repo"
    chmod 000 "$HOME/repo/tools/launchd/lib/quorum-artifacts.sh"
    expect_rc 2 mc_quorum_gate2 doc
    chmod 600 "$HOME/repo/tools/launchd/lib/quorum-artifacts.sh"
    unset MISSION_REPO
    expect_rc 1 mc_quorum_gate2 doc
    for value in ' ' "$HOME/policy-secret"; do
      AILANG_AGENT_POLICY="$value" expect_rc 2 mc_quorum_gate2 doc
    done
    export MISSION_DRIVER_ROOT="$HOME/no-helper"
    # Source failure, not merely missing/readability: a readable helper returning 9.
    mkdir -p "$HOME/no-helper/tools/launchd/lib"
    echo 'return 9' > "$HOME/no-helper/tools/launchd/lib/quorum-artifacts.sh"
    expect_rc 2 mc_quorum_gate2 doc
    export MISSION_DRIVER_ROOT="$HOME/repo"
  done
}
lookup_error_matrix() {
  mc_quorum_reader_dirs "$HOME/repo" || fail dirs
  expect_rc 1 mc_quorum_lookup doc
  mkdir -p "$(local_dir)"
  expect_rc 1 mc_quorum_lookup doc
  echo bad > "$(local_dir)/doc-bad.json"
  expect_rc 2 mc_quorum_lookup doc
  rm "$(local_dir)/doc-bad.json"
  chmod 000 "$(local_dir)"
  expect_rc 2 mc_quorum_lookup doc
  chmod 700 "$(local_dir)"
  json_at "$(local_dir)"
  chmod 000 "$(local_dir)/doc-fixed.json"
  expect_rc 2 mc_quorum_lookup doc
  chmod 600 "$(local_dir)/doc-fixed.json"
  MISSION_QUORUM_ARTIFACT_DIR="$HOME/not-directory"
  echo x > "$MISSION_QUORUM_ARTIFACT_DIR"
  mc_quorum_reader_dirs "$HOME/repo" || fail dirs
  expect_rc 2 mc_quorum_lookup doc
  blocks .claude
  QUORUM_GIT_FAIL=1 expect_rc 2 mc_quorum_gate2 doc
}
extraction_is_nonvacuous() {
  blocks .claude; blocks .agents
  cmp "$ROOT/.claude/skills/mission-control/resources/gate-2-pick.md" "$ROOT/.agents/skills/mission-control/resources/gate-2-pick.md" || fail mirror
  [ -r "$ROOT/tools/launchd/lib/quorum-artifacts.sh" ] || fail helper
}
driver_block() {
  extract "$ROOT/tools/launchd/mission-control.sh" 'QUORUM ARTIFACT INIT' "$HOME/driver.sh"
  local profiles secrets block binary legacy
  profiles=$(grep -n 'mission-${MISSION_NAME}.env' "$ROOT/tools/launchd/mission-control.sh" | head -1 | cut -d: -f1)
  secrets=$(grep -n 'secrets.env" ]' "$ROOT/tools/launchd/mission-control.sh" | head -1 | cut -d: -f1)
  block=$(grep -n '^# --- QUORUM ARTIFACT INIT START' "$ROOT/tools/launchd/mission-control.sh" | cut -d: -f1)
  binary=$(grep -n 'exec ailang mission iterate' "$ROOT/tools/launchd/mission-control.sh" | head -1 | cut -d: -f1)
  legacy=$(grep -n 'CODEX CONTROLLER EXEC START' "$ROOT/tools/launchd/mission-control.sh" | head -1 | cut -d: -f1)
  [ "$profiles" -lt "$secrets" ] && [ "$secrets" -lt "$block" ] && [ "$block" -lt "$binary" ] && [ "$block" -lt "$legacy" ] || fail ordering
}
driver_export_before_both_launches() {
  driver_block
  local layer
  for layer in inherited profile secrets; do
    (
      export MC_DRIVER_ROOT="$HOME/repo"
      # Extract the actual profile and secrets source lines, not a parallel implementation.
      awk '/^\[ -n "\$\{MISSION_PROFILE:-\}"/,/export MISSION_NAME MISSION_REPO MISSION_DOC/' "$ROOT/tools/launchd/mission-control.sh" > "$HOME/profile.sh"
      awk '/^\[ -f "\$HOME\/\.config\/ailang\/mission-\$\{MISSION_NAME\}.env" \]/{print;getline;print}' "$ROOT/tools/launchd/mission-control.sh" >> "$HOME/profile.sh"
      awk '/^\[ -f "\$HOME\/\.config\/ailang\/secrets.env" \]/{print}' "$ROOT/tools/launchd/mission-control.sh" > "$HOME/secrets.sh"
      export MISSION_QUORUM_ARTIFACT_DIR="$HOME/inherited path"
      mkdir -p "$HOME/.config/ailang"
      if [ "$layer" != inherited ]; then printf 'MISSION_QUORUM_ARTIFACT_DIR="%s"\n' "$HOME/profile path" > "$HOME/.config/ailang/mission-v1.env"; fi
      if [ "$layer" = secrets ]; then printf 'MISSION_QUORUM_ARTIFACT_DIR="%s"\n' "$HOME/secrets path" > "$HOME/.config/ailang/secrets.env"; fi
      . "$HOME/profile.sh"; . "$HOME/secrets.sh"
      log() { echo "$*"; }
      . "$HOME/driver.sh"
      [ "$(/usr/bin/printenv MISSION_QUORUM_ARTIFACT_DIR)" = "$HOME/$layer path" ] || fail export
    ) || fail "$layer"
  done
  # Start with an UNEXPORTED assignment: removing export must kill this positive arm.
  (
    unset MISSION_QUORUM_ARTIFACT_DIR
    MISSION_QUORUM_ARTIFACT_DIR="$HOME/unexported path"
    MC_DRIVER_ROOT="$HOME/repo"; log() { :; }
    . "$HOME/driver.sh"
    [ "$(/usr/bin/printenv MISSION_QUORUM_ARTIFACT_DIR)" = "$HOME/unexported path" ] || fail unexported
  ) || fail export
  expect_rc 2 /bin/bash -c 'MC_DRIVER_ROOT="$1"; MISSION_QUORUM_ARTIFACT_DIR=relative; log() { echo "$*"; }; . "$2"; echo launch > "$3"' bash "$HOME/repo" "$HOME/driver.sh" "$HOME/launches"
  [ ! -e "$HOME/launches" ] || fail launched
}
codex_actual_forwarding() {
  . "$ROOT/tools/launchd/lib/codex-env-args.sh"
  export MISSION_QUORUM_ARTIFACT_DIR="$HOME/path with spaces" AILANG_AGENT_POLICY=' '
  [ -n "${AILANG_AGENT_POLICY:-}" ] || fail marker-presence
  # Actual legacy call site, with a fake capture function and no provider probe.
  awk '/# --- CODEX CONTROLLER EXEC START ---/{inside=1;next} /# --- CODEX CONTROLLER EXEC END ---/{inside=0} inside {print}' "$ROOT/tools/launchd/mission-control.sh" > "$HOME/codex.sh"
  [ -s "$HOME/codex.sh" ] || fail extraction
  echo fi >> "$HOME/codex.sh"
  CONTROLLER_PROVIDER=codex LOG="$HOME/codex.log" MODEL=fake REPO="$HOME/repo" PROMPT=fake
  mc_codex_exec() { printf '%s\n' "$@" > "$HOME/argv"; }
  . "$HOME/codex.sh"; wait
  grep -F "shell_environment_policy.set.MISSION_QUORUM_ARTIFACT_DIR=\"$HOME/path with spaces\"" "$HOME/argv" >/dev/null || fail forwarding
  if grep -F 'shell_environment_policy.set.AILANG_AGENT_POLICY' "$HOME/argv" >/dev/null; then fail marker-claim; fi
}
contaminated_host_startup() {
  driver_block; blocks .claude
  local marker override
  for marker in ' ' "$HOME/secret-policy"; do
    for override in "$HOME/valid" invalid; do
      expect_rc 2 env AILANG_AGENT_POLICY="$marker" MISSION_QUORUM_ARTIFACT_DIR="$override" /bin/bash -c 'MC_DRIVER_ROOT="$1"; log() { echo "$*"; }; . "$2"; echo launch > "$3"' bash "$HOME/repo" "$HOME/driver.sh" "$HOME/launches"
      grep -F 'local confinement' "$HOME/out" "$HOME/err" >/dev/null || fail diagnostic
      if grep -F 'secret-policy' "$HOME/out" "$HOME/err" >/dev/null; then fail leaked-policy; fi
      [ ! -e "$HOME/launches" ] || fail launched
      MISSION_CONTROL_ACTIVE=1 AILANG_AGENT_POLICY="$marker" MISSION_QUORUM_ARTIFACT_DIR="$override" expect_rc 2 mc_quorum_gate2 doc
    done
  done
}
standalone_policy_reader() {
  blocks .claude
  mkdir -p "$HOME/repo/sandbox/tools/launchd/lib"
  cp "$ROOT/tools/launchd/lib/quorum-artifacts.sh" "$HOME/repo/sandbox/tools/launchd/lib/"
  json_at "$(local_dir)" # outside sandbox must never be read
  cd "$HOME/repo/sandbox"
  unset MISSION_DRIVER_ROOT
  local marker override
  for marker in ' ' "$HOME/secret-policy"; do
    for override in "$HOME/shared" invalid; do
      AILANG_AGENT_POLICY="$marker" MISSION_QUORUM_ARTIFACT_DIR="$override" expect_rc 1 mc_quorum_gate2 doc
      grep -F "$HOME/repo/sandbox/.ailang/state/mission-quorum" "$HOME/err" >/dev/null || fail effective
      grep -F suppressed "$HOME/err" >/dev/null || fail suppressed
      if grep -F 'secret-policy' "$HOME/err" >/dev/null; then fail leak; fi
    done
  done
}
bash32_and_suite_reach() {
  # /bin/bash is 3.2 only on macOS; Linux CI runs this suite under bash 5.
  if [ "$(uname -s)" = Darwin ]; then
    [ "$BASH_VERSINFO" = 3 ] && [ "${BASH_VERSINFO[1]}" = 2 ] || fail "expected bash 3.2"
  else
    echo "SKIP bash32 version assertion: /bin/bash is $BASH_VERSION on $(uname -s)"
  fi
  /bin/bash -n "$ROOT/tools/launchd/lib/quorum-artifacts.sh" || fail syntax
  make -C "$ROOT" -n test-launchd-drivers > "$HOME/recipes"
  # Reached = exactly one argument of the run-suites.sh line (which launches every
  # suite through suite-env.sh — test_suite_env.sh arm F pins that).
  [ "$(awk '/\/bin\/bash tools\/launchd\/lib\/run-suites\.sh / { for (i = 1; i <= NF; i++) if ($i == "tools/launchd/test_quorum_artifacts.sh") n++ } END { print n + 0 }' "$HOME/recipes")" = 1 ] || fail reach
}
attended_absent_and_errors() {
  local copy
  for copy in .claude .agents; do
    blocks "$copy"
    unset MISSION_DRIVER_ROOT MISSION_REPO
    expect_rc 1 mc_quorum_gate2 doc
    cd "$HOME/repo"
    MISSION_DRIVER_ROOT=relative expect_rc 2 mc_quorum_gate2 doc
    QUORUM_GIT_FAIL=1 expect_rc 2 mc_quorum_gate2 doc
    json_at "$(local_dir)"
    chmod 000 "$(local_dir)/doc-fixed.json"
    expect_rc 2 mc_quorum_gate2 doc
    chmod 600 "$(local_dir)/doc-fixed.json"
    echo broken > "$(local_dir)/doc-fixed.json"
    expect_rc 2 mc_quorum_gate2 doc
    rm "$(local_dir)/doc-fixed.json"
  done
}
cross_mode_errors_dominate() {
  export MISSION_QUORUM_ARTIFACT_DIR="$HOME/shared"
  local first second
  first="$MISSION_QUORUM_ARTIFACT_DIR"; second="$(local_dir)"
  json_at "$first"; json_at "$second"
  mc_quorum_reader_dirs "$HOME/repo" || fail dirs
  expect_rc 0 mc_quorum_lookup doc
  echo broken > "$second/doc-bad.json"
  expect_rc 2 mc_quorum_lookup doc
  rm "$second/doc-bad.json"
  echo broken > "$first/doc-bad.json"
  expect_rc 2 mc_quorum_lookup doc
  rm "$first/doc-bad.json"
  chmod 000 "$second"
  expect_rc 2 mc_quorum_lookup doc
  chmod 700 "$second"
  rm "$first/doc-fixed.json"
  expect_rc 0 mc_quorum_lookup doc
}
generated_docs_and_mirror() {
  grep -F '{EnvMissionQuorumArtifactDir, "", AreaMission,' "$ROOT/internal/config/mission.go" >/dev/null || fail registry
  grep -F '\`MISSION_QUORUM_ARTIFACT_DIR\`' "$ROOT/docs/docs/reference/env-vars.md" >/dev/null || grep -F 'MISSION_QUORUM_ARTIFACT_DIR' "$ROOT/docs/docs/reference/env-vars.md" >/dev/null || fail docs
  /bin/bash "$ROOT/tools/launchd/sync-agents-skills.sh" --check || fail sync
}
fixture_home_and_observations() {
  mc_quorum_init || fail init
  [ "$MISSION_QUORUM_ARTIFACT_DIR" = "$HOME/.ailang/state/mission-quorum" ] || fail home
  [ ! -e "$HOME/.ailang" ] || fail write
}
write_real() {
  [ -n "$WRITER" ] && [ -x "$WRITER" ] || fail "real writer required"
  "$WRITER" > "$HOME/writer-out" 2> "$HOME/writer-err" || fail "writer: $(cat "$HOME/writer-err")"
  WRITTEN=$(sed -n 's/^ARTIFACT=//p' "$HOME/writer-out")
  [ -n "$WRITTEN" ] && [ -r "$WRITTEN" ] || fail writer-path
}
cross_tree_durable() {
  export MISSION_QUORUM_ARTIFACT_DIR="$HOME/durable path"
  mkdir -p "$HOME/treeA" "$HOME/treeB" "$HOME/cloneC"
  cd "$HOME/treeA"; write_real
  cp "$WRITTEN" "$HOME/expected"
  rmdir "$HOME/treeA" # remove only disposable writer tree; evidence survives
  local tree
  for tree in treeB cloneC; do
    cd "$HOME/$tree"
    mc_quorum_reader_dirs "$HOME/$tree" || fail dirs
    mc_quorum_lookup doc || fail durable
    [ "${MC_QUORUM_MATCHES[0]}" = "$WRITTEN" ] || fail path
    cmp "$WRITTEN" "$HOME/expected" || fail bytes
  done
}
attended_no_exports() {
  unset MISSION_QUORUM_ARTIFACT_DIR MISSION_DRIVER_ROOT MISSION_REPO AILANG_AGENT_POLICY MISSION_CONTROL_ACTIVE
  cd "$HOME/repo"; write_real
  cp "$WRITTEN" "$HOME/expected"
  mkdir -p "$HOME/repo/nested"; cd "$HOME/repo/nested"
  blocks .claude; mc_quorum_gate2 doc || fail attended
  [ "$(pwd -P)" = "$HOME/repo" ] || fail cwd
  cmp "${MC_QUORUM_MATCHES[0]}" "$HOME/expected" || fail bytes
}
cross_mode_fixed_order() {
  unset MISSION_QUORUM_ARTIFACT_DIR
  write_real; cp "$WRITTEN" "$HOME/local-bytes"
  export MISSION_QUORUM_ARTIFACT_DIR="$HOME/shared" MISSION_CONTROL_ACTIVE=1
  blocks .claude; mc_quorum_gate2 doc 2> "$HOME/order" || fail local-to-mission
  cmp "${MC_QUORUM_MATCHES[0]}" "$HOME/local-bytes" || fail bytes
  [ "$(sed -n '1p' "$HOME/order")" = "quorum consult: $HOME/shared" ] || fail order-first
  [ "$(sed -n '2p' "$HOME/order")" = "quorum consult: $(local_dir)" ] || fail order-second
  write_real; cp "$WRITTEN" "$HOME/host-bytes"
  rm "$(local_dir)"/*.json
  unset MISSION_CONTROL_ACTIVE MISSION_DRIVER_ROOT MISSION_REPO
  blocks .agents; mc_quorum_gate2 doc || fail mission-to-attended
  cmp "${MC_QUORUM_MATCHES[0]}" "$HOME/host-bytes" || fail bytes
  rm "$MISSION_QUORUM_ARTIFACT_DIR"/*.json
  expect_rc 1 mc_quorum_gate2 doc
}
# These are child-process behavior checks, including an oracle negative control:
# removing only AFTER_CALL from a real success must make the same checker reject it.
zsh_quorum_result_ok() {
  local child_rc="$1" out="$2" expected="$3"
  [ "$child_rc" = 0 ] || return 1
  [ "$(grep -cx 'AFTER_CALL' "$out" || true)" = 1 ] || return 1
  sed '/^AFTER_CALL$/d' "$out" | cmp -s "$expected" -
}
zsh_quorum_case() {
  local mode="$1" entry="${2:-lookup}" options="${3:-ignore}" copy="${4:-.claude}"
  local artifact want_rc=0 child_rc=0
  export MISSION_QUORUM_ARTIFACT_DIR="$HOME/export with spaces"
  mkdir -p "$MISSION_QUORUM_ARTIFACT_DIR" "$(local_dir)"
  artifact="$MISSION_QUORUM_ARTIFACT_DIR/doc-x-2026.json"
  case "$mode" in
    local) artifact="$(local_dir)/doc-x-2026.json" ;;
    absent) artifact=''; want_rc=1 ;;
  esac
  if [ -n "$artifact" ]; then
    printf '{"doc":"doc-x.md","synthesis":{"verdict":"pass"},"reviewers":[]}\n' > "$artifact"
  fi
  if [ "$entry" = gate2 ]; then
    extract "$ROOT/$copy/skills/mission-control/resources/gate-2-pick.md" 'QUORUM READER' "$HOME/reader.sh"
    export MISSION_CONTROL_ACTIVE=1
  fi
  /bin/zsh -f -c '
    setopt nomatch; unsetopt null_glob
    if [ "$2" = gate2 ]; then
      . "$3" || exit 80
      mc_quorum_gate2 doc-x
    else
      . "$1" || exit 80
      mc_quorum_reader_dirs "$MISSION_DRIVER_ROOT" || exit 80
      mc_quorum_lookup doc-x
    fi
    fn_rc=$?
    printf "%s\n" AFTER_CALL
    if [ "$4" = check ]; then
      [[ -o nomatch && ! -o null_glob ]] || { print -u2 OPTIONS_LEAKED; exit 81; }
    fi
    printf "RC=%s\nCOUNT=%s\n" "$fn_rc" "${#MC_QUORUM_MATCHES[@]}"
    for match in "${MC_QUORUM_MATCHES[@]}"; do printf "MATCH=%s\n" "$match"; done
  ' zsh "$ROOT/tools/launchd/lib/quorum-artifacts.sh" "$entry" "$HOME/reader.sh" "$options" > "$HOME/zsh-out" 2> "$HOME/zsh-err" || child_rc=$?
  {
    printf 'RC=%s\n' "$want_rc"
    if [ -n "$artifact" ]; then printf 'COUNT=1\nMATCH=%s\n' "$artifact"; else printf 'COUNT=0\n'; fi
  } > "$HOME/zsh-expected"
  zsh_quorum_result_ok "$child_rc" "$HOME/zsh-out" "$HOME/zsh-expected" || fail "child rc=$child_rc, result/AFTER_CALL mismatch: $(cat "$HOME/zsh-out" "$HOME/zsh-err")"
  sed '/^AFTER_CALL$/d' "$HOME/zsh-out" > "$HOME/zsh-no-marker"
  if zsh_quorum_result_ok "$child_rc" "$HOME/zsh-no-marker" "$HOME/zsh-expected"; then
    fail 'missing-marker negative control accepted'
  fi
}
zsh_quorum_export_only() { zsh_quorum_case export; }
zsh_quorum_local_only() { zsh_quorum_case local; }
zsh_quorum_absent() { zsh_quorum_case absent; }
zsh_quorum_options() {
  zsh_quorum_case export lookup check
  rm "$MISSION_QUORUM_ARTIFACT_DIR/doc-x-2026.json"
  zsh_quorum_case absent lookup check
}
zsh_gate2_export_only() {
  local copy
  for copy in .claude .agents; do zsh_quorum_case export gate2 ignore "$copy"; done
}

run_arm() {
  CURRENT_ARM="$1"
  case "$1" in
    zsh_*)
      if [ ! -x /bin/zsh ]; then
        [ "$(uname -s)" != Darwin ] || fail 'no /bin/zsh on Darwin'
        echo "SKIP (no /bin/zsh): $1"; return 0
      fi ;;
  esac
  ( export HOME="$fixture/$1"; mkdir -p "$HOME"; setup || exit 1; "$1" ) || fail "$1"
  echo "PASS $1"
}
if [ -n "$ARM" ]; then run_arm "$ARM"; exit; fi
for arm in zsh_quorum_export_only zsh_quorum_local_only zsh_quorum_absent zsh_quorum_options zsh_gate2_export_only repo_namespace_missions repo_namespace_isolation invalid_repo_and_home invalid_and_explicit_override gate2_extracted_active_matrix lookup_error_matrix extraction_is_nonvacuous driver_export_before_both_launches codex_actual_forwarding contaminated_host_startup standalone_policy_reader bash32_and_suite_reach attended_absent_and_errors cross_mode_errors_dominate generated_docs_and_mirror fixture_home_and_observations; do run_arm "$arm"; done
if [ -n "$WRITER" ]; then
  for arm in cross_tree_durable attended_no_exports cross_mode_fixed_order; do run_arm "$arm"; done
fi
