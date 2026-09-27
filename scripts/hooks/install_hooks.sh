#!/bin/bash
# install_hooks.sh — link versioned git hooks into this clone's hooks dir.
# Idempotent. Never overwrites an unmanaged hook: says so and exits 1 instead.
# Quiet when already installed (safe to call from SessionStart).
set -u
root=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "install_hooks: not in a git repo" >&2; exit 1; }
hooks_dir=$(git config --get core.hooksPath 2>/dev/null)
[ -n "$hooks_dir" ] || hooks_dir="$(git rev-parse --git-common-dir)/hooks"
case "$hooks_dir" in /*) ;; *) hooks_dir="$root/$hooks_dir" ;; esac
mkdir -p "$hooks_dir" || exit 1
src="$root/scripts/hooks/pre-push"
dst="$hooks_dir/pre-push"
# Link to the MAIN checkout's copy when run from a worktree, so the hook survives
# the worktree being removed.
common=$(cd "$(git rev-parse --git-common-dir)" && pwd)
main_root=$(dirname "$common")
[ -f "$main_root/scripts/hooks/pre-push" ] && src="$main_root/scripts/hooks/pre-push"
if [ -L "$dst" ] && [ "$(readlink "$dst")" = "$src" ]; then
    exit 0
fi
if [ -e "$dst" ] || [ -L "$dst" ]; then
    echo "install_hooks: $dst exists and is not ours — leaving it; merge by hand." >&2
    exit 1
fi
ln -s "$src" "$dst" && echo "install_hooks: pre-push -> $src"
