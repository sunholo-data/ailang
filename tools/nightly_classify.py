#!/usr/bin/env python3
"""Classify persistent nightly eval failures using durable trial history."""

from __future__ import annotations

import argparse
import glob
import json
import logging
import os
import re
import secrets
import shutil
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}
DATE_RE = re.compile(r"nightly_eval_(\d{8})_rag_on")
MODEL_RE = re.compile(r"_ailang_(.+)_\d+$")
TRIAL_RE = re.compile(r"_trial\d+$")
LOGGER = logging.getLogger("nightly-classify")


@dataclass
class Verdict:
    label: str
    bench: str
    cats: list[str]
    passes: int
    trials: int
    nights: int
    consecutive: int
    escalated_from: str = ""

    def tsv(self) -> str:
        cats = f"[{','.join(self.cats)}]"
        return "\t".join(
            (
                self.label,
                self.bench,
                cats,
                f"{self.passes}/{self.trials}",
                str(self.nights),
                str(self.consecutive),
                self.escalated_from or "-",
            )
        )


def _slot_parts(slot: str) -> tuple[str, str]:
    before, marker, after = slot.partition("_ailang_")
    if not marker:
        return TRIAL_RE.sub("", slot), "unknown"
    return TRIAL_RE.sub("", before), after


def parse_results_dir(path: str | Path) -> dict[str, list[tuple[bool, str]]]:
    """Return newest trial slots grouped by benchmark."""
    latest: dict[str, tuple[int, str]] = {}
    for filename in glob.glob(os.path.join(str(path), "*.json")):
        base = os.path.basename(filename)
        try:
            timestamp = int(base.rsplit("_", 1)[1][:-5])
        except (IndexError, ValueError):
            continue
        slot = base.rsplit("_", 1)[0]
        if slot not in latest or timestamp > latest[slot][0]:
            latest[slot] = (timestamp, filename)

    trials: dict[str, list[tuple[bool, str]]] = {}
    for slot, (_, filename) in latest.items():
        try:
            with open(filename, encoding="utf-8") as handle:
                data = json.load(handle)
        except (OSError, json.JSONDecodeError):
            continue
        bench, _ = _slot_parts(slot)
        passed = bool(
            data.get("compile_ok")
            and data.get("runtime_ok")
            and data.get("stdout_ok")
        )
        trials.setdefault(bench, []).append(
            (passed, data.get("error_category") or "—")
        )
    return trials


def persistent_failures(
    results: dict[str, list[tuple[bool, str]]],
) -> dict[str, list[str]]:
    failures = {}
    for bench, trials in results.items():
        cats = sorted({cat for _, cat in trials})
        if (
            len(trials) >= 2
            and all(not passed for passed, _ in trials)
            and set(cats) - INFRA_CATEGORIES
        ):
            failures[bench] = cats
    return failures


def parse_date(path: str | Path) -> str:
    match = DATE_RE.search(str(path))
    if not match:
        raise ValueError(f"cannot parse nightly date from {path}")
    raw = match.group(1)
    return f"{raw[:4]}-{raw[4:6]}-{raw[6:]}"


def infer_model(path: str | Path) -> str:
    for filename in glob.glob(os.path.join(str(path), "*.json")):
        slot = os.path.basename(filename).rsplit("_", 1)[0]
        _, model = _slot_parts(slot)
        if model != "unknown":
            return model
    return "unknown"


def records_for_dir(
    path: str | Path, model: str | None = None, arm: str = "rag_on"
) -> list[dict]:
    date = parse_date(path)
    model = model or infer_model(path)
    records = []
    for bench, trials in sorted(parse_results_dir(path).items()):
        cats = sorted({cat for passed, cat in trials if not passed})
        records.append(
            {
                "date": date,
                "bench": bench,
                "model": model,
                "arm": arm,
                "trials": len(trials),
                "passes": sum(1 for passed, _ in trials if passed),
                "cats": cats,
                "class": "",
            }
        )
    return records


def record_key(record: dict) -> tuple[str, str, str, str]:
    return tuple(record[field] for field in ("date", "bench", "model", "arm"))


def load_history(path: str | Path) -> tuple[list[dict], int]:
    """Load valid records, resolving duplicate keys last-in-file-wins."""
    ordered: dict[tuple[str, str, str, str], dict] = {}
    skipped = 0
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            try:
                record = json.loads(line)
                key = record_key(record)
                if not all(isinstance(value, str) for value in key):
                    raise ValueError("non-string key")
                int(record["passes"])
                int(record["trials"])
            except (json.JSONDecodeError, KeyError, TypeError, ValueError):
                skipped += 1
                continue
            if key in ordered:
                del ordered[key]
            ordered[key] = record
    return list(ordered.values()), skipped


def history_health(path: str | Path, records: list[dict], skipped: int) -> str:
    benches = len({record["bench"] for record in records})
    dates = sorted({record["date"] for record in records})
    newest = dates[-1] if dates else "none"
    return (
        f"history: {path} | {benches} benchmarks, {len(dates)} nights, "
        f"newest {newest}, {skipped} skipped lines"
    )


def select_window(
    records: Iterable[dict],
    bench: str,
    model: str,
    arm: str,
    tonight: str,
    window_nights: int,
) -> list[dict]:
    eligible = [
        record
        for record in records
        if record["bench"] == bench
        and record["model"] == model
        and record["arm"] == arm
        and record["date"] < tonight
    ]
    dates = sorted({record["date"] for record in eligible}, reverse=True)[
        :window_nights
    ]
    return [record for record in eligible if record["date"] in dates]


def consecutive_failures(
    records: Iterable[dict], bench: str, model: str, arm: str, tonight: str
) -> tuple[int, bool]:
    """Count the current all-fail run, and whether it already paged."""
    relevant = sorted(
        (
            record
            for record in records
            if record["bench"] == bench
            and record["model"] == model
            and record["arm"] == arm
            and record["date"] < tonight
        ),
        key=lambda record: record["date"],
        reverse=True,
    )
    count = 1  # tonight is a persistent all-fail
    already_regressed = False
    for record in relevant:
        if int(record["trials"]) < 2 or int(record["passes"]) != 0:
            break
        count += 1
        already_regressed |= str(record.get("class", "")).lower() == "regression"
    return count, already_regressed


def classify_bench(
    bench: str,
    cats: list[str],
    records: list[dict],
    model: str,
    arm: str,
    tonight: str,
    window_nights: int = 5,
    min_nights: int = 2,
    min_trials: int = 4,
    escalate_after: int = 3,
) -> Verdict:
    window = select_window(records, bench, model, arm, tonight, window_nights)
    passes = sum(int(record["passes"]) for record in window)
    trials = sum(int(record["trials"]) for record in window)
    nights = len({record["date"] for record in window})
    consecutive, already_regressed = consecutive_failures(
        records, bench, model, arm, tonight
    )

    if nights < min_nights or trials < min_trials:
        label = "INSUFFICIENT-HISTORY"
    elif passes == trials:
        label = "REGRESSION"
    elif passes == 0:
        label = "GAP"
    else:
        label = "SUSPECTED-FLAKE"

    escalated_from = ""
    if (
        label not in {"REGRESSION", "GAP"}
        and consecutive >= escalate_after
        and not already_regressed
    ):
        escalated_from = label
        label = "REGRESSION"
    return Verdict(
        label, bench, cats, passes, trials, nights, consecutive, escalated_from
    )


def legacy_classify(
    tonight_dir: str | Path, previous_dir: str | Path | None
) -> list[str]:
    """The extracted pre-sprint classifier, retained for golden comparison."""
    previous = parse_results_dir(previous_dir) if previous_dir else {}
    lines = []
    for bench, cats in sorted(persistent_failures(parse_results_dir(tonight_dir)).items()):
        seen = previous.get(bench, [])
        solid = bool(seen) and all(passed for passed, _ in seen)
        label = "REGRESSION" if solid else "GAP"
        lines.append(f"{label}\t{bench}\t[{','.join(cats)}]")
    return lines


class HistoryLock:
    def __init__(
        self,
        lock_dir: str | Path,
        timeout: float = 60.0,
        stale_after: float = 600.0,
        poll: float = 0.1,
    ):
        self.path = Path(lock_dir)
        self.timeout = timeout
        self.stale_after = stale_after
        self.poll = poll
        self.token = secrets.token_hex(16)

    @property
    def metadata(self) -> Path:
        return self.path / "owner.json"

    def _read_owner(self) -> dict | None:
        try:
            with self.metadata.open(encoding="utf-8") as handle:
                owner = json.load(handle)
            if not isinstance(owner["pid"], int) or not isinstance(owner["token"], str):
                return None
            return owner
        except (OSError, json.JSONDecodeError, KeyError, TypeError):
            return None

    @staticmethod
    def _alive(pid: int) -> bool:
        try:
            os.kill(pid, 0)
            return True
        except ProcessLookupError:
            return False
        except PermissionError:
            return True

    def _age(self) -> float:
        try:
            return time.time() - self.path.stat().st_mtime
        except OSError:
            return 0

    def _try_recover(self) -> None:
        owner = self._read_owner()
        if owner is not None:
            if not self._alive(owner["pid"]):
                LOGGER.warning("stealing lock held by dead pid %s", owner["pid"])
                shutil.rmtree(self.path, ignore_errors=True)
            return
        if self._age() >= self.stale_after:
            LOGGER.warning("stealing stale lock with unreadable metadata")
            shutil.rmtree(self.path, ignore_errors=True)

    def acquire(self) -> None:
        deadline = time.monotonic() + self.timeout
        self.path.parent.mkdir(parents=True, exist_ok=True)
        while True:
            try:
                self.path.mkdir()
                with self.metadata.open("x", encoding="utf-8") as handle:
                    json.dump({"pid": os.getpid(), "token": self.token}, handle)
                    handle.flush()
                    os.fsync(handle.fileno())
                if not self.owned():
                    raise RuntimeError("lock ownership token mismatch after acquisition")
                return
            except FileExistsError:
                self._try_recover()
            if time.monotonic() >= deadline:
                raise TimeoutError(f"history lock wait exceeded {self.timeout:.1f}s")
            time.sleep(self.poll)

    def owned(self) -> bool:
        owner = self._read_owner()
        return bool(owner and owner["pid"] == os.getpid() and owner["token"] == self.token)

    def release(self) -> bool:
        if not self.owned():
            return False
        try:
            self.metadata.unlink()
            self.path.rmdir()
            return True
        except OSError:
            return False

    def __enter__(self) -> "HistoryLock":
        self.acquire()
        return self

    def __exit__(self, *_args: object) -> None:
        self.release()


def atomic_write_history(
    path: str | Path, records: list[dict], before_replace_delay: float = 0
) -> None:
    path = Path(path)
    temp = Path(f"{path}.tmp.{os.getpid()}")
    with temp.open("w", encoding="utf-8") as handle:
        for record in sorted(records, key=record_key):
            handle.write(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n")
        handle.flush()
        os.fsync(handle.fileno())
    if before_replace_delay:
        time.sleep(before_replace_delay)
    os.replace(temp, path)


def update_history(
    path: str | Path,
    tonight_records: list[dict],
    lock_timeout: float = 60,
    stale_after: float = 600,
    before_replace_delay: float = 0,
) -> None:
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    lock = HistoryLock(
        f"{path}.lock.d", timeout=lock_timeout, stale_after=stale_after
    )
    with lock:
        if not lock.owned():
            raise RuntimeError("history lock ownership lost before update")
        for stray in path.parent.glob(f"{path.name}.tmp.*"):
            try:
                stray.unlink()
            except OSError:
                pass
        existing, _ = load_history(path) if path.exists() else ([], 0)
        keys = {record_key(record) for record in tonight_records}
        merged = [record for record in existing if record_key(record) not in keys]
        merged.extend(tonight_records)
        if not lock.owned():
            raise RuntimeError("history lock ownership lost before write")
        atomic_write_history(path, merged, before_replace_delay)


def bootstrap_history(path: str | Path, pattern: str) -> None:
    records = []
    for directory in sorted(glob.glob(pattern)):
        if os.path.isdir(directory) and glob.glob(os.path.join(directory, "*.json")):
            records.extend(records_for_dir(directory))
    update_history(path, records)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tonight")
    parser.add_argument("--history", default="~/.ailang/state/nightly-eval-history.jsonl")
    parser.add_argument("--model")
    parser.add_argument("--arm", default="rag_on")
    parser.add_argument("--window-nights", type=int, default=5)
    parser.add_argument("--min-nights", type=int, default=2)
    parser.add_argument("--min-trials", type=int, default=4)
    parser.add_argument("--escalate-after", type=int, default=3)
    parser.add_argument("--update-history", action="store_true")
    parser.add_argument("--bootstrap", action="store_true")
    parser.add_argument(
        "--bootstrap-glob", default="/tmp/nightly_eval_*_rag_on/agent"
    )
    parser.add_argument("--legacy", action="store_true")
    parser.add_argument("--previous")
    parser.add_argument("--lock-timeout", type=float, default=60)
    parser.add_argument("--stale-after", type=float, default=600)
    parser.add_argument("--before-replace-delay", type=float, default=0)
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    logging.basicConfig(level=logging.WARNING, format="%(levelname)s: %(message)s")
    history = Path(os.path.expanduser(args.history))
    history.parent.mkdir(parents=True, exist_ok=True)
    try:
        if args.bootstrap:
            bootstrap_history(history, args.bootstrap_glob)
            records, skipped = load_history(history)
            print(f"HEALTH\t{history_health(history, records, skipped)}")
            return 0
        if not args.tonight:
            raise ValueError("--tonight is required unless --bootstrap is used")
        if args.legacy:
            print("\n".join(legacy_classify(args.tonight, args.previous)))
            return 0
        if not os.path.isdir(args.tonight):
            raise ValueError(f"tonight directory unreadable: {args.tonight}")

        tonight_date = parse_date(args.tonight)
        model = args.model or infer_model(args.tonight)
        failures = persistent_failures(parse_results_dir(args.tonight))
        unavailable = None
        try:
            records, skipped = load_history(history)
        except OSError as error:
            records, skipped = [], 0
            unavailable = str(error)
        if skipped and not records:
            unavailable = f"corrupt history: {skipped} unparseable line(s), no valid records"

        if unavailable:
            warning = (
                f"⚠ history unavailable ({unavailable}) — regression detection "
                "DEGRADED tonight"
            )
            print(f"HEALTH\t{warning}")
            verdicts = [
                Verdict("INSUFFICIENT-HISTORY", bench, cats, 0, 0, 0, 1)
                for bench, cats in sorted(failures.items())
            ]
        else:
            print(f"HEALTH\t{history_health(history, records, skipped)}")
            verdicts = [
                classify_bench(
                    bench,
                    cats,
                    records,
                    model,
                    args.arm,
                    tonight_date,
                    args.window_nights,
                    args.min_nights,
                    args.min_trials,
                    args.escalate_after,
                )
                for bench, cats in sorted(failures.items())
            ]
        for verdict in verdicts:
            print(verdict.tsv())

        # Absence/unreadability never auto-heals. Only explicit bootstrap creates
        # a missing history file.
        if args.update_history and not unavailable:
            tonight_records = records_for_dir(args.tonight, model, args.arm)
            by_bench = {verdict.bench: verdict for verdict in verdicts}
            for record in tonight_records:
                verdict = by_bench.get(record["bench"])
                record["class"] = verdict.label.lower() if verdict else ""
            update_history(
                history,
                tonight_records,
                args.lock_timeout,
                args.stale_after,
                args.before_replace_delay,
            )
        return 0
    except (OSError, RuntimeError, TimeoutError, ValueError) as error:
        print(f"nightly-classify: ERROR: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
