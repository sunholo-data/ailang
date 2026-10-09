#!/usr/bin/env bash
# Offline fixtures only: no real OAuth tokens or network calls.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT
export CODEX_INTERACTIVE_HOME="$TEST_ROOT/interactive" CODEX_MISSION_HOME="$TEST_ROOT/missions" CODEX_CLOUD_HOME="$TEST_ROOT/cloud"
fixture() {
  local dir=$1 refresh=$2 user=${3:-user-fixture} workspace=${4:-workspace-fixture} mode=${5:-chatgpt}
  mkdir -p "$dir"; chmod 700 "$dir"
  local payload
  payload=$(jq -nc --arg user "$user" --arg workspace "$workspace" '{sub:$user,"https://api.openai.com/auth":{chatgpt_user_id:$user,chatgpt_account_id:$workspace}}' | base64 | tr -d '\n=' | tr '+/' '-_')
  jq -n --arg token "fixture.$payload.fixture" --arg refresh "$refresh" --arg workspace "$workspace" --arg mode "$mode" '{auth_mode:$mode,tokens:{id_token:$token,access_token:"access-fixture",refresh_token:$refresh,account_id:$workspace}}' > "$dir/auth.json"
  chmod 600 "$dir/auth.json"
}
reset_fixtures() {
  rm -rf "$CODEX_INTERACTIVE_HOME" "$CODEX_MISSION_HOME" "$CODEX_CLOUD_HOME"
  fixture "$CODEX_INTERACTIVE_HOME" refresh-fixture-i
  fixture "$CODEX_MISSION_HOME" refresh-fixture-m
  fixture "$CODEX_CLOUD_HOME" refresh-fixture-c
}
expect() {
  local outcome=$1 label=$2 actual=0 output
  output=$(bash "$SCRIPT_DIR/check_codex_oauth_profiles.sh" 2>&1) || actual=$?
  if [[ $outcome == pass && $actual != 0 ]] || [[ $outcome == fail && $actual == 0 ]]; then
    printf 'FAIL test: %s\n%s\n' "$label" "$output"; exit 1
  fi
  if [[ $output == *fixture* || $output == *eyJ* ]]; then printf 'FAIL credential output\n'; exit 1; fi
  printf 'PASS test: %s\n' "$label"
}
reset_fixtures; expect pass distinct
fixture "$CODEX_CLOUD_HOME" refresh-fixture-m; expect fail copied_refresh
reset_fixtures; fixture "$CODEX_CLOUD_HOME" refresh-fixture-c other-user; expect fail different_account
reset_fixtures; fixture "$CODEX_CLOUD_HOME" refresh-fixture-c user-fixture other-workspace; expect fail different_workspace
reset_fixtures; fixture "$CODEX_CLOUD_HOME" refresh-fixture-c user-fixture workspace-fixture apikey; expect fail api_key_mode
reset_fixtures; chmod 644 "$CODEX_MISSION_HOME/auth.json"; expect fail file_permissions
reset_fixtures; chmod 755 "$CODEX_MISSION_HOME"; expect fail home_permissions
reset_fixtures; rm -rf "$CODEX_CLOUD_HOME"; ln -s "$CODEX_MISSION_HOME" "$CODEX_CLOUD_HOME"; expect fail symlink_home
reset_fixtures; rm "$CODEX_CLOUD_HOME/auth.json"; ln -s "$CODEX_MISSION_HOME/auth.json" "$CODEX_CLOUD_HOME/auth.json"; expect fail symlink_auth
reset_fixtures; printf '{broken' > "$CODEX_CLOUD_HOME/auth.json"; expect fail malformed_json
reset_fixtures; jq '.tokens.account_id="mismatch"' "$CODEX_CLOUD_HOME/auth.json" > "$TEST_ROOT/temp"; cat "$TEST_ROOT/temp" > "$CODEX_CLOUD_HOME/auth.json"; expect fail identity_mismatch
reset_fixtures; jq 'del(.tokens.refresh_token)' "$CODEX_CLOUD_HOME/auth.json" > "$TEST_ROOT/temp"; cat "$TEST_ROOT/temp" > "$CODEX_CLOUD_HOME/auth.json"; expect fail missing_refresh
reset_fixtures; CODEX_CLOUD_HOME="$CODEX_MISSION_HOME/." expect fail alias_home
# Attended helper: use mock binaries to verify sequencing and credential paths.
reset_fixtures
HELPER="$SCRIPT_DIR/provision_codex_oauth_profile.sh"
expect_helper_failure() {
  if bash "$HELPER" "$@" > "$TEST_ROOT/output" 2>&1; then printf 'FAIL test: helper-refusal\n'; exit 1; fi
}
expect_helper_failure login missions
CODEX_MISSION_HOME="$CODEX_INTERACTIVE_HOME/." expect_helper_failure prepare missions
reset_fixtures; rm -rf "$CODEX_MISSION_HOME"
bash "$HELPER" prepare missions
[[ $(cat "$CODEX_MISSION_HOME/config.toml") == 'cli_auth_credentials_store = "file"' ]]
mkdir "$TEST_ROOT/bin"
export PATH="$TEST_ROOT/bin:$PATH" TEST_ROOT
cat > "$TEST_ROOT/bin/codex" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ $* == '-c cli_auth_credentials_store="file" login' || $* == '-c cli_auth_credentials_store="file" login --device-auth' ]]
[[ $* != *--device-auth ]] || touch "$TEST_ROOT/device-login"
[[ -z ${OPENAI_API_KEY:-} && -z ${CODEX_API_KEY:-} ]]
cp "$TEST_ROOT/cloud/auth.json" "$CODEX_HOME/auth.json"
MOCK
cat > "$TEST_ROOT/bin/gcloud" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ $* == "secrets versions add fixture-secret --project fixture-project --data-file $TEST_ROOT/cloud/auth.json --quiet" ]]
touch "$TEST_ROOT/published"
MOCK
chmod +x "$TEST_ROOT/bin/"*
OPENAI_API_KEY=dummy CODEX_API_KEY=dummy bash "$HELPER" login missions
rm "$CODEX_MISSION_HOME/auth.json"
OPENAI_API_KEY=dummy CODEX_API_KEY=dummy bash "$HELPER" login missions --device-auth
[[ -f $TEST_ROOT/device-login ]]
# Mock codex copied cloud fixture: the checker must refuse publication.
expect_helper_failure publish-cloud fixture-project fixture-secret
[[ ! -e $TEST_ROOT/published ]]
fixture "$CODEX_MISSION_HOME" refresh-fixture-m
bash "$HELPER" publish-cloud fixture-project fixture-secret
[[ -f $TEST_ROOT/published ]]
# Explicit configuration preserves MCP/model sections while replacing global store.
reset_fixtures; rm -rf "$CODEX_MISSION_HOME"
cat > "$TEST_ROOT/audited.toml" <<'CONFIG'
cli_auth_credentials_store = "keyring"
model = "gpt-6.1-sol"
[mcp_servers.fixture]
command = "fixture-server"
CONFIG
bash "$HELPER" prepare missions --config "$TEST_ROOT/audited.toml"
[[ $(sed -n '1p' "$CODEX_MISSION_HOME/config.toml") == 'cli_auth_credentials_store = "file"' ]]
[[ $(sed -n '2p' "$CODEX_MISSION_HOME/config.toml") == 'model = "gpt-6.1-sol"' ]]
expect_helper_failure prepare missions --config "$TEST_ROOT/audited.toml"
rm "$CODEX_MISSION_HOME/config.toml"
ln -s "$TEST_ROOT/audited.toml" "$CODEX_MISSION_HOME/config.toml"
expect_helper_failure prepare missions
printf 'PASS test: attended-provisioning-and-publish-gate\n' 
