"""Run every fine-tune training-data reader's real entry point against a
fixture (finetune_heldout_test.go). For each reader, print one JSON line
{"reader": name, "admitted": [benchmark ids it would admit to training]}.

argv: <tools/finetune dir> <fixture dir> <yaml stub dir>

The fixture dir holds heldout.json, a repo root (eval_results/, benchmarks/)
and traces. Everything that would touch the network, a model or the real
repo is replaced: no collection, training or eval runs. The held-out filter
itself is never replaced, so a reader that re-reads heldout.json without
retired_excluded admits the retired id and the test fails.
"""
import argparse
import json
import os
import sys

FT, FX, STUB = sys.argv[1:4]
sys.path[:0] = [STUB, FT]

import extract_lang  # noqa: E402

IDS = ["held_core", "task", "task_v2", "other"]
REPO = os.path.join(FX, "repo")
for mod in [extract_lang]:
    mod.HERE = FX          # heldout.json fixture
    mod.REPO = REPO

import extract_traj  # noqa: E402
import onpolicy_pairs  # noqa: E402
import probe_after_read  # noqa: E402
import stale_fix  # noqa: E402
import synth_episodes  # noqa: E402

for mod in (extract_traj, onpolicy_pairs, probe_after_read, stale_fix, synth_episodes):
    mod.REPO = REPO
    mod.HERE = FX


def out(reader, admitted):
    print(json.dumps({"reader": reader, "admitted": sorted(set(admitted))}), flush=True)


def run_main(mod, argv):
    sys.argv = [mod.__file__] + argv
    try:
        mod.main()
        return None
    except SystemExit as e:
        return str(e.code)


def outdir(name):
    d = os.path.join(FX, "out", name)
    os.makedirs(d, exist_ok=True)
    return d


# extract_lang: eval_results rows -> eval.jsonl
extract_lang.AILANG = sys.executable                     # report header: `python --version`
extract_lang.check_eval_program = lambda code: (True, "")
extract_lang.check_first_passing = lambda paths: (paths[0], True, "")
extract_lang.task_prompt = lambda bid: ""
d = outdir("extract_lang")
run_main(extract_lang, ["--out", d, "--roots", os.path.join(FX, "empty")])
out("extract_lang", [json.loads(l)["benchmark"] for l in open(os.path.join(d, "eval.jsonl"))])

# stale_fix.collect: eval_results rows -> stale.jsonl (every candidate "fails" today)
stale_fix.spec = lambda bid, cache=None: {"id": bid}
stale_fix.check = lambda code: (False, "stale")
d = outdir("stale_fix")
stale_fix.collect(argparse.Namespace(out=d))
out("stale_fix", [json.loads(l)["benchmark"] for l in open(os.path.join(d, "stale.jsonl"))])

# onpolicy_pairs: one trace per benchmark; a refused trace admits nothing
onpolicy_pairs.spec = lambda bid: None
admitted = []
for bid in IDS:
    trace = os.path.join(FX, "traces", "onpolicy_" + bid + ".jsonl")
    msg = run_main(onpolicy_pairs, ["--trace", trace, "--out", os.path.join(outdir("onpolicy"), bid + ".jsonl")])
    if not (msg and "held-out" in msg):
        admitted.append(bid)
out("onpolicy_pairs", admitted)

# probe_after_read: stops before any network call
def no_network(*a, **k):
    raise SystemExit("would-probe")
probe_after_read.first_action = no_network
admitted = []
for bid in IDS:
    trace = os.path.join(FX, "traces", "probe_" + bid + ".jsonl")
    msg = run_main(probe_after_read, ["--trace", trace, "--model", "none", "-n", "1"])
    if not (msg and "held-out" in msg):
        admitted.append(bid)
out("probe_after_read", admitted)

# synth_episodes: the benchmarks it would render tasks for
captured = []
def capture(bids):
    captured.extend(bids)
    raise SystemExit("stop")
synth_episodes.render_tasks = capture
run_main(synth_episodes, ["--out", outdir("synth"), "--lang-data", os.path.join(FX, "lang"),
                          "--stale", os.path.join(FX, "empty")])
out("synth_episodes", captured)

# extract_traj: opencode sessions, one per benchmark. The session's steps
# record that they were looked at, which happens only after the held-out
# filter let the session through (and are empty, so nothing is emitted).
passed = []


class Steps:
    def __init__(self, bid):
        self.bid = bid

    def __bool__(self):
        passed.append(self.bid)
        return False


extract_traj.iter_opencode = lambda: iter([("s-" + bid, bid, Steps(bid)) for bid in IDS])
run_main(extract_traj, ["--out", outdir("extract_traj"), "--sources", "opencode", "--name", "t"])
out("extract_traj", passed)
