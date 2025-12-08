#!/usr/bin/env python3
"""
Validate eval results for output corruption and race conditions.

This script checks for:
1. Output mismatches (benchmark A outputting B's expected output)
2. Suspicious patterns (fibonacci outputting "All results equal", etc.)
3. Code hash validation (if available)
4. Duplicate benchmark IDs within same model
"""

import json
import sys
from pathlib import Path
from collections import defaultdict
from typing import Dict, List, Tuple

# Known benchmark output patterns
EXPECTED_PATTERNS = {
    "recursion_fibonacci": "6765",
    "referential_transparency": "All results equal: true",
    "simple_print": "Hello",
    "exhaustive_pattern_matching": ["Waiting to start", "Currently working", "All done"],
}

# Suspicious cross-contamination patterns
SUSPICIOUS_PATTERNS = {
    "recursion_fibonacci": ["All results equal"],  # Should never output this
    "referential_transparency": ["6765"],          # Should never output fibonacci
    "simple_print": ["6765", "All results equal"], # Should never output these
}


def load_results(results_dir: Path) -> List[Dict]:
    """Load all JSON result files"""
    results = []

    # Check both standard/ and agent/ subdirectories
    for subdir in ["standard", "agent"]:
        subpath = results_dir / subdir
        if not subpath.exists():
            continue

        for json_file in subpath.glob("*.json"):
            try:
                with open(json_file) as f:
                    data = json.load(f)
                    data["_file"] = str(json_file)
                    results.append(data)
            except Exception as e:
                print(f"⚠️  Error loading {json_file}: {e}", file=sys.stderr)

    return results


def check_output_corruption(results: List[Dict]) -> Tuple[int, List[str]]:
    """Check for output corruption patterns"""
    issues = []
    corruption_count = 0

    for result in results:
        if result.get("lang") != "ailang":
            continue

        bench_id = result.get("id", "unknown")
        stdout = result.get("stdout", "")

        # Check suspicious patterns
        if bench_id in SUSPICIOUS_PATTERNS:
            for suspicious in SUSPICIOUS_PATTERNS[bench_id]:
                if suspicious in stdout:
                    issues.append(
                        f"❌ {bench_id} ({result.get('model')}): "
                        f"Contains suspicious pattern '{suspicious}'"
                    )
                    corruption_count += 1
                    break

    return corruption_count, issues


def check_duplicates(results: List[Dict]) -> Tuple[int, List[str]]:
    """Check for duplicate benchmark runs (same benchmark+model in same eval)"""
    issues = []
    duplicate_count = 0

    # Group by (benchmark_id, model, lang)
    groups = defaultdict(list)
    for result in results:
        key = (result.get("id"), result.get("model"), result.get("lang"))
        groups[key].append(result)

    for key, runs in groups.items():
        if len(runs) > 1:
            bench_id, model, lang = key
            issues.append(
                f"⚠️  Duplicate runs detected: {bench_id} ({model}, {lang}) "
                f"has {len(runs)} runs"
            )
            duplicate_count += 1

    return duplicate_count, issues


def check_code_hash(results: List[Dict]) -> Tuple[int, List[str]]:
    """Check if code hash validation is present (for debugging)"""
    issues = []
    hash_count = 0

    for result in results:
        if "code_hash" in result or "CodeHash" in result:
            hash_count += 1

    if hash_count == 0:
        issues.append(
            "ℹ️  No code hash validation found in results "
            "(this is expected for older eval runs)"
        )

    return hash_count, issues


def analyze_success_rate(results: List[Dict]) -> None:
    """Print success rate statistics"""
    ailang_results = [r for r in results if r.get("lang") == "ailang"]

    if not ailang_results:
        print("No AILANG results found")
        return

    total = len(ailang_results)
    passed = sum(1 for r in ailang_results if r.get("stdout_ok"))

    print(f"\n=== Success Rate ===")
    print(f"Total AILANG runs: {total}")
    print(f"Passed: {passed}")
    print(f"Success rate: {100 * passed / total:.1f}%")


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <results_directory>")
        print(f"Example: {sys.argv[0]} eval_results/baselines/v0.4.2")
        sys.exit(1)

    results_dir = Path(sys.argv[1])
    if not results_dir.exists():
        print(f"Error: Directory not found: {results_dir}", file=sys.stderr)
        sys.exit(1)

    print(f"=== Validating Eval Results ===")
    print(f"Directory: {results_dir}\n")

    # Load results
    results = load_results(results_dir)
    print(f"Loaded {len(results)} result files\n")

    if not results:
        print("No results found")
        sys.exit(1)

    # Run checks
    total_issues = 0

    print("=== Output Corruption Check ===")
    corruption_count, corruption_issues = check_output_corruption(results)
    if corruption_issues:
        for issue in corruption_issues:
            print(issue)
        total_issues += corruption_count
    else:
        print("✅ No output corruption detected")

    print("\n=== Duplicate Check ===")
    duplicate_count, duplicate_issues = check_duplicates(results)
    if duplicate_issues:
        for issue in duplicate_issues:
            print(issue)
        total_issues += duplicate_count
    else:
        print("✅ No duplicate runs detected")

    print("\n=== Code Hash Check ===")
    hash_count, hash_issues = check_code_hash(results)
    if hash_issues:
        for issue in hash_issues:
            print(issue)
    else:
        print(f"✅ Code hash validation present in {hash_count}/{len(results)} results")

    # Print summary
    analyze_success_rate(results)

    print("\n=== Final Summary ===")
    if total_issues == 0:
        print("✅ All validation checks PASSED")
        print("✅ No corruption or race condition issues detected")
        sys.exit(0)
    else:
        print(f"❌ Found {total_issues} issues")
        print("❌ Results may be corrupted or contain race conditions")
        sys.exit(1)


if __name__ == "__main__":
    main()
