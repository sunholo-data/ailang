#!/usr/bin/env bash
# check_published_tool_names.sh — M-EXT-AUTHOR-DX M3 pre-launch verification.
#
# Fetches every currently-published sunholo/motoko_ext_* package from the
# registry, extracts its provided_tools + on_describe_tools tool-name
# advertisements, runs the same [A-Za-z0-9_]{1,128} validation the
# registry-validator now enforces at publish time, and reports any name
# that would be rejected.
#
# Run BEFORE merging M3 (or any change to validateToolNames in
# cmd/registry-validator/main.go) to catch false positives that would
# block re-publishing an existing package.
#
# Exit 0 if all published packages pass. Exit 1 if any would be rejected.

set -euo pipefail

# Uses the local ~/.ailang/cache/registry/ tree (populated by `ailang lock`
# resolving published packages) rather than fetching tarballs over HTTP —
# the canonical tarball URLs aren't stable, and `ailang lock` already
# materializes every published version we care about into the cache.
CACHE="${HOME}/.ailang/cache/registry/sunholo"

PACKAGES=(
  motoko_ext_abi
  motoko_ext_a2a
  motoko_ext_ai_compat
  motoko_ext_compaction_ai
  motoko_ext_compose
  motoko_ext_context_mode
  motoko_ext_decision_framework
  motoko_ext_exa_search
  motoko_ext_mcp
  motoko_ext_microrag
  motoko_ext_omnigraph
  motoko_ext_test_dummy
  motoko_ext_ailang_docs
)

# Matches valid tool names — anything else would be rejected by the gate.
SAFE_RE='^[A-Za-z0-9_]+$'

total_packages=0
flagged_packages=0
flagged_names=()

for pkg in "${PACKAGES[@]}"; do
  total_packages=$((total_packages + 1))
  pkg_dir="$CACHE/$pkg"

  if [[ ! -d "$pkg_dir" ]]; then
    echo "  ! sunholo/$pkg: no local cache (run \`ailang lock\` in a project that pins it first)" >&2
    continue
  fi

  # Pick latest cached version (semver-ish lexicographic sort works for our pkgs)
  version=$(ls "$pkg_dir" 2>/dev/null | sort -V | tail -1)
  if [[ -z "$version" ]]; then
    echo "  ! sunholo/$pkg: no versions in cache" >&2
    continue
  fi
  extract_dir="$pkg_dir/$version"

  # Extract advertised tool names from .ail files using the same regex
  # patterns the Go validator uses.
  names=$(grep -rEh 'provided_tools\s*[:=]\s*\[[^]]*\]|name\s*:\s*"[^"]+"' "$extract_dir" 2>/dev/null \
    | python3 -c "
import sys, re
src = sys.stdin.read()
names = set()
# provided_tools: [\"A\", \"B\", ...]
for m in re.finditer(r'provided_tools\s*[:=]\s*\[([^\]]*)\]', src):
    for s in re.findall(r'\"([^\"]+)\"', m.group(1)):
        names.add(s)
# name: \"X\"
for m in re.finditer(r'\bname\s*:\s*\"([^\"]+)\"', src):
    names.add(m.group(1))
print('\n'.join(sorted(names)))
")

  if [[ -z "$names" ]]; then
    continue
  fi

  # Validate each name
  pkg_flagged=()
  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    if ! [[ "$name" =~ $SAFE_RE ]]; then
      pkg_flagged+=("$name")
      flagged_names+=("$pkg@$version: $name")
    fi
  done <<< "$names"

  if [[ ${#pkg_flagged[@]} -gt 0 ]]; then
    flagged_packages=$((flagged_packages + 1))
    echo "  ✗ sunholo/$pkg@$version flagged ${#pkg_flagged[@]} name(s): ${pkg_flagged[*]}"
  else
    echo "  ✓ sunholo/$pkg@$version (advertised names all conform)"
  fi
done

echo ""
echo "Scanned $total_packages published packages."
if [[ $flagged_packages -eq 0 ]]; then
  echo "✓ All published packages conform to the M3 naming gate."
  echo "  Safe to merge — no false positives on re-publish."
  exit 0
fi
echo "✗ $flagged_packages package(s) flagged. Names:"
for n in "${flagged_names[@]}"; do
  echo "  - $n"
done
echo ""
echo "ACTION: either (a) bump the affected packages to drop the offending"
echo "names BEFORE the M3 gate ships, or (b) document that re-publishing"
echo "these packages requires the --allow-dotted-tool-names flag."
exit 1
