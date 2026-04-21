#!/usr/bin/env python3
"""Fair comparison of eval baselines (dev models only, AILANG only, deduplicated)

Defaults to comparing the two most recent baselines.
Use --v1 and --v2 to specify custom versions.
"""

import json
import argparse
import re
from pathlib import Path
from collections import defaultdict


def _find_repo_root(start: Path) -> Path | None:
    """Walk up looking for AILANG repo markers."""
    for d in [start, *start.parents]:
        if (d / "go.mod").exists() and (d / "internal/eval_harness").exists():
            return d
    return None


def resolve_dev_models() -> list[str]:
    """Read dev_models from internal/eval_harness/models.yml — single source of truth.

    Falls back to an empty list if the file is missing; callers must either
    accept --models explicitly or fail loudly. No hardcoded defaults.
    """
    root = _find_repo_root(Path(__file__).resolve())
    if root is None:
        return []
    yml = root / "internal/eval_harness/models.yml"
    if not yml.exists():
        return []
    models: list[str] = []
    in_block = False
    for line in yml.read_text().splitlines():
        if line.startswith("dev_models:"):
            in_block = True
            continue
        if in_block:
            m = re.match(r"\s*-\s*\"?([^\"#\s]+)\"?", line)
            if m:
                models.append(m.group(1))
            elif line.strip() and not line.startswith(" "):
                break  # next top-level key
    return models


def load_results(results_dir):
    """Load result files from standard/ subdirectory"""
    results = []
    standard_dir = Path(results_dir) / "standard"

    if not standard_dir.exists():
        print(f"Warning: {standard_dir} doesn't exist")
        return results

    for json_file in standard_dir.glob("*_ailang_*.json"):
        try:
            with open(json_file) as f:
                data = json.load(f)
                results.append(data)
        except Exception as e:
            print(f"Error loading {json_file}: {e}")

    return results

def get_latest_baselines(baselines_dir="eval_results/baselines", n=2):
    """Get the n most recent baseline directories by modification time"""
    baselines_path = Path(baselines_dir)

    if not baselines_path.exists():
        return []

    # Get all directories that start with 'v' (version directories)
    baseline_dirs = [d for d in baselines_path.iterdir()
                     if d.is_dir() and d.name.startswith('v')]

    # Sort by modification time (most recent first)
    baseline_dirs.sort(key=lambda d: d.stat().st_mtime, reverse=True)

    return baseline_dirs[:n]

def compare_versions(v1_dir, v2_dir, dev_models=None):
    """Compare two versions with fair filtering (dev models, AILANG, deduplicated)"""
    if dev_models is None:
        dev_models = resolve_dev_models()
    if not dev_models:
        raise RuntimeError(
            "No dev_models resolved — pass --models or run from inside the AILANG repo"
        )

    v1_name = Path(v1_dir).name
    v2_name = Path(v2_dir).name

    v1_results = load_results(v1_dir)
    v2_results = load_results(v2_dir)

    if not v1_results:
        print(f"Error: No results found in {v1_dir}/standard/")
        return

    if not v2_results:
        print(f"Error: No results found in {v2_dir}/standard/")
        return

    # Filter to dev models only
    v1_dev = [r for r in v1_results if r.get('model') in dev_models]
    v2_dev = [r for r in v2_results if r.get('model') in dev_models]

    # Deduplicate v1 (keep last run per benchmark+model)
    v1_dedup = {}
    for result in v1_dev:
        key = (result.get('id'), result.get('model'))
        v1_dedup[key] = result  # Last one wins

    v2_lookup = {(r.get('id'), r.get('model')): r for r in v2_dev}

    # Compare
    fixed = []
    broken = []
    still_passing = []
    still_failing = []

    for key, v1_result in v1_dedup.items():
        if key not in v2_lookup:
            continue

        v2_result = v2_lookup[key]

        v1_ok = v1_result.get('stdout_ok', False)
        v2_ok = v2_result.get('stdout_ok', False)

        if not v1_ok and v2_ok:
            fixed.append(key)
        elif v1_ok and not v2_ok:
            broken.append((key, v2_result.get('error_category', 'unknown')))
        elif v1_ok and v2_ok:
            still_passing.append(key)
        else:
            still_failing.append(key)

    v1_pass = len([r for r in v1_dedup.values() if r.get('stdout_ok')])
    v2_pass = len([r for r in v2_lookup.values() if r.get('stdout_ok')])

    print("=== FAIR COMPARISON (dev models, AILANG only, deduplicated) ===\n")
    print(f"{v1_name}: {v1_pass}/{len(v1_dedup)} = {100*v1_pass/len(v1_dedup):.1f}%")
    print(f"{v2_name}: {v2_pass}/{len(v2_lookup)} = {100*v2_pass/len(v2_lookup):.1f}%")
    print(f"Delta:  {v2_pass - v1_pass:+d} ({100*(v2_pass - v1_pass)/len(v1_dedup):+.1f}pp)\n")

    print(f"✅ Fixed:           {len(fixed)} benchmarks")
    print(f"❌ Broken:          {len(broken)} benchmarks")
    print(f"→  Still passing:   {len(still_passing)} benchmarks")
    print(f"→  Still failing:   {len(still_failing)} benchmarks")
    print(f"NET:               {len(fixed) - len(broken):+d} benchmarks\n")

    if fixed:
        print("Fixed benchmarks:")
        for bench_id, model in sorted(fixed)[:10]:
            print(f"  ✅ {bench_id} ({model})")
        if len(fixed) > 10:
            print(f"  ... and {len(fixed) - 10} more")
        print()

    if broken:
        print("Broken benchmarks:")
        for (bench_id, model), err_cat in sorted(broken)[:10]:
            print(f"  ❌ {bench_id} ({model}): {err_cat}")
        if len(broken) > 10:
            print(f"  ... and {len(broken) - 10} more")
        print()

    # Per-model breakdown
    print("=== PER-MODEL BREAKDOWN ===\n")
    for model in sorted(dev_models):
        v1_model = {k: v for k, v in v1_dedup.items() if k[1] == model}
        v2_model = {k: v for k, v in v2_lookup.items() if k[1] == model}

        if not v1_model or not v2_model:
            continue

        v1_model_pass = sum(1 for r in v1_model.values() if r.get('stdout_ok'))
        v2_model_pass = sum(1 for r in v2_model.values() if r.get('stdout_ok'))

        print(f"{model}:")
        print(f"  {v1_name}: {v1_model_pass}/{len(v1_model)} = {100*v1_model_pass/len(v1_model):.1f}%")
        print(f"  {v2_name}: {v2_model_pass}/{len(v2_model)} = {100*v2_model_pass/len(v2_model):.1f}%")
        print(f"  Delta:  {v2_model_pass - v1_model_pass:+d} ({100*(v2_model_pass - v1_model_pass)/len(v1_model):+.1f}pp)\n")

def main():
    parser = argparse.ArgumentParser(
        description='Fair comparison of AILANG eval baselines (dev models only, deduplicated)',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Compare the two most recent baselines (default)
  %(prog)s

  # Compare specific versions
  %(prog)s --v1 v0.4.5 --v2 v0.4.6

  # Compare with full paths
  %(prog)s --v1 eval_results/baselines/v0.4.5 --v2 eval_results/baselines/v0.4.6

  # Use custom dev models
  %(prog)s --models gpt5,claude-sonnet-4-5,gemini-2-5-pro
"""
    )

    parser.add_argument('--v1',
                        help='First version to compare (default: second-most recent baseline)')
    parser.add_argument('--v2',
                        help='Second version to compare (default: most recent baseline)')
    parser.add_argument('--baselines-dir',
                        default='eval_results/baselines',
                        help='Directory containing baseline subdirectories (default: eval_results/baselines)')
    parser.add_argument('--models',
                        help='Comma-separated model list. Default: dev_models from internal/eval_harness/models.yml')

    args = parser.parse_args()

    # Resolve dev models — prefer --models, else read models.yml (single source of truth)
    if args.models:
        dev_models = [m.strip() for m in args.models.split(',')]
    else:
        dev_models = resolve_dev_models()
        if not dev_models:
            print("Error: could not resolve dev_models from internal/eval_harness/models.yml")
            print("Pass --models explicitly, or run from inside the AILANG repo.")
            return

    # Determine which versions to compare
    if args.v1 and args.v2:
        # Both versions specified
        v1_path = args.v1 if '/' in args.v1 else f"{args.baselines_dir}/{args.v1}"
        v2_path = args.v2 if '/' in args.v2 else f"{args.baselines_dir}/{args.v2}"
    else:
        # Default to two most recent
        latest = get_latest_baselines(args.baselines_dir, n=2)

        if len(latest) < 2:
            print(f"Error: Need at least 2 baseline directories in {args.baselines_dir}")
            print(f"Found: {[d.name for d in latest]}")
            return

        v2_path = str(latest[0])  # Most recent
        v1_path = str(latest[1])  # Second most recent

        print(f"Auto-detected latest baselines:")
        print(f"  v1 (older):  {latest[1].name}")
        print(f"  v2 (newer):  {latest[0].name}")
        print()

    compare_versions(v1_path, v2_path, dev_models)

if __name__ == '__main__':
    main()
