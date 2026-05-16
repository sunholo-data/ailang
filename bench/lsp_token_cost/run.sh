#!/usr/bin/env bash
# bench/lsp_token_cost/run.sh
#
# Single-trial token-cost eval for the M-AILANG-LSP-FOR-AI milestone:
# runs the same fixed task once with the ailang-lsp plugin enabled and
# once without, and reports the input+output token delta.
#
# This is a SINGLE-TRIAL probe — N=1, single task, single model. The
# point is to publish a real number with caveats, not to be a
# statistically robust eval (that's the M-LSP-EVAL-FOLLOWUP follow-up).
#
# Usage: bash bench/lsp_token_cost/run.sh
#
# Requires:
#   - claude (Claude Code CLI) on PATH
#   - jq on PATH
#   - ailang installed (make install) and ailang-lsp plugin installed
#     (/plugin install ailang-lsp@ailang-tools)

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
TASK_FILE="$REPO_ROOT/bench/lsp_token_cost/task.md"
RESULTS="$REPO_ROOT/bench/lsp_token_cost/results.json"
WORKDIR="$(mktemp -d -t lsp_eval_XXXXXX)"
TASK_PROMPT="$(cat "$TASK_FILE")"

trap 'rm -rf "$WORKDIR"' EXIT

if ! command -v claude >/dev/null; then
  echo "ERROR: 'claude' (Claude Code CLI) not on PATH"
  exit 2
fi
if ! command -v jq >/dev/null; then
  echo "ERROR: 'jq' not on PATH"
  exit 2
fi

# Snapshot the fixture so each trial starts from the same state.
seed_workspace() {
  local dest="$1"
  rm -rf "$dest"
  mkdir -p "$dest/examples/lsp_xref_fixture"
  cp "$REPO_ROOT/examples/lsp_xref_fixture/a.ail" "$dest/examples/lsp_xref_fixture/a.ail"
  cp "$REPO_ROOT/examples/lsp_xref_fixture/b.ail" "$dest/examples/lsp_xref_fixture/b.ail"
}

run_trial() {
  local label="$1"     # "lsp_on" or "lsp_off"
  local enable_lsp="$2" # "true" or "false"
  local trial_dir="$WORKDIR/$label"
  seed_workspace "$trial_dir"

  # Headless Claude Code run. --output-format json gives us token usage
  # in the final result. Direct stdin pipe is the prompt.
  local plugin_flag=""
  if [[ "$enable_lsp" == "true" ]]; then
    plugin_flag="--plugin ailang-lsp@ailang-tools"
  fi

  cd "$trial_dir"
  echo "→ Running trial: $label (lsp=$enable_lsp)" >&2
  printf '%s' "$TASK_PROMPT" | claude --print --output-format json $plugin_flag 2>"$trial_dir/stderr.log" \
    > "$trial_dir/result.json"
  cd "$REPO_ROOT"

  # Extract metrics from the JSON result.
  jq --arg label "$label" '{
    label: $label,
    input_tokens: (.usage.input_tokens // 0),
    output_tokens: (.usage.output_tokens // 0),
    total_tokens: ((.usage.input_tokens // 0) + (.usage.output_tokens // 0)),
    duration_ms: (.duration_ms // 0),
    num_turns: (.num_turns // 0)
  }' "$trial_dir/result.json"
}

LSP_ON=$(run_trial "lsp_on" "true" 2>/dev/null || echo '{"label":"lsp_on","error":"trial failed"}')
LSP_OFF=$(run_trial "lsp_off" "false" 2>/dev/null || echo '{"label":"lsp_off","error":"trial failed"}')

# Compute deltas + write results.json.
jq -n --argjson on "$LSP_ON" --argjson off "$LSP_OFF" '{
  meta: {
    timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    task: "examples/lsp_xref_fixture: add subtract+use_subtract",
    note: "N=1, single task. Indicative not statistically robust. See README.md."
  },
  trials: { lsp_on: $on, lsp_off: $off },
  delta: {
    input_tokens: (($off.input_tokens // 0) - ($on.input_tokens // 0)),
    output_tokens: (($off.output_tokens // 0) - ($on.output_tokens // 0)),
    total_tokens_saved_by_lsp: (($off.total_tokens // 0) - ($on.total_tokens // 0)),
    pct_reduction: (
      if (($off.total_tokens // 0) > 0) then
        ((($off.total_tokens - ($on.total_tokens // 0)) * 100) / $off.total_tokens)
      else null end
    )
  }
}' > "$RESULTS"

echo
echo "=== Results written to $RESULTS ==="
jq . "$RESULTS"
