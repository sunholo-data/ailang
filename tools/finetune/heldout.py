"""The fine-tune held-out set (heldout.json), read in one place.

Two roles: core + stretch is the before/after EVAL list (the pilot scripts
pass it to --benchmarks); core + stretch + retired_excluded is everything
excluded from TRAINING data, so a retired held-out task's banked solutions
stay out after its successor takes its place in the eval list
(amendment 2026-11-11, D-FT-17). Standard library only: collect_onpolicy.sh
imports it under `python3 -I`. internal/eval_harness/finetune_heldout_test.go
runs every reader against a fixture.
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))


def _load(path=None):
    with open(path or os.path.join(HERE, "heldout.json")) as f:
        return json.load(f)


def load_heldout(path=None):
    """Every benchmark id excluded from TRAINING data."""
    h = _load(path)
    return set(h["core"] + h["stretch"] + h.get("retired_excluded", []))


def eval_ids(path=None):
    """The held-out EVAL list (core + stretch), in file order."""
    h = _load(path)
    return h["core"] + h["stretch"]
