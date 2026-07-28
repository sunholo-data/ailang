#!/bin/bash
# Find language gaps from Claude session JSONL files
# Parses tool calls for repeated failed searches that indicate missing features
# Usage: find_gaps.sh

BENCH_DIR="/Users/mark/dev/sunholo/ai-coding-lang-bench"

echo "════════════════════════════════════════════════════════════"
echo "  AILANG Language Gap Analysis"
echo "════════════════════════════════════════════════════════════"
echo ""

python3 - "$BENCH_DIR" <<'PYEOF'
import json, sys, os, re
from collections import defaultdict

bench = sys.argv[1]
logs_dir = os.path.join(bench, "logs")

# Gap patterns to search for in tool calls
gap_patterns = {
    "exit/process exit": re.compile(r"exit|exitCode|exitWith|process\.exit|sys\.exit|os\.Exit", re.I),
    "cons :: expression": re.compile(r"::\s|cons\b|prepend", re.I),
    "module loading": re.compile(r"module.*loading|MOD010|module.*mismatch|relax.modules", re.I),
    "mutable state": re.compile(r"mutable|var\b|ref\b|IORef|setState", re.I),
    "hash map/dict": re.compile(r"hash.?map|dict|Map\[|HashMap|assoc", re.I),
    "string to bytes": re.compile(r"toBytes|fromBytes|encode|decode.*utf", re.I),
    "file permissions": re.compile(r"chmod|permissions|executable|shebang", re.I),
    "error/exception": re.compile(r"throw|raise|exception|try.*catch|panic", re.I),
}

gaps = defaultdict(lambda: {"count": 0, "trials": set(), "examples": []})

for fname in sorted(os.listdir(logs_dir)):
    if not fname.startswith("session-") or not fname.endswith(".jsonl"):
        continue

    trial_name = fname.replace("session-", "").replace(".jsonl", "")
    fpath = os.path.join(logs_dir, fname)

    for line in open(fpath):
        try:
            msg = json.loads(line)
            m = msg.get("message", {})
            content = m.get("content", [])
            if not isinstance(content, list):
                continue
            for block in content:
                if block.get("type") == "tool_use":
                    inp = block.get("input", {})
                    cmd = inp.get("command", "") + inp.get("pattern", "") + inp.get("file_path", "")
                    for gap_name, pattern in gap_patterns.items():
                        if pattern.search(cmd):
                            gaps[gap_name]["count"] += 1
                            gaps[gap_name]["trials"].add(trial_name)
                            if len(gaps[gap_name]["examples"]) < 3:
                                gaps[gap_name]["examples"].append(f"{trial_name}: {cmd[:80]}")
        except:
            pass

if not gaps:
    print("No session JSONL files found. Run trials first, then re-analyze.")
    sys.exit(0)

# Sort by count descending
for gap_name, data in sorted(gaps.items(), key=lambda x: -x[1]["count"]):
    count = data["count"]
    trials = len(data["trials"])
    print(f"  {gap_name} ({count} searches across {trials} trial(s))")
    for ex in data["examples"]:
        print(f"    - {ex}")
    print()

# Check for design docs
print("Design docs for identified gaps:")
print("-" * 50)
dd_dir = os.path.join(os.path.dirname(bench), "ailang", "design_docs", "planned", "v0_10_0")
if os.path.isdir(dd_dir):
    for f in sorted(os.listdir(dd_dir)):
        if f.startswith("m-") and f.endswith(".md") and "sprint" not in f:
            print(f"  [exists] {f}")
else:
    print(f"  Design docs dir not found at {dd_dir}")
PYEOF
