#!/usr/bin/env bash
# Deterministically estimate Task ELO and Model Band for a sprint slice
# based on code location and objective structural complexity signals.
#
# Usage:
#   estimate_task_elo.sh [--json] <path1> [path2 ...]

set -euo pipefail

if ! command -v python3 >/dev/null 2>&1; then
    echo "error: python3 is required to run estimate_task_elo.sh" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || pwd)"

exec python3 - "$REPO_ROOT" "$@" <<'PYEOF'
import json
import os
import re
import subprocess
import sys

FRONTIER_PREFIXES = (
    "internal/types",
    "internal/effects",
    "internal/eval",
    "internal/bytecode",
    "internal/smt",
    "internal/parser",
    "internal/elaborate",
)

CORE_PREFIXES = (
    "internal/pipeline",
    "internal/loader",
    "internal/lsp",
    "internal/quickfix",
    "internal/coordinator",
    "cmd/ailang",
)

ROUTINE_PREFIXES = (
    "std",
    "examples",
    "docs",
    "prompts",
    "benchmarks",
    ".agents",
    ".claude",
)

BANDS = (
    (2800, "Deep Frontier", ["gemini-4-carbon-high", "claude-opus-5-5-high"]),
    (2650, "Frontier SWE", ["gemini-4-neon-high", "gemini-4-argon-high"]),
    (2400, "Core SWE", ["gemini-4-neon-medium", "claude-sonnet-5-5-medium"]),
    (0, "Routine", ["gemini-4-hydrogen-medium", "gemini-4-hydrogen-high"]),
)

SKIP_DIRS = {
    ".git",
    "vendor",
    "node_modules",
    "ui",
    "docs",
    "dist",
    "bin",
    ".ailang",
}


def usage() -> None:
    print(
        "Usage: estimate_task_elo.sh [--json] <path1> [path2 ...]\n"
        "\n"
        "Deterministically compute Task ELO and recommended Model Band from\n"
        "code-location risk and structural complexity signals (no network).\n"
        "\n"
        "Options:\n"
        "  --json    Emit machine-readable JSON output\n"
        "  -h, --help  Show this help message",
        file=sys.stderr,
    )


def normalize_path(raw: str, repo_root: str) -> str:
    p = raw.strip()
    if os.path.isabs(p):
        try:
            rel = os.path.relpath(p, repo_root)
            if not rel.startswith(".."):
                p = rel
        except ValueError:
            pass
    p = p.replace("\\", "/")
    while p.startswith("./"):
        p = p[2:]
    p = p.rstrip("/")
    return p


def matches_prefix(path: str, prefix: str) -> bool:
    return path == prefix or path.startswith(prefix + "/")


def classify_base_elo(path: str) -> tuple[int, str, str]:
    for pref in FRONTIER_PREFIXES:
        if matches_prefix(path, pref):
            return 2650, "Frontier", f"{pref}/"
    for pref in CORE_PREFIXES:
        if matches_prefix(path, pref):
            return 2400, "Core SWE", f"{pref}/"
    for pref in ROUTINE_PREFIXES:
        if matches_prefix(path, pref):
            return 2100, "Routine", f"{pref}/"
    if matches_prefix(path, "internal"):
        return 2350, "Core SWE", "internal/* (default)"
    return 2100, "Routine", "other (default)"


def top_level_package(path: str) -> str:
    parts = [part for part in path.split("/") if part]
    if not parts:
        return "."
    if parts[0] in ("internal", "cmd") and len(parts) >= 2:
        return f"{parts[0]}/{parts[1]}"
    return parts[0]


def read_go_module(repo_root: str) -> str:
    gomod = os.path.join(repo_root, "go.mod")
    if not os.path.isfile(gomod):
        return ""
    with open(gomod, "r", encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if line.startswith("module "):
                return line.split()[1].strip()
    return ""


def target_go_pkg_dir(path: str, repo_root: str) -> str:
    abs_path = os.path.join(repo_root, path)
    if path.endswith(".go"):
        return os.path.dirname(path)
    if os.path.isdir(abs_path):
        try:
            if any(name.endswith(".go") for name in os.listdir(abs_path)):
                return path
        except OSError:
            pass
    if matches_prefix(path, "internal") or matches_prefix(path, "cmd"):
        if "." in os.path.basename(path):
            return os.path.dirname(path)
        return path
    return ""


_SINGLE_IMPORT_RE = re.compile(r'^\s*import\s+(?:[A-Za-z0-9_.]+\s+)?"([^"]+)"')
_GROUP_START_RE = re.compile(r"^\s*import\s*\(")
_IMPORT_PATH_RE = re.compile(r'^\s*(?:[A-Za-z0-9_.]+\s+)?"([^"]+)"')


def extract_go_imports(file_path: str) -> set[str]:
    imports: set[str] = set()
    in_group = False
    try:
        with open(file_path, "r", encoding="utf-8", errors="replace") as f:
            for line in f:
                stripped = line.strip()
                if not stripped or stripped.startswith("//"):
                    continue
                if in_group:
                    if stripped.startswith(")"):
                        in_group = False
                        continue
                    m = _IMPORT_PATH_RE.match(line)
                    if m:
                        imports.add(m.group(1))
                    continue
                m = _SINGLE_IMPORT_RE.match(line)
                if m:
                    imports.add(m.group(1))
                    continue
                if _GROUP_START_RE.match(line):
                    in_group = True
                    continue
                if (
                    stripped.startswith("func ")
                    or stripped.startswith("type ")
                    or stripped.startswith("var ")
                    or stripped.startswith("const ")
                ):
                    break
    except OSError:
        pass
    return imports


def count_internal_importers(
    repo_root: str, module_path: str, target_pkg_dirs: set[str]
) -> dict[str, int]:
    counts: dict[str, set[str]] = {pkg: set() for pkg in target_pkg_dirs}
    if not module_path or not target_pkg_dirs:
        return {pkg: 0 for pkg in target_pkg_dirs}

    full_to_pkg = {f"{module_path}/{pkg}": pkg for pkg in target_pkg_dirs}
    target_full_set = set(full_to_pkg.keys())

    for root, dirs, files in os.walk(repo_root):
        dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS and not d.startswith("."))
        go_files = [fn for fn in files if fn.endswith(".go")]
        if not go_files:
            continue
        rel_dir = os.path.relpath(root, repo_root).replace("\\", "/")
        if rel_dir == ".":
            rel_dir = ""
        for fn in go_files:
            file_imports = extract_go_imports(os.path.join(root, fn))
            matched = file_imports & target_full_set
            for full_imp in matched:
                target_pkg = full_to_pkg[full_imp]
                if rel_dir != target_pkg:
                    counts[target_pkg].add(rel_dir)

    return {pkg: len(importers) for pkg, importers in counts.items()}


def count_lines(abs_path: str) -> int:
    try:
        with open(abs_path, "r", encoding="utf-8", errors="replace") as f:
            return sum(1 for _ in f)
    except OSError:
        return 0


def check_large_files(repo_root: str, paths: list[str]) -> list[tuple[str, int]]:
    hits: list[tuple[str, int]] = []
    seen: set[str] = set()
    for p in paths:
        abs_p = os.path.join(repo_root, p)
        if os.path.isfile(abs_p) and p not in seen:
            seen.add(p)
            n = count_lines(abs_p)
            if n > 600:
                hits.append((p, n))
        elif os.path.isdir(abs_p):
            try:
                for entry in sorted(os.listdir(abs_p)):
                    if entry.startswith("."):
                        continue
                    child_abs = os.path.join(abs_p, entry)
                    child_rel = f"{p}/{entry}"
                    if os.path.isfile(child_abs) and child_rel not in seen:
                        seen.add(child_rel)
                        n = count_lines(child_abs)
                        if n > 600:
                            hits.append((child_rel, n))
            except OSError:
                pass
    return hits


def check_recent_churn(repo_root: str, paths: list[str]) -> list[tuple[str, int]]:
    hits: list[tuple[str, int]] = []
    for p in paths:
        abs_p = os.path.join(repo_root, p)
        if not os.path.exists(abs_p):
            continue
        try:
            proc = subprocess.run(
                [
                    "git",
                    "-C",
                    repo_root,
                    "log",
                    "--since=30 days ago",
                    "--format=%H",
                    "--",
                    p,
                ],
                capture_output=True,
                text=True,
                check=False,
            )
            if proc.returncode == 0:
                commits = [line for line in proc.stdout.splitlines() if line.strip()]
                if len(commits) >= 5:
                    hits.append((p, len(commits)))
        except OSError:
            pass
    return hits


def map_elo_to_band(total_elo: int) -> tuple[str, list[str]]:
    for threshold, band, models in BANDS:
        if total_elo >= threshold:
            return band, models
    return "Routine", ["gemini-4-hydrogen-medium", "gemini-4-hydrogen-high"]


def main() -> int:
    repo_root = sys.argv[1]
    raw_args = sys.argv[2:]

    as_json = False
    raw_paths: list[str] = []
    for arg in raw_args:
        if arg in ("-h", "--help"):
            usage()
            return 0
        elif arg == "--json":
            as_json = True
        elif arg.startswith("-"):
            print(f"error: unknown option: {arg}", file=sys.stderr)
            usage()
            return 2
        else:
            raw_paths.append(arg)

    if not raw_paths:
        print("error: at least one target path is required", file=sys.stderr)
        usage()
        return 2

    paths = [normalize_path(p, repo_root) for p in raw_paths if p.strip()]
    if not paths:
        print("error: no valid target paths provided", file=sys.stderr)
        return 2

    # 1. Base ELO (highest-risk location touched)
    base_elo = -1
    base_tier = ""
    base_rule = ""
    base_path = ""
    for p in paths:
        elo, tier, rule = classify_base_elo(p)
        if elo > base_elo:
            base_elo = elo
            base_tier = tier
            base_rule = rule
            base_path = p

    # 2. Structural modifiers
    modifiers = []

    # +50: any existing target file > 600 lines
    large_files = check_large_files(repo_root, paths)
    large_triggered = len(large_files) > 0
    large_detail = (
        ", ".join(f"{p} ({n}L)" for p, n in large_files)
        if large_triggered
        else "no target file > 600 lines"
    )
    modifiers.append(
        {
            "name": "large_file_near_800l",
            "delta": 50 if large_triggered else 0,
            "max_delta": 50,
            "triggered": large_triggered,
            "detail": large_detail,
        }
    )

    # +50: any target Go package has >= 10 internal importers
    module_path = read_go_module(repo_root)
    target_go_pkgs = {
        pkg for p in paths if (pkg := target_go_pkg_dir(p, repo_root))
    }
    importer_counts = count_internal_importers(repo_root, module_path, target_go_pkgs)
    high_fanin = sorted(
        ((pkg, cnt) for pkg, cnt in importer_counts.items() if cnt >= 10),
        key=lambda x: (-x[1], x[0]),
    )
    fanin_triggered = len(high_fanin) > 0
    if fanin_triggered:
        fanin_detail = ", ".join(
            f"{pkg} ({cnt} internal importers)" for pkg, cnt in high_fanin
        )
    elif importer_counts:
        top_pkg, top_cnt = max(importer_counts.items(), key=lambda x: (x[1], x[0]))
        fanin_detail = f"max {top_pkg} ({top_cnt} internal importers, < 10)"
    else:
        fanin_detail = "no Go target packages"
    modifiers.append(
        {
            "name": "high_fan_in_go_pkg",
            "delta": 50 if fanin_triggered else 0,
            "max_delta": 50,
            "triggered": fanin_triggered,
            "detail": fanin_detail,
        }
    )

    # +75: paths span >= 3 distinct top-level packages
    top_pkgs = sorted({top_level_package(p) for p in paths})
    cross_pkg_triggered = len(top_pkgs) >= 3
    cross_pkg_detail = f"{len(top_pkgs)} package(s): {', '.join(top_pkgs)}"
    modifiers.append(
        {
            "name": "cross_package_span",
            "delta": 75 if cross_pkg_triggered else 0,
            "max_delta": 75,
            "triggered": cross_pkg_triggered,
            "detail": cross_pkg_detail,
        }
    )

    # +25: any target file has >= 5 commits in last 30 days
    churn_files = check_recent_churn(repo_root, paths)
    churn_triggered = len(churn_files) > 0
    churn_detail = (
        ", ".join(f"{p} ({cnt} commits/30d)" for p, cnt in churn_files)
        if churn_triggered
        else "no target path with >= 5 commits in last 30d"
    )
    modifiers.append(
        {
            "name": "recent_churn_30d",
            "delta": 25 if churn_triggered else 0,
            "max_delta": 25,
            "triggered": churn_triggered,
            "detail": churn_detail,
        }
    )

    modifier_total = sum(m["delta"] for m in modifiers)
    total_elo = base_elo + modifier_total
    band, models = map_elo_to_band(total_elo)

    result = {
        "paths": paths,
        "base_elo": base_elo,
        "base_tier": base_tier,
        "base_rule": base_rule,
        "base_source": base_path,
        "modifier_total": modifier_total,
        "modifiers": modifiers,
        "total_elo": total_elo,
        "band": band,
        "recommended_models": models,
        "top_level_packages": top_pkgs,
    }

    if as_json:
        print(json.dumps(result, indent=2))
        return 0

    print(f"Task ELO Estimate: {total_elo} ({band})")
    print(f"Recommended Models: {' / '.join(models)}")
    print(f"Targets ({len(paths)}): {', '.join(paths)}")
    print("")
    print("| Signal | Delta | Status | Detail |")
    print("|--------|------:|--------|--------|")
    print(
        f"| Base location (`{base_rule}`) | {base_elo} | {base_tier} | `{base_path}` |"
    )
    for m in modifiers:
        delta_str = f"+{m['delta']}" if m["triggered"] else "+0"
        status = "triggered" if m["triggered"] else "clear"
        print(f"| `{m['name']}` | {delta_str} | {status} | {m['detail']} |")
    print(f"| **Total ELO** | **{total_elo}** | **{band}** | `{' / '.join(models)}` |")
    return 0


if __name__ == "__main__":
    sys.exit(main())
PYEOF
