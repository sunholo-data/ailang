#!/usr/bin/env bash
set -euo pipefail

archive=${1:?usage: verify_release_archive.sh ARCHIVE BINARY [--smoke]}
binary=${2:?usage: verify_release_archive.sh ARCHIVE BINARY [--smoke]}
smoke=${3:-}
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
extract_dir=$(mktemp -d)
trap 'rm -rf "$extract_dir"' EXIT

case "$archive" in
  *.tar.gz) tar -xzf "$archive" -C "$extract_dir" ;;
  *.zip) unzip -q "$archive" -d "$extract_dir" ;;
  *) echo "unsupported archive: $archive" >&2; exit 2 ;;
esac

test -f "$extract_dir/bin/$binary"
test -f "$extract_dir/std/VERSION"
expected=$(find "$repo_root/std" -maxdepth 1 -type f -name '*.ail' | wc -l | tr -d ' ')
actual=$(find "$extract_dir/std" -maxdepth 1 -type f -name '*.ail' | wc -l | tr -d ' ')
test "$actual" = "$expected"

if [[ "$smoke" == "--smoke" ]]; then
  (
    cd "$extract_dir"
    env -u AILANG_STDLIB_PATH "$extract_dir/bin/$binary" docs --list >/dev/null
    env -u AILANG_STDLIB_PATH AILANG_RELAX_MODULES=1 "$extract_dir/bin/$binary" check "$repo_root/examples/runnable/array_basic.ail" >/dev/null
  )
fi
