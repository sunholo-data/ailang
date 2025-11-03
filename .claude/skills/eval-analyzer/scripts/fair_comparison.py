#!/usr/bin/env python3
"""Fair comparison of v0.4.0 vs v0.4.2 (dev models only, AILANG only)"""

import json
from pathlib import Path
from collections import defaultdict

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

def compare_versions():
    dev_models = ['gpt5-mini', 'claude-haiku-4-5', 'gemini-2-5-flash']
    
    v040_results = load_results("eval_results/baselines/v0.4.0")
    v042_results = load_results("eval_results/baselines/v0.4.2")
    
    # Filter to dev models only
    v040_dev = [r for r in v040_results if r.get('model') in dev_models]
    v042_dev = [r for r in v042_results if r.get('model') in dev_models]
    
    # Deduplicate v0.4.0 (keep last run per benchmark+model)
    v040_dedup = {}
    for result in v040_dev:
        key = (result.get('id'), result.get('model'))
        v040_dedup[key] = result  # Last one wins
    
    v042_lookup = {(r.get('id'), r.get('model')): r for r in v042_dev}
    
    # Compare
    fixed = []
    broken = []
    still_passing = []
    still_failing = []
    
    for key, v040_result in v040_dedup.items():
        if key not in v042_lookup:
            continue
        
        v042_result = v042_lookup[key]
        
        v040_ok = v040_result.get('stdout_ok', False)
        v042_ok = v042_result.get('stdout_ok', False)
        
        if not v040_ok and v042_ok:
            fixed.append(key)
        elif v040_ok and not v042_ok:
            broken.append(key)
        elif v040_ok and v042_ok:
            still_passing.append(key)
        else:
            still_failing.append(key)
    
    v040_pass = len([r for r in v040_dedup.values() if r.get('stdout_ok')])
    v042_pass = len([r for r in v042_lookup.values() if r.get('stdout_ok')])
    
    print("=== FAIR COMPARISON (dev models, AILANG only, deduplicated) ===\n")
    print(f"v0.4.0: {v040_pass}/{len(v040_dedup)} = {100*v040_pass/len(v040_dedup):.1f}%")
    print(f"v0.4.2: {v042_pass}/{len(v042_lookup)} = {100*v042_pass/len(v042_lookup):.1f}%")
    print(f"Delta:  {v042_pass - v040_pass:+d} ({100*(v042_pass - v040_pass)/len(v040_dedup):+.1f}pp)\n")
    
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
        for bench_id, model in sorted(broken)[:10]:
            v042_err = v042_lookup[(bench_id, model)].get('error_category', 'unknown')
            print(f"  ❌ {bench_id} ({model}): {v042_err}")
        if len(broken) > 10:
            print(f"  ... and {len(broken) - 10} more")
        print()
    
    # Per-model breakdown
    print("=== PER-MODEL BREAKDOWN ===\n")
    for model in sorted(dev_models):
        v040_model = {k: v for k, v in v040_dedup.items() if k[1] == model}
        v042_model = {k: v for k, v in v042_lookup.items() if k[1] == model}
        
        v040_model_pass = sum(1 for r in v040_model.values() if r.get('stdout_ok'))
        v042_model_pass = sum(1 for r in v042_model.values() if r.get('stdout_ok'))
        
        print(f"{model}:")
        print(f"  v0.4.0: {v040_model_pass}/{len(v040_model)} = {100*v040_model_pass/len(v040_model):.1f}%")
        print(f"  v0.4.2: {v042_model_pass}/{len(v042_model)} = {100*v042_model_pass/len(v042_model):.1f}%")
        print(f"  Delta:  {v042_model_pass - v040_model_pass:+d} ({100*(v042_model_pass - v040_model_pass)/len(v040_model):+.1f}pp)\n")

if __name__ == '__main__':
    compare_versions()
