#!/usr/bin/env bash
# verify_setup.sh — Verify the local Ollama eval rig is ready.
#
# Usage:
#   .claude/skills/local-ollama-eval/scripts/verify_setup.sh
#
# Returns 0 if everything is green, non-zero on the first missing piece.
# Designed to be run before a fresh eval session, especially after reboot.

set -u

GREEN="\033[0;32m"; RED="\033[0;31m"; YELLOW="\033[1;33m"; CYAN="\033[0;36m"; RESET="\033[0m"
ok()   { printf "${GREEN}✓${RESET} %s\n" "$1"; }
fail() { printf "${RED}✗${RESET} %s\n" "$1"; }
warn() { printf "${YELLOW}⚠${RESET} %s\n" "$1"; }
info() { printf "${CYAN}ℹ${RESET} %s\n" "$1"; }

ERRORS=0
fail_and_count() { fail "$1"; ERRORS=$((ERRORS + 1)); }

printf "${CYAN}=== Local Ollama Eval Rig — Setup Check ===${RESET}\n\n"

# 1. Hardware sanity
if [[ "$(uname -s)" != "Darwin" ]]; then
  warn "This skill is tuned for macOS. Other OSes will need manual launchd-equivalent setup."
fi
TOTAL_MEM_GB=$(sysctl -n hw.memsize 2>/dev/null | awk '{printf "%.0f", $1/1024/1024/1024}')
if [[ -z "$TOTAL_MEM_GB" || "$TOTAL_MEM_GB" -lt 32 ]]; then
  warn "Only ${TOTAL_MEM_GB:-unknown} GB total memory. gemma4:26b needs ~26 GB resident. Smaller models recommended."
else
  ok "Memory: ${TOTAL_MEM_GB} GB total (gemma4:26b needs ~26 GB)"
fi

# 2. Ollama
if command -v ollama >/dev/null 2>&1; then
  ok "ollama CLI installed: $(ollama --version 2>&1 | head -1)"
else
  fail_and_count "ollama not installed. Install: brew install ollama"
fi
if curl -s -o /dev/null -w "%{http_code}" http://localhost:11434/api/tags 2>/dev/null | grep -q 200; then
  ok "ollama serve responding on localhost:11434"
else
  fail_and_count "ollama serve NOT responding. Start: ollama serve  (or launch the app)"
fi

# 3. Required model present
if curl -s http://localhost:11434/api/tags 2>/dev/null | grep -q '"name":"gemma4:26b"'; then
  ok "gemma4:26b model available locally"
else
  warn "gemma4:26b not pulled. Pull: ollama pull gemma4:26b  (~17 GB)"
fi

# 4. opencode
if command -v opencode >/dev/null 2>&1; then
  ok "opencode CLI installed: $(opencode --version 2>&1 | head -1)"
else
  fail_and_count "opencode not installed. Install: npm install -g opencode-ai"
fi

# 5. opencode Ollama provider config
CFG="$HOME/.config/opencode/opencode.jsonc"
if [[ -f "$CFG" ]] && grep -q '"ollama"' "$CFG" 2>/dev/null; then
  ok "opencode Ollama provider configured: $CFG"
else
  fail_and_count "opencode Ollama provider not configured. See resources/opencode_jsonc_example.txt"
fi
if command -v opencode >/dev/null 2>&1 && opencode models 2>/dev/null | grep -q 'ollama/gemma4:26b'; then
  ok "opencode sees ollama/gemma4:26b"
else
  warn "opencode does not see ollama/gemma4:26b — check config or run 'opencode models'"
fi

# 6. AILANG binary
if command -v ailang >/dev/null 2>&1; then
  ok "ailang installed: $(ailang --version 2>&1 | head -1)"
else
  fail_and_count "ailang not installed. Build: cd <repo> && make install"
fi

# 7. AILANG server (OTLP receiver)
if curl -s http://localhost:1957/health 2>/dev/null | grep -q healthy; then
  ok "ailang server responding on localhost:1957 (OTLP receiver alive)"
else
  warn "ailang server NOT running. Start: make services-start  (or install launchd plist for 24/7)"
fi

# 8. Observability env var
if [[ -n "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" ]]; then
  ok "OTEL_EXPORTER_OTLP_ENDPOINT=${OTEL_EXPORTER_OTLP_ENDPOINT}"
else
  warn "OTEL_EXPORTER_OTLP_ENDPOINT not set — live monitoring (ailang chains live) will show '(no spans yet)'"
  info "  Fix: export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957"
fi

# 9. Ollama parallelism env vars (best-effort check via launchctl)
PARALLEL=$(launchctl getenv OLLAMA_NUM_PARALLEL 2>/dev/null)
if [[ -n "$PARALLEL" ]]; then
  ok "OLLAMA_NUM_PARALLEL=$PARALLEL (launchctl)"
else
  warn "OLLAMA_NUM_PARALLEL not set via launchctl. Recommended: launchctl setenv OLLAMA_NUM_PARALLEL 4"
fi

# 10. Disk space for output
FREE_GB=$(df -g . 2>/dev/null | tail -1 | awk '{print $4}')
if [[ -n "$FREE_GB" && "$FREE_GB" -gt 10 ]]; then
  ok "Disk free: ${FREE_GB} GB"
else
  warn "Disk free: ${FREE_GB:-unknown} GB — eval_results can grow large under continuous rotation"
fi

printf "\n${CYAN}=== Summary ===${RESET}\n"
if [[ "$ERRORS" -eq 0 ]]; then
  printf "${GREEN}✅ Rig is ready for eval. Recommended next:${RESET}\n"
  printf "   .claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-gemma4-26b fizzbuzz\n"
  exit 0
else
  printf "${RED}❌ ${ERRORS} required components missing. Fix the items marked ✗ above.${RESET}\n"
  exit 1
fi
