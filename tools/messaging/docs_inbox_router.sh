#!/usr/bin/env bash
# Poll doc-related traffic and route it to docs-mission.
#
# The heuristic is deliberately case-insensitive and matches documentation terms
# (docs/documentation, guide, example, website, published page, reference, API
# docs, tutorial, README, and related labels). It catches plausible requests,
# not only exact labels, and is not a precision classifier: it can miss unusual
# wording and can route a false positive. GitHub issue IDs are represented as
# github:sunholo-data/ailang#N because gh issues are not message-store records.
#
# Every cloud selector is attached to the individual ailang invocation. Calls
# are bounded and a failed/invalid read is fatal; an empty valid JSON array is
# different from an unavailable store. State is configurable with
# DOCS_ROUTER_STATE_DIR (default: ~/.ailang/docs-inbox-router).

set -u

CALL_TIMEOUT_SECONDS=${DOCS_ROUTER_TIMEOUT_SECONDS:-20}
AILANG_CMD=${AILANG_CMD:-ailang}
GH_CMD=${GH_CMD:-gh}
STATE_DIR=${DOCS_ROUTER_STATE_DIR:-${HOME}/.ailang/docs-inbox-router}
LEDGER_FILE=${STATE_DIR}/forwarded.keys
WATERMARK_FILE=${STATE_DIR}/github.watermark

die() {
  printf 'docs-inbox-router: %s\n' "$*" >&2
  exit 1
}

bounded_exec() {
  if command -v timeout >/dev/null 2>&1; then
    timeout "$CALL_TIMEOUT_SECONDS" "$@"
  elif command -v gtimeout >/dev/null 2>&1; then
    gtimeout "$CALL_TIMEOUT_SECONDS" "$@"
  else
    "$@" &
    child_pid=$!
    (
      sleep "$CALL_TIMEOUT_SECONDS"
      kill -TERM "$child_pid" 2>/dev/null
      sleep 1
      kill -KILL "$child_pid" 2>/dev/null
    ) &
    watchdog_pid=$!
    wait "$child_pid"
    child_status=$?
    kill "$watchdog_pid" 2>/dev/null
    wait "$watchdog_pid" 2>/dev/null
    return "$child_status"
  fi
}

run_ailang_json() {
  # $1 is a diagnostic label; remaining arguments are the ailang arguments.
  label=$1
  shift
  output_file=$(mktemp "${TMPDIR:-/tmp}/docs-router.XXXXXX") || die "cannot create temporary output for $label"
  if ! AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
    bounded_exec "$AILANG_CMD" "$@" >"$output_file" 2>"${output_file}.err"; then
    error_text=$(tr '\n' ' ' <"${output_file}.err")
    rm -f "$output_file" "${output_file}.err"
    die "$label failed or timed out (canonical message store): ${error_text:-no diagnostic}"
  fi
  rm -f "${output_file}.err"
  if ! jq -e 'type == "array"' "$output_file" >/dev/null 2>&1; then
    rm -f "$output_file"
    die "$label returned invalid JSON (canonical message store)"
  fi
  JSON_FILE=$output_file
}

run_gh_json() {
  output_file=$(mktemp "${TMPDIR:-/tmp}/docs-router-gh.XXXXXX") || die "cannot create temporary output for GitHub"
  if ! bounded_exec "$GH_CMD" issue list --repo sunholo-data/ailang --state all --limit 1000 \
    --json number,title,body,labels,createdAt >"$output_file" 2>"${output_file}.err"; then
    error_text=$(tr '\n' ' ' <"${output_file}.err")
    rm -f "$output_file" "${output_file}.err"
    die "GitHub issue list failed or timed out: ${error_text:-no diagnostic}"
  fi
  rm -f "${output_file}.err"
  if ! jq -e 'type == "array" and all(.[]; (.number|type == "number") and (.createdAt|type == "string"))' \
    "$output_file" >/dev/null 2>&1; then
    rm -f "$output_file"
    die "GitHub issue list returned invalid JSON shape"
  fi
  JSON_FILE=$output_file
}

is_doc_related() {
  # stdin is one JSON object; labels are included in the searchable text.
  jq -r '[(.title // ""), (.body // ""), ((.labels // [])[]?.name // "")] | join(" ")' \
    | grep -Eiq '(^|[^[:alnum:]])(doc|docs|documentation|guide|guides|example|examples|website|web[ -]?site|published[ -]?page|reference|api[ -]?docs|tutorial|readme)([^[:alnum:]]|$)'
}

key_seen() {
  [ -f "$LEDGER_FILE" ] && grep -Fqx -- "$1" "$LEDGER_FILE"
}

record_key() {
  key=$1
  tmp_file=$(mktemp "${LEDGER_FILE}.XXXXXX") || die "cannot create ledger temporary"
  if [ -f "$LEDGER_FILE" ]; then
    cp "$LEDGER_FILE" "$tmp_file" || { rm -f "$tmp_file"; die "cannot copy ledger"; }
  fi
  printf '%s\n' "$key" >>"$tmp_file" || { rm -f "$tmp_file"; die "cannot write ledger"; }
  mv "$tmp_file" "$LEDGER_FILE" || { rm -f "$tmp_file"; die "cannot atomically update ledger"; }
}

forward_item() {
  source_key=$1
  message_id=$2
  reason=$3
  key_seen "$source_key" && return 1
  if ! AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
    bounded_exec "$AILANG_CMD" messages forward --to docs-mission --reason "$reason" "$message_id" \
    >/dev/null 2>"${TMPDIR:-/tmp}/docs-router-forward.err"; then
    error_text=$(tr '\n' ' ' <"${TMPDIR:-/tmp}/docs-router-forward.err")
    rm -f "${TMPDIR:-/tmp}/docs-router-forward.err"
    die "forward failed for ${source_key}: ${error_text:-no diagnostic}"
  fi
  rm -f "${TMPDIR:-/tmp}/docs-router-forward.err"
  record_key "$source_key"
  return 0
}

poll_messages() {
  inbox=$1
  run_ailang_json "message list ${inbox}" messages list --inbox "$inbox" --unread --json
  json_file=$JSON_FILE
  count=$(jq 'length' "$json_file")
  checked=$((checked + count))
  while IFS=$'\t' read -r message_id title body; do
    [ -n "$message_id" ] || die "message list ${inbox} contained an item without an ID"
    item=$(jq -cn --arg id "$message_id" --arg title "$title" --arg body "$body" '{id:$id,title:$title,body:$body}')
    if printf '%s\n' "$item" | is_doc_related; then
      if forward_item "message:${inbox}|${message_id}" "$message_id" "doc-related traffic from ${inbox}"; then
        forwarded=$((forwarded + 1))
      fi
    fi
  done <<EOF
$(jq -r '.[] | [(.id // .message_id // .messageId // ""), (.title // ""), (.body // .content // "")] | @tsv' "$json_file")
EOF
  rm -f "$json_file"
}

poll_github() {
  run_gh_json
  json_file=$JSON_FILE
  watermark=1970-01-01T00:00:00Z
  [ -f "$WATERMARK_FILE" ] && watermark=$(sed -n '1p' "$WATERMARK_FILE")
  checked=$((checked + $(jq --arg since "$watermark" '[.[] | select(.createdAt > $since)] | length' "$json_file")))
  new_watermark=$(jq -r --arg since "$watermark" '[.[] | select(.createdAt > $since) | .createdAt] | max // $since' "$json_file")
  while IFS=$'\t' read -r number title body; do
    issue=$(jq -cn --arg title "$title" --arg body "$body" '{title:$title,body:$body}')
    if printf '%s\n' "$issue" | is_doc_related; then
      if forward_item "github:sunholo-data/ailang|${number}" \
        "github:sunholo-data/ailang#${number}" "doc-related GitHub issue #${number}"; then
        forwarded=$((forwarded + 1))
      fi
    fi
  done <<EOF
$(jq -r --arg since "$watermark" '.[] | select(.createdAt > $since) | [(.number|tostring), (.title // ""), (.body // "")] | @tsv' "$json_file")
EOF
  tmp_file=$(mktemp "${WATERMARK_FILE}.XXXXXX") || die "cannot create watermark temporary"
  if ! printf '%s\n' "$new_watermark" >"$tmp_file"; then
    rm -f "$tmp_file"
    die "cannot write GitHub watermark"
  fi
  if ! mv "$tmp_file" "$WATERMARK_FILE"; then
    rm -f "$tmp_file"
    die "cannot atomically update GitHub watermark"
  fi
  rm -f "$json_file"
}

run_router() {
  mkdir -p "$STATE_DIR" || die "cannot create state directory: $STATE_DIR"
  touch "$LEDGER_FILE" || die "cannot write state ledger: $LEDGER_FILE"
  checked=0
  forwarded=0
  poll_messages public-feedback
  run_ailang_json "message inbox discovery" messages list --unread --json
  discovery_file=$JSON_FILE
  inboxes=$(jq -r '.[] | (.inbox // .to // "") | select(test("^pkg:[^/]+/[^/]+$"))' "$discovery_file" | sort -u)
  rm -f "$discovery_file"
  while IFS= read -r inbox; do
    [ -n "$inbox" ] && poll_messages "$inbox"
  done <<EOF
$inboxes
EOF
  poll_github
  printf 'checked=%s forwarded=%s\n' "$checked" "$forwarded"
}

selftest() {
  test_dir=$(mktemp -d "${TMPDIR:-/tmp}/docs-router-selftest.XXXXXX") || die "cannot create self-test directory"
  fake_ailang="$test_dir/ailang"
  fake_gh="$test_dir/gh"
  log_file="$test_dir/forward.log"
  cat >"$fake_ailang" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = messages ] && [ "$2" = list ]; then
  case "$*" in
    *"--inbox public-feedback"*) printf '%s\n' '[{"id":"msg-positive","title":"Update the docs guide","body":"Please document this example"},{"id":"msg-negative","title":"Compiler crash","body":"Runtime failure"}]' ;;
    *"--inbox pkg:sunholo/example"*) printf '%s\n' '[{"id":"pkg-positive","title":"Published page typo","body":"Fix the reference page"}]' ;;
    *) printf '%s\n' '[{"id":"discovery","inbox":"pkg:sunholo/example"}]' ;;
  esac
elif [ "$1" = messages ] && [ "$2" = forward ]; then
  printf '%s\n' "$*" >>"$DOCS_ROUTER_TEST_LOG"
else
  exit 2
fi
EOF
  cat >"$fake_gh" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' '[{"number":101,"title":"Improve examples","body":"Add a tutorial","labels":[],"createdAt":"2026-01-02T00:00:00Z"},{"number":102,"title":"Bug","body":"Crash on input","labels":[],"createdAt":"2026-01-03T00:00:00Z"}]'
EOF
  chmod +x "$fake_ailang" "$fake_gh"
  first=$(AILANG_CMD="$fake_ailang" GH_CMD="$fake_gh" DOCS_ROUTER_TEST_LOG="$log_file" \
    DOCS_ROUTER_STATE_DIR="$test_dir/state" "$0") || { rm -rf "$test_dir"; die "self-test first pass failed"; }
  second=$(AILANG_CMD="$fake_ailang" GH_CMD="$fake_gh" DOCS_ROUTER_TEST_LOG="$log_file" \
    DOCS_ROUTER_STATE_DIR="$test_dir/state" "$0") || { rm -rf "$test_dir"; die "self-test second pass failed"; }
  [ "$first" = checked=5\ forwarded=3 ] || { rm -rf "$test_dir"; die "self-test expected first pass checked=5 forwarded=3, got: $first"; }
  [ "$second" = checked=3\ forwarded=0 ] || { rm -rf "$test_dir"; die "self-test expected second pass checked=3 forwarded=0, got: $second"; }
  [ "$(wc -l <"$log_file" | tr -d ' ')" = 3 ] || { rm -rf "$test_dir"; die "self-test duplicate suppression failed"; }
  grep -Fq 'messages forward --to docs-mission --reason doc-related traffic from public-feedback msg-positive' "$log_file" || { rm -rf "$test_dir"; die "self-test forward ordering failed"; }
  printf 'selftest: known-positive matched; known-negative suppressed\n'
  printf 'selftest: first pass %s\n' "$first"
  printf 'selftest: second pass %s (duplicate suppression)\n' "$second"
  printf 'selftest: forward argument ordering and persisted ledger verified\n'
  rm -rf "$test_dir"
}

case "${1:-}" in
  --selftest) selftest ;;
  "") run_router ;;
  *) die "usage: $0 [--selftest]" ;;
esac
