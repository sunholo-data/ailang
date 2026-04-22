#!/usr/bin/env bash
# index_ailang_syntax.sh — bootstrap (and reindex) the brain corpora that
# back micro-rag (μRAG). Always resolves the *active* prompt version — never
# pin to a specific version anywhere in this script or the resulting chunks.
#
# Namespaces populated:
#   ailang-syntax    — h2 chunks of the active prompt + LIMITATIONS.md
#   ailang-builtins  — one chunk per registered builtin (signature + module + desc)
#   ailang-examples  — one pointer per runnable example (header + first comment)
#
# Usage:
#   tools/index_ailang_syntax.sh             # idempotent additive run
#   tools/index_ailang_syntax.sh --reset     # drop namespaces and rebuild
#   AILANG_BIN=/custom/ailang tools/index_ailang_syntax.sh
#
# Exit codes:
#   0  success
#   1  active prompt could not be resolved (loud failure — no silent fallback)
#   2  ailang binary missing or wrong subcommand surface

set -euo pipefail

AILANG_BIN="${AILANG_BIN:-ailang}"
SCOPE="${SCOPE:-project}"

if ! command -v "$AILANG_BIN" >/dev/null 2>&1; then
  echo "ERROR: ailang binary not found at '$AILANG_BIN'" >&2
  exit 2
fi

# Resolve the active prompt version. No fallback — if the binary cannot
# tell us, the indexer must fail so callers know to run `ailang prompt --list`.
ACTIVE_VERSION="$("$AILANG_BIN" prompt --version-active 2>/dev/null || true)"
if [[ -z "$ACTIVE_VERSION" ]]; then
  echo "ERROR: could not resolve active prompt version." >&2
  echo "  Try: $AILANG_BIN prompt --list" >&2
  exit 1
fi

PROMPT_FILE="prompts/${ACTIVE_VERSION}.md"
if [[ ! -f "$PROMPT_FILE" ]]; then
  echo "ERROR: active prompt file '$PROMPT_FILE' not found." >&2
  exit 1
fi
LIMITATIONS_FILE="docs/LIMITATIONS.md"

RESET=0
for arg in "$@"; do
  case "$arg" in
    --reset) RESET=1 ;;
    --help|-h)
      sed -n '2,18p' "$0"
      exit 0
      ;;
  esac
done

echo "═══ μRAG corpus indexer ═══"
echo "  Active prompt:  $ACTIVE_VERSION"
echo "  Prompt file:    $PROMPT_FILE"
echo "  Limitations:    $LIMITATIONS_FILE"
echo "  Scope:          $SCOPE"
echo "  Reset mode:     $RESET"
echo

if [[ "$RESET" -eq 1 ]]; then
  echo "Dropping existing namespaces (--reset)…"
  "$AILANG_BIN" cache delete-namespace --namespace ailang-syntax   --scope "$SCOPE" --yes || true
  "$AILANG_BIN" cache delete-namespace --namespace ailang-builtins --scope "$SCOPE" --yes || true
  "$AILANG_BIN" cache delete-namespace --namespace ailang-examples --scope "$SCOPE" --yes || true
  echo
fi

# --- 1. Chunk the active prompt by H2 -------------------------------------
echo "[1/3] Indexing $PROMPT_FILE → ailang-syntax …"
prompt_count=0
chunk_h2() {
  local source_file="$1"
  local key_prefix="$2"
  awk -v src="$source_file" -v ver="$ACTIVE_VERSION" -v prefix="$key_prefix" '
    BEGIN { section=""; body=""; nchunks=0 }
    /^## / {
      if (section != "") {
        slug = section
        gsub(/[^a-zA-Z0-9]+/, "-", slug)
        slug = tolower(slug)
        sub(/^-+/, "", slug); sub(/-+$/, "", slug)
        printf "%s|%s|%s\n%s\n---END---\n", prefix "-" slug, section, ver, body
        nchunks++
      }
      section = $0
      sub(/^## /, "", section)
      body = $0 "\n"
      next
    }
    { body = body $0 "\n" }
    END {
      if (section != "") {
        slug = section
        gsub(/[^a-zA-Z0-9]+/, "-", slug)
        slug = tolower(slug)
        sub(/^-+/, "", slug); sub(/-+$/, "", slug)
        printf "%s|%s|%s\n%s\n---END---\n", prefix "-" slug, section, ver, body
        nchunks++
      }
    }
  ' "$source_file"
}

put_chunks() {
  local namespace="$1"
  local count_var="$2"
  local key="" header="" body=""
  while IFS= read -r line; do
    if [[ -z "$key" ]]; then
      key="$line"
      continue
    fi
    if [[ "$line" == "---END---" ]]; then
      # Parse key|header|version
      IFS='|' read -r ckey csection cver <<<"$key"
      # Skip empty bodies and trivially short ones (<200 bytes — no signal)
      if [[ ${#body} -ge 200 ]]; then
        # Tag content with metadata header for retrieval clarity.
        content=$(printf "[ns:%s] [version:%s] [section:%s]\n%s" "$namespace" "$cver" "$csection" "$body")
        # Note: key MUST come last — flag.Parse stops at first positional.
        "$AILANG_BIN" cache put \
          --content "$content" \
          --ns "$namespace" \
          --scope "$SCOPE" \
          --embed \
          "$ckey" >/dev/null 2>&1 || echo "    warn: failed to put $ckey" >&2
        eval "$count_var=\$((\$$count_var + 1))"
      fi
      key=""; header=""; body=""
      continue
    fi
    body+="$line"$'\n'
  done
}

# Process substitution `< <(...)` keeps the while loop in the parent shell
# so counter updates persist (a `|` pipe would fork a subshell and lose them).
put_chunks "ailang-syntax" prompt_count < <(chunk_h2 "$PROMPT_FILE" "syntax-${ACTIVE_VERSION}")
echo "    indexed $prompt_count prompt sections"

if [[ -f "$LIMITATIONS_FILE" ]]; then
  echo "    + $LIMITATIONS_FILE"
  put_chunks "ailang-syntax" prompt_count < <(chunk_h2 "$LIMITATIONS_FILE" "limitations")
  echo "    indexed $prompt_count syntax chunks total"
fi
echo

# --- 2. One chunk per builtin --------------------------------------------
echo "[2/3] Indexing builtins → ailang-builtins …"
builtins_json="$("$AILANG_BIN" builtins list --json 2>/dev/null)"
if [[ -z "$builtins_json" ]] || ! echo "$builtins_json" | python3 -c "import json,sys; json.load(sys.stdin)" >/dev/null 2>&1; then
  echo "ERROR: 'ailang builtins list --json' produced no/invalid output." >&2
  exit 2
fi

builtin_count=0
while IFS=$'\t' read -r bname bmodule bsignature bisp beffect bdesc; do
  [[ -z "$bname" ]] && continue
  effect_label="pure"
  [[ "$bisp" == "false" && -n "$beffect" ]] && effect_label="$beffect"
  body=$(printf "[ns:ailang-builtins] [version:%s] [module:%s] [effect:%s]\n%s\n%s" \
    "$ACTIVE_VERSION" "$bmodule" "$effect_label" "$bsignature" "$bdesc")
  key="builtin-$(echo "$bname" | tr -c 'A-Za-z0-9_' '_')"
  "$AILANG_BIN" cache put \
    --content "$body" \
    --ns ailang-builtins \
    --scope "$SCOPE" \
    --embed \
    "$key" >/dev/null 2>&1 || echo "    warn: failed to put $key" >&2
  builtin_count=$((builtin_count + 1))
done < <(echo "$builtins_json" | python3 -c '
import json, sys
d = json.load(sys.stdin)
for b in d["builtins"]:
    print("\t".join([
        b.get("name",""),
        b.get("module",""),
        b.get("signature",""),
        "true" if b.get("is_pure") else "false",
        b.get("effect",""),
        (b.get("description","") or "").replace("\n"," ").replace("\t"," "),
    ]))
')
echo "    indexed $builtin_count builtins"
echo

# --- 3. Runnable examples → pointers --------------------------------------
echo "[3/3] Indexing examples/runnable/*.ail → ailang-examples …"
example_count=0
if [[ -d examples/runnable ]]; then
  while IFS= read -r f; do
    head_block="$(head -20 "$f")"
    rel="${f#./}"
    body=$(printf "[ns:ailang-examples] [version:%s] [path:%s]\n%s" \
      "$ACTIVE_VERSION" "$rel" "$head_block")
    key="example-$(echo "${rel#examples/runnable/}" | tr -c 'A-Za-z0-9_' '_')"
    "$AILANG_BIN" cache put \
      --content "$body" \
      --ns ailang-examples \
      --scope "$SCOPE" \
      --embed \
      "$key" >/dev/null 2>&1 || echo "    warn: failed to put $key" >&2
    example_count=$((example_count + 1))
  done < <(find examples/runnable -maxdepth 2 -name '*.ail' -type f | sort)
fi
echo "    indexed $example_count examples"
echo

# --- Summary --------------------------------------------------------------
echo "═══ Done ═══"
echo "  Active version:    $ACTIVE_VERSION"
echo "  Syntax chunks:     $prompt_count"
echo "  Builtin chunks:    $builtin_count"
echo "  Example pointers:  $example_count"
echo
"$AILANG_BIN" cache stats 2>/dev/null | grep -E "ailang-(syntax|builtins|examples)" || true
