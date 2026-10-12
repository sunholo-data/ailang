#!/usr/bin/env bash
# Read-only/offline. JWT claims are inspected for consistency, not verified online.
# Deliberately emit only fixed labels: never auth values, IDs, hashes or paths.
set -euo pipefail
set +x
fail() { printf 'FAIL %s\n' "$1" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || fail dependency-jq
homes=("${CODEX_INTERACTIVE_HOME:-$HOME/.codex}" "${CODEX_MISSION_HOME:-$HOME/.codex-missions}" "${CODEX_CLOUD_HOME:-$HOME/.codex-cloud-staging}")
labels=(interactive missions cloud)
canonical=()
file_mode() {
  if [[ $(uname -s) == Darwin ]]; then stat -f '%Lp' "$1" 2>/dev/null; else stat -c '%a' "$1" 2>/dev/null; fi
}
for index in 0 1 2; do
  home=${homes[$index]}; label=${labels[$index]}
  [[ -d $home && ! -L $home ]] || fail "$label-home"
  canonical[index]=$(cd "$home" && pwd -P) || fail "$label-home"
  [[ $(file_mode "$home") == 700 ]] || fail "$label-home-permissions"
  [[ -f $home/auth.json && ! -L $home/auth.json ]] || fail "$label-auth-file"
  [[ $(file_mode "$home/auth.json") == 600 ]] || fail "$label-auth-permissions"
done
[[ ${canonical[0]} != "${canonical[1]}" && ${canonical[0]} != "${canonical[2]}" && ${canonical[1]} != "${canonical[2]}" ]] || fail distinct-homes
# jq owns all sensitive comparisons; nothing sensitive goes into command arguments.
# sub is the OAuth user identity; account_id is the selected ChatGPT workspace.
# Require the ID-token workspace claim to agree with Codex's header identity.
if ! jq -en --slurpfile interactive "${homes[0]}/auth.json" --slurpfile missions "${homes[1]}/auth.json" --slurpfile cloud "${homes[2]}/auth.json" '
  def present: type == "string" and length > 0;
  def identity:
    . as $a |
    ($a.tokens.id_token | split(".")[1] | gsub("-";"+") | gsub("_";"/") | @base64d | fromjson) as $claims |
    {user:$claims.sub, workspace:$a.tokens.account_id, claimed_workspace:$claims["https://api.openai.com/auth"].chatgpt_account_id};
  [$interactive, $missions, $cloud] |
  if any(.[]; length != 1) then false else
    map(.[0]) as $profiles |
    ($profiles | map(identity)) as $identities |
    all($profiles[]; .auth_mode == "chatgpt" and (.OPENAI_API_KEY == null or .OPENAI_API_KEY == "") and (.tokens.access_token | present) and (.tokens.id_token | present) and (.tokens.refresh_token | present)) and
    all($identities[]; (.user | present) and (.workspace | present) and .workspace == .claimed_workspace) and
    ([$identities[].user] | unique | length == 1) and
    ([$identities[].workspace] | unique | length == 1) and
    ([$profiles[].tokens.refresh_token] | unique | length == 3)
  end
' >/dev/null 2>&1; then fail account-workspace-subscription-isolation; fi
printf 'PASS interactive\nPASS missions\nPASS cloud\nPASS account-workspace-subscription-isolation\n'
