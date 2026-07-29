#!/usr/bin/env python3
"""Contract tests for tools/nightly_classify.py (stdlib only)."""

from __future__ import annotations

import contextlib
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

TOOLS = Path(__file__).resolve().parent
ROOT = TOOLS.parent
sys.path.insert(0, str(TOOLS))

import nightly_classify as nc  # noqa: E402

MODEL = "opencode-qwen3-5-35b-a3b-mxfp8"
ARM = "rag_on"
FIXTURE = TOOLS / "testdata/nightly_classify/replay_2026-07.jsonl"


def rec(
    date: str,
    bench: str = "bench",
    passes: int = 2,
    trials: int = 2,
    cls: str = "",
) -> dict:
    return {
        "date": date,
        "bench": bench,
        "model": MODEL,
        "arm": ARM,
        "trials": trials,
        "passes": passes,
        "cats": [] if passes else ["compile_error"],
        "class": cls,
    }


def verdict(records: list[dict], date: str, bench: str = "bench", **kwargs) -> nc.Verdict:
    return nc.classify_bench(
        bench, ["compile_error"], records, MODEL, ARM, date, **kwargs
    )


def write_trial(
    directory: Path,
    bench: str,
    trial: int,
    passed: bool,
    cat: str = "compile_error",
    timestamp: int = 100,
) -> None:
    directory.mkdir(parents=True, exist_ok=True)
    suffix = "" if trial == 1 else f"_trial{trial}"
    name = f"{bench}{suffix}_ailang_{MODEL}_{timestamp}.json"
    data = {
        "compile_ok": passed,
        "runtime_ok": passed,
        "stdout_ok": passed,
        "error_category": "none" if passed else cat,
    }
    (directory / name).write_text(json.dumps(data), encoding="utf-8")


def result_dir(root: Path, raw_date: str) -> Path:
    return root / f"nightly_eval_{raw_date}_rag_on" / "agent"


def run_cli(*args: str, timeout: float = 10) -> subprocess.CompletedProcess:
    return subprocess.run(
        [sys.executable, str(TOOLS / "nightly_classify.py"), *map(str, args)],
        text=True,
        capture_output=True,
        timeout=timeout,
        check=False,
    )


class LegacyClassifierTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        root = Path(self.temp.name)
        self.tonight = result_dir(root, "20260728")
        self.previous = result_dir(root, "20260727")

    def tearDown(self) -> None:
        self.temp.cleanup()

    def test_pass_pass_not_persistent(self):
        write_trial(self.tonight, "x", 1, True)
        write_trial(self.tonight, "x", 2, True)
        self.assertEqual(nc.legacy_classify(self.tonight, self.previous), [])

    def test_pass_fail_not_persistent(self):
        write_trial(self.tonight, "x", 1, True)
        write_trial(self.tonight, "x", 2, False)
        self.assertEqual(nc.legacy_classify(self.tonight, self.previous), [])

    def test_solid_to_allfail_is_regression(self):
        for trial in (1, 2):
            write_trial(self.tonight, "x", trial, False)
            write_trial(self.previous, "x", trial, True)
        self.assertEqual(
            nc.legacy_classify(self.tonight, self.previous),
            ["REGRESSION\tx\t[compile_error]"],
        )

    def test_all_infra_cats_only_is_ignored(self):
        write_trial(self.tonight, "x", 1, False, "api_error")
        write_trial(self.tonight, "x", 2, False, "timeout")
        self.assertEqual(nc.legacy_classify(self.tonight, self.previous), [])

    def test_single_trial_tonight_not_persistent(self):
        write_trial(self.tonight, "x", 1, False)
        self.assertEqual(nc.legacy_classify(self.tonight, self.previous), [])

    def test_missing_prior_dir_yields_gap(self):
        for trial in (1, 2):
            write_trial(self.tonight, "x", trial, False)
        self.assertEqual(nc.legacy_classify(self.tonight, None), ["GAP\tx\t[compile_error]"])

    def test_empty_prior_dir_yields_gap(self):
        self.previous.mkdir(parents=True)
        for trial in (1, 2):
            write_trial(self.tonight, "x", trial, False)
        self.assertEqual(
            nc.legacy_classify(self.tonight, self.previous),
            ["GAP\tx\t[compile_error]"],
        )

    def test_trial_grouping(self):
        write_trial(self.tonight, "json_parse", 1, False)
        write_trial(self.tonight, "json_parse", 2, False)
        lines = nc.legacy_classify(self.tonight, self.previous)
        self.assertEqual(lines, ["GAP\tjson_parse\t[compile_error]"])
        self.assertNotIn("json_parse_trial2", "\n".join(lines))

    def test_newest_duplicate_trial_slot_wins(self):
        write_trial(self.tonight, "x", 1, True, timestamp=100)
        write_trial(self.tonight, "x", 1, False, timestamp=200)
        write_trial(self.tonight, "x", 2, False, timestamp=200)
        self.assertEqual(nc.legacy_classify(self.tonight, None), ["GAP\tx\t[compile_error]"])


class TaintTests(unittest.TestCase):
    @staticmethod
    def _failures(*cats: str) -> dict[str, list[tuple[bool, str]]]:
        return {"bench": [(False, cat) for cat in cats]}

    @staticmethod
    def _record_trials(record: dict) -> list[tuple[bool, str]]:
        passes = int(record["passes"])
        trials = int(record["trials"])
        failed = trials - passes
        cats = list(record["cats"])
        if failed and not cats:
            cats = ["—"]
        failure_cats = (cats * failed)[:failed] if len(cats) == 1 else cats[:failed]
        return [(True, "none")] * passes + [
            (False, cat) for cat in failure_cats
        ]

    def test_Taint_mixed_infra_and_real_category_is_suppressed(self):
        got = nc.persistent_failures(
            self._failures("api_error", "compile_error")
        )
        self.assertEqual(got, {})

    def test_Taint_clean_failure_still_files(self):
        got = nc.persistent_failures(
            self._failures("compile_error", "compile_error")
        )
        self.assertEqual(got, {"bench": ["compile_error"]})

    def test_Taint_timeout_and_executor_error_also_taint(self):
        for category in sorted(nc.INFRA_CATEGORIES):
            with self.subTest(category=category):
                got = nc.persistent_failures(
                    self._failures(category, "logic_error")
                )
                self.assertEqual(got, {})

    def test_Taint_replay_costs_are_pinned(self):
        records, skipped = nc.load_history(FIXTURE)
        self.assertEqual(skipped, 0)
        expected = {
            "2026-07-24": (6, 1),
            "2026-07-25": (7, 1),
            "2026-07-26": (4, 0),
            "2026-07-27": (5, 2),
            "2026-07-28": (8, 0),
        }
        totals = [0, 0]
        for date, want in expected.items():
            results = {
                record["bench"]: self._record_trials(record)
                for record in records
                if record["date"] == date
            }
            old_persistent = sum(
                len(trials) >= 2
                and all(not passed for passed, _ in trials)
                and bool(
                    {cat for _, cat in trials} - nc.INFRA_CATEGORIES
                )
                for trials in results.values()
            )
            clean = len(nc.persistent_failures(results))
            got = (old_persistent, old_persistent - clean)
            self.assertEqual(got, want, date)
            totals[0] += old_persistent
            totals[1] += old_persistent - clean
        self.assertEqual(tuple(totals), (30, 4))

    def test_Taint_csv_to_json_escalation_moves_to_0728_not_lost(self):
        records, skipped = nc.load_history(FIXTURE)
        self.assertEqual(skipped, 0)
        records = [dict(record) for record in records if record["date"] <= "2026-07-27"]
        row_0727 = next(
            record
            for record in records
            if record["bench"] == "csv_to_json_converter"
            and record["date"] == "2026-07-27"
        )
        failures = nc.persistent_failures(
            {"csv_to_json_converter": self._record_trials(row_0727)}
        )
        self.assertNotIn("csv_to_json_converter", failures)
        row_0727["class"] = ""
        got = verdict(records, "2026-07-28", "csv_to_json_converter")
        self.assertEqual(got.label, "REGRESSION")
        self.assertEqual(got.consecutive, 4)


class ValidityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.records, cls.skipped = nc.load_history(FIXTURE)

    @staticmethod
    def _results_for(records: list[dict], date: str) -> dict:
        return {
            record["bench"]: TaintTests._record_trials(record)
            for record in records
            if record["date"] == date
        }

    def test_Validity_six_night_table(self):
        self.assertEqual(self.skipped, 0)
        self.assertEqual(
            nc.build_parser().get_default("invalid_infra_fraction"), 0.30
        )
        expected = [
            ("2026-07-24", 2, 42, True),
            ("2026-07-25", 1, 42, True),
            ("2026-07-26", 2, 42, True),
            ("2026-07-27", 2, 42, True),
            ("2026-07-28", 2, 42, True),
            ("2026-07-29", 42, 42, False),
        ]
        for date, tainted, total, valid in expected:
            with self.subTest(date=date):
                results = self._results_for(self.records, date)
                if date == "2026-07-29":
                    results = {
                        f"bench-{index:02d}": (
                            [(True, "none"), (False, "api_error")]
                            if index < 14
                            else [(False, "api_error"), (False, "api_error")]
                        )
                        for index in range(42)
                    }
                got = nc.run_validity(
                    results, 0.30
                )
                self.assertEqual(got, (valid, "" if valid else "infra_outage", tainted, total))

    def test_Validity_boundary_is_inclusive(self):
        def results(tainted: int, total: int) -> dict:
            return {
                f"bench-{index}": [
                    (False, "api_error" if index < tainted else "compile_error")
                ]
                for index in range(total)
            }

        self.assertEqual(
            nc.run_validity(results(3000, 10000), 0.30)[:2],
            (False, "infra_outage"),
        )
        self.assertEqual(
            nc.run_validity(results(2999, 10000), 0.30)[:2],
            (True, ""),
        )
        # A catastrophic but clean subject regression must never be hidden.
        self.assertEqual(
            nc.run_validity(results(0, 42), 0.30)[:2],
            (True, ""),
        )

    def test_Validity_zero_benches_is_zero_files_not_zerodivision(self):
        self.assertEqual(nc.run_validity({}, 0.30), (False, "zero_files", 0, 0))
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            tonight = result_dir(root, "20260730")
            tonight.mkdir(parents=True)
            history = root / "history.jsonl"
            shutil.copyfile(FIXTURE, history)
            proc = run_cli("--tonight", tonight, "--history", history)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertIn("INVALID\tzero_files\t0/0\t", proc.stdout)
            self.assertNotIn("Traceback", proc.stderr)

    def test_Validity_invalid_cli_suppresses_pre_gate_regression(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            tonight = result_dir(root, "20260730")
            history = root / "history.jsonl"
            shutil.copyfile(FIXTURE, history)
            # Thirteen of 42 benches are tainted (30.95%). The other 29 fail
            # cleanly, so the pre-gate classifier has real verdicts to emit.
            for index in range(42):
                bench = f"outage-{index:02d}"
                category = "api_error" if index < 13 else "compile_error"
                write_trial(tonight, bench, 1, False, category)
                write_trial(tonight, bench, 2, False, category)
            # Give one clean-failing bench a solid two-night window.
            seeded = [
                rec("2026-07-28", bench="outage-41"),
                rec("2026-07-29", bench="outage-41"),
            ]
            with history.open("a", encoding="utf-8") as handle:
                for record in seeded:
                    handle.write(json.dumps(record) + "\n")

            records, _ = nc.load_history(history)
            results = nc.parse_results_dir(tonight)
            failures = nc.persistent_failures(results)
            pre_gate = [
                nc.classify_bench(
                    bench, cats, records, MODEL, ARM, "2026-07-30"
                ).tsv()
                for bench, cats in sorted(failures.items())
            ]
            self.assertGreaterEqual(
                sum(line.startswith("REGRESSION\t") for line in pre_gate), 1
            )

            proc = run_cli("--tonight", tonight, "--history", history)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            invalid = [
                line for line in proc.stdout.splitlines()
                if line.startswith("INVALID\t")
            ]
            self.assertEqual(len(invalid), 1, proc.stdout)
            for label in (
                "REGRESSION\t",
                "SUSPECTED-FLAKE\t",
                "GAP\t",
                "INSUFFICIENT-HISTORY\t",
            ):
                self.assertFalse(
                    any(line.startswith(label) for line in proc.stdout.splitlines()),
                    proc.stdout,
                )


class RuleTests(unittest.TestCase):
    def test_Rule_solid_window_regresses(self):
        records = [rec("2026-07-24"), rec("2026-07-25")]
        self.assertEqual(verdict(records, "2026-07-26").label, "REGRESSION")

    def test_Rule_mixed_window_is_suspected_flake(self):
        records = [rec("2026-07-24"), rec("2026-07-25", passes=1)]
        self.assertEqual(verdict(records, "2026-07-26").label, "SUSPECTED-FLAKE")

    def test_Rule_all_fail_window_is_gap(self):
        records = [rec("2026-07-24", passes=0), rec("2026-07-25", passes=0)]
        self.assertEqual(verdict(records, "2026-07-26").label, "GAP")

    def test_Rule_three_trials_is_insufficient(self):
        records = [
            rec("2026-07-24", passes=1, trials=1),
            rec("2026-07-25", passes=2),
        ]
        self.assertEqual(verdict(records, "2026-07-26").label, "INSUFFICIENT-HISTORY")

    def test_Rule_one_night_is_insufficient(self):
        records = [rec("2026-07-24", passes=4, trials=4)]
        self.assertEqual(verdict(records, "2026-07-26").label, "INSUFFICIENT-HISTORY")

    def test_single_trial_prior_night_cannot_certify_solid(self):
        records = [rec("2026-07-24", passes=1, trials=1)]
        new = verdict(records, "2026-07-25")
        self.assertEqual(new.label, "INSUFFICIENT-HISTORY")
        # Negative control: the old one-night rule alarms on this same evidence.
        old_rule = "REGRESSION" if records and all(r["passes"] == r["trials"] for r in records) else "GAP"
        self.assertEqual(old_rule, "REGRESSION")

    def test_never_passed_bench_does_not_escalate(self):
        records = [
            rec("2026-07-24", passes=0),
            rec("2026-07-25", passes=0),
            rec("2026-07-26", passes=0),
        ]
        got = verdict(records, "2026-07-27")
        self.assertEqual(got.label, "GAP")
        self.assertGreaterEqual(got.consecutive, 3)

    def test_new_benchmark_timeline(self):
        records = [rec("2026-07-20")]
        n2 = verdict(records, "2026-07-21")
        self.assertEqual(n2.label, "INSUFFICIENT-HISTORY")
        self.assertEqual(n2.bench, "bench")
        self.assertEqual(n2.consecutive, 1)
        records.append(rec("2026-07-21", passes=0, cls=n2.label.lower()))
        n3 = verdict(records, "2026-07-22")
        self.assertEqual(n3.label, "SUSPECTED-FLAKE")
        records.append(rec("2026-07-22", passes=0, cls=n3.label.lower()))
        n4 = verdict(records, "2026-07-23")
        self.assertEqual(n4.label, "REGRESSION")
        self.assertEqual(n4.escalated_from, "SUSPECTED-FLAKE")


class EscalationTests(unittest.TestCase):
    def test_Escalation_fires_exactly_once(self):
        records = [rec("2026-07-19"), rec("2026-07-20", passes=1)]
        labels = []
        for day in range(21, 26):
            date = f"2026-07-{day:02d}"
            got = verdict(records, date)
            labels.append(got.label)
            records.append(rec(date, passes=0, cls=got.label.lower()))
        self.assertEqual(labels.count("REGRESSION"), 1)
        self.assertEqual(labels[2], "REGRESSION")

    def test_Escalation_missed_third_night_fires_on_fourth(self):
        records = [
            rec("2026-07-19"),
            rec("2026-07-20", passes=1),
            rec("2026-07-21", passes=0, cls="suspected-flake"),
            rec("2026-07-22", passes=0, cls="suspected-flake"),
            # 07-23 all-failed, but classifier/send was missed: no class record.
            rec("2026-07-23", passes=0, cls=""),
        ]
        got = verdict(records, "2026-07-24")
        self.assertEqual(got.label, "REGRESSION")
        self.assertGreater(got.consecutive, 3)  # fails under literal consec == K


class HistoryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.history = self.root / "state/history.jsonl"

    def tearDown(self):
        self.temp.cleanup()

    def test_History_same_date_rerun_replaces_records(self):
        nc.update_history(self.history, [rec("2026-07-24", passes=2)])
        nc.update_history(self.history, [rec("2026-07-24", passes=0)])
        records, _ = nc.load_history(self.history)
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]["passes"], 0)

    def test_History_rerun_yields_byte_identical_verdicts(self):
        nc.update_history(self.history, [rec("2026-07-24"), rec("2026-07-25")])
        records, _ = nc.load_history(self.history)
        first = verdict(records, "2026-07-26").tsv()
        nc.update_history(self.history, [rec("2026-07-25")])
        records, _ = nc.load_history(self.history)
        self.assertEqual(first, verdict(records, "2026-07-26").tsv())

    def test_History_preseeded_duplicate_keys_last_wins_and_compacts(self):
        self.history.parent.mkdir(parents=True)
        self.history.write_text(
            json.dumps(rec("2026-07-24", passes=2)) + "\n"
            + json.dumps(rec("2026-07-24", passes=0)) + "\n",
            encoding="utf-8",
        )
        records, _ = nc.load_history(self.history)
        self.assertEqual(records[0]["passes"], 0)
        nc.update_history(self.history, [rec("2026-07-25")])
        self.assertEqual(len(self.history.read_text().splitlines()), 2)

    def test_History_tonight_records_do_not_enter_own_window(self):
        records = [rec("2026-07-24"), rec("2026-07-25"), rec("2026-07-26", passes=0)]
        got = verdict(records, "2026-07-26")
        self.assertEqual((got.passes, got.trials, got.nights), (4, 4, 2))

    def test_History_no_auto_heal_and_fresh_state_dir(self):
        tonight = result_dir(self.root, "20260728")
        write_trial(tonight, "x", 1, False)
        write_trial(tonight, "x", 2, False)
        proc = run_cli("--tonight", tonight, "--history", self.history, "--update-history")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("history unavailable", proc.stdout)
        self.assertNotIn("REGRESSION\t", proc.stdout)
        self.assertFalse(self.history.exists())
        self.assertTrue(self.history.parent.is_dir())
        self.assertNotIn("Traceback", proc.stderr)

    def test_History_explicit_bootstrap_is_idempotent(self):
        for raw in ("20260724", "20260725"):
            directory = result_dir(self.root, raw)
            write_trial(directory, "x", 1, raw == "20260724")
            write_trial(directory, "x", 2, raw == "20260724")
        pattern = str(self.root / "nightly_eval_*_rag_on/agent")
        first = run_cli("--bootstrap", "--history", self.history, "--bootstrap-glob", pattern)
        self.assertEqual(first.returncode, 0, first.stderr)
        before = self.history.read_bytes()
        second = run_cli("--bootstrap", "--history", self.history, "--bootstrap-glob", pattern)
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertEqual(before, self.history.read_bytes())
        self.assertGreaterEqual(len(before.splitlines()), 2)

    def test_History_corrupt_lines_counted_then_compacted(self):
        self.history.parent.mkdir(parents=True)
        valid = json.dumps(rec("2026-07-24"))
        self.history.write_text("{not json\n{\"truncated\":\n" + valid + "\n", encoding="utf-8")
        records, skipped = nc.load_history(self.history)
        self.assertEqual(skipped, 2)
        self.assertIn("2 skipped lines", nc.history_health(self.history, records, skipped))
        before = verdict(records, "2026-07-25").tsv()
        self.assertEqual(before, verdict([rec("2026-07-24")], "2026-07-25").tsv())
        nc.update_history(self.history, [rec("2026-07-25")])
        self.assertNotIn("{not json", self.history.read_text())

    def test_History_fully_corrupt_file_degrades_without_healing(self):
        tonight = result_dir(self.root, "20260728")
        write_trial(tonight, "x", 1, False)
        write_trial(tonight, "x", 2, False)
        self.history.parent.mkdir(parents=True)
        self.history.write_text("{truncated\n", encoding="utf-8")
        before = self.history.read_bytes()
        proc = run_cli("--tonight", tonight, "--history", self.history, "--update-history")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("corrupt history", proc.stdout)
        self.assertNotIn("REGRESSION\t", proc.stdout)
        self.assertEqual(before, self.history.read_bytes())


class LockTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.history = self.root / "history.jsonl"
        self.history.write_text(json.dumps(rec("2026-07-20")) + "\n", encoding="utf-8")
        self.before = self.history.read_bytes()

    def tearDown(self):
        self.temp.cleanup()

    def _live_holder(self, backdate: bool = False):
        code = """
import json, os, pathlib, sys, time
p=pathlib.Path(sys.argv[1]); p.mkdir()
(p/'owner.json').write_text(json.dumps({'pid':os.getpid(),'token':'holder'}))
print('READY', flush=True)
time.sleep(30)
"""
        proc = subprocess.Popen(
            [sys.executable, "-c", code, f"{self.history}.lock.d"],
            stdout=subprocess.PIPE,
            text=True,
        )
        self.assertEqual(proc.stdout.readline().strip(), "READY")
        if backdate:
            old = time.time() - 700
            os.utime(f"{self.history}.lock.d", (old, old))
        def cleanup():
            if proc.poll() is None:
                proc.kill()
            proc.wait()
            if proc.stdout:
                proc.stdout.close()
        self.addCleanup(cleanup)
        return proc

    def _waiter(self, timeout: float = 0.35, stale: float = 600):
        start = time.monotonic()
        proc = run_cli(
            "--tonight", result_dir(self.root, "20260728"),
            "--history", self.history,
            "--update-history",
            "--lock-timeout", str(timeout),
            "--stale-after", str(stale),
        )
        return proc, time.monotonic() - start

    def test_Lock_held_lock_waiter_waits_bounded_then_exits_nonzero(self):
        result_dir(self.root, "20260728").mkdir(parents=True)
        self._live_holder()
        proc, elapsed = self._waiter()
        self.assertNotEqual(proc.returncode, 0)
        self.assertGreaterEqual(elapsed, 0.30)
        self.assertLessEqual(elapsed, 5.35)
        self.assertEqual(self.before, self.history.read_bytes())

    def test_Lock_stale_but_alive_holder_is_not_stolen_from(self):
        result_dir(self.root, "20260728").mkdir(parents=True)
        holder = self._live_holder(backdate=True)
        proc, elapsed = self._waiter(stale=0.1)
        self.assertNotEqual(proc.returncode, 0)
        self.assertGreaterEqual(elapsed, 0.30)
        self.assertIsNone(holder.poll())
        self.assertTrue(Path(f"{self.history}.lock.d").exists())
        self.assertEqual(self.before, self.history.read_bytes())

    def test_Lock_old_holder_resuming_after_replacement_detects_token_mismatch_and_does_not_write(self):
        lock = nc.HistoryLock(f"{self.history}.lock.d", timeout=0.1)
        lock.acquire()
        shutil.rmtree(lock.path)
        lock.path.mkdir()
        lock.metadata.write_text(json.dumps({"pid": os.getpid(), "token": "replacement"}))
        self.assertFalse(lock.owned())
        self.assertFalse(lock.release())
        self.assertEqual(self.before, self.history.read_bytes())

    def test_Lock_ownership_checked_release_cannot_delete_another_process_lock(self):
        lock = nc.HistoryLock(f"{self.history}.lock.d", timeout=0.1)
        lock.acquire()
        lock.metadata.write_text(json.dumps({"pid": os.getpid(), "token": "other"}))
        self.assertFalse(lock.release())
        self.assertTrue(lock.path.exists())

    def test_Lock_unreadable_lock_metadata_conservative_recovery(self):
        lock_dir = Path(f"{self.history}.lock.d")
        lock_dir.mkdir()
        (lock_dir / "owner.json").write_text("{")
        lock = nc.HistoryLock(lock_dir, timeout=0.2, stale_after=60, poll=0.03)
        with self.assertRaises(TimeoutError):
            lock.acquire()
        self.assertTrue(lock_dir.exists())
        old = time.time() - 70
        os.utime(lock_dir, (old, old))
        recovered = nc.HistoryLock(lock_dir, timeout=0.5, stale_after=60, poll=0.03)
        recovered.acquire()
        self.assertTrue(recovered.owned())
        recovered.release()


class AtomicTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.history = self.root / "history.jsonl"
        self.history.write_text(json.dumps(rec("2026-07-20")) + "\n")

    def tearDown(self):
        self.temp.cleanup()

    def test_Atomic_kill_between_tmpwrite_and_rename_leaves_prior_history_byte_intact(self):
        before = hashlib.sha256(self.history.read_bytes()).hexdigest()
        code = (
            "import sys;sys.path.insert(0,sys.argv[1]);import nightly_classify as n;"
            "n.atomic_write_history(sys.argv[2],["
            + repr(rec("2026-07-21"))
            + "],10)"
        )
        proc = subprocess.Popen([sys.executable, "-c", code, str(TOOLS), str(self.history)])
        temp = Path(f"{self.history}.tmp.{proc.pid}")
        deadline = time.monotonic() + 3
        while not temp.exists() and time.monotonic() < deadline:
            time.sleep(0.02)
        self.assertTrue(temp.exists())
        proc.kill()
        proc.wait()
        after = hashlib.sha256(self.history.read_bytes()).hexdigest()
        self.assertEqual(before, after)

    def test_Atomic_stray_temp_removed_on_next_locked_update(self):
        stray = Path(f"{self.history}.tmp.999999")
        stray.write_text("partial")
        nc.update_history(self.history, [rec("2026-07-21")])
        self.assertFalse(stray.exists())


class ReplayTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.records, cls.skipped = nc.load_history(FIXTURE)

    def replay(self, bench: str, date: str, include_v9: bool = True):
        records = self.records
        if not include_v9:
            records = [r for r in records if not r.get("provenance")]
        return verdict(records, date, bench)

    def test_Replay_suppresses_real_false_alarms(self):
        self.assertEqual(self.replay("json_parse", "2026-07-25").label, "SUSPECTED-FLAKE")
        live = self.replay("json_parse", "2026-07-25", include_v9=False)
        self.assertEqual(live.label, "INSUFFICIENT-HISTORY")
        jp = self.replay("json_parse", "2026-07-27", False)
        self.assertEqual((jp.label, jp.passes, jp.trials, jp.nights), ("SUSPECTED-FLAKE", 4, 6, 3))
        bst = self.replay("contract_bst_validate", "2026-07-28", False)
        self.assertEqual((bst.label, bst.passes, bst.trials, bst.nights), ("SUSPECTED-FLAKE", 4, 8, 4))
        lc = self.replay("list_comprehension", "2026-07-28", False)
        self.assertEqual((lc.label, lc.passes, lc.trials, lc.nights), ("SUSPECTED-FLAKE", 7, 8, 4))

    def test_Replay_real_genuine_regression_pages_same_night(self):
        got = self.replay("higher_order_functions", "2026-07-26", False)
        self.assertEqual((got.label, got.passes, got.trials), ("REGRESSION", 4, 4))
        synthetic = [rec(f"2026-07-{day:02d}") for day in range(20, 25)]
        self.assertEqual(verdict(synthetic, "2026-07-25").label, "REGRESSION")

    def test_replay_csv_to_json_escalates_once(self):
        records = [dict(r) for r in self.records if r["date"] <= "2026-07-26"]
        n3 = verdict(records, "2026-07-27", "csv_to_json_converter")
        self.assertEqual(n3.label, "REGRESSION")
        self.assertEqual(n3.consecutive, 3)
        for record in self.records:
            if record["bench"] == "csv_to_json_converter" and record["date"] == "2026-07-27":
                current = dict(record)
                current["class"] = "regression"
                records.append(current)
        n4 = verdict(records, "2026-07-28", "csv_to_json_converter")
        self.assertNotEqual(n4.label, "REGRESSION")

    def test_Replay_aggregate_is_five_today_vs_two_guarded(self):
        filed = [
            ("json_parse", "2026-07-25"),
            ("higher_order_functions", "2026-07-26"),
            ("json_parse", "2026-07-27"),
            ("contract_bst_validate", "2026-07-28"),
            ("list_comprehension", "2026-07-28"),
        ]
        labels = [self.replay(bench, date, False).label for bench, date in filed]
        regressions = labels.count("REGRESSION") + 1  # csv_to_json K=3 escalation
        suppressed = len(filed) - labels.count("REGRESSION")
        print(f"REPLAY SUMMARY: filed=5 guarded_regressions={regressions} suppressed={suppressed}")
        self.assertEqual(regressions, 2)
        self.assertEqual(suppressed, 4)


class RoutingContractTests(unittest.TestCase):
    def test_Routing_invalid_run_files_nothing(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            calls = root / "calls.log"
            stub = bin_dir / "ailang"
            stub.write_text(
                "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \"$CALLS\"\n",
                encoding="utf-8",
            )
            stub.chmod(0o755)
            script = (TOOLS / "launchd/nightly-eval.sh").read_text(encoding="utf-8")
            route = script[script.index('HEALTH=$(echo "$CLASSIFIED"') :]
            harness = root / "route.sh"
            harness.write_text(
                "set -euo pipefail\n"
                "log(){ :; }\n"
                "DATE=2026-07-29\nMODEL=model\nBENCH_TIERS=smoke,core\n"
                "RESULTS_DIR=/tmp/results\nPASS=14/84\nRATE=16%\n"
                "BUILD_VERSION=vtest\nSHORT=abc123\n"
                + route,
                encoding="utf-8",
            )
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{bin_dir}:{env['PATH']}",
                    "CALLS": str(calls),
                    "CLASSIFIED": (
                        "HEALTH\thistory: fixture\n"
                        "INVALID\tinfra_outage\t42/42\t0.167\t0.643"
                    ),
                }
            )
            proc = subprocess.run(
                ["bash", str(harness)],
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(proc.returncode, 0, proc.stderr)
            logged = calls.read_text(encoding="utf-8")
            self.assertEqual(logged.count("messages send"), 1, logged)
            self.assertEqual(logged.count("--type note"), 1, logged)
            for forbidden in ("--type bug", "--type feature", "--github", "public-feedback"):
                self.assertNotIn(forbidden, logged)

    def test_routing_smoke_aggregates_suppressed_labels(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            calls = root / "calls.log"
            stub = bin_dir / "ailang"
            stub.write_text(
                "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \"$CALLS\"\n",
                encoding="utf-8",
            )
            stub.chmod(0o755)
            script = (TOOLS / "launchd/nightly-eval.sh").read_text(encoding="utf-8")
            route = script[script.index('HEALTH=$(echo "$CLASSIFIED"') :]
            classified = "\n".join(
                [
                    "HEALTH\thistory: fixture | 5 benchmarks, 3 nights, newest 2026-07-27, 0 skipped lines",
                    "REGRESSION\treg\t[compile_error]\t4/4\t2\t1\t-",
                    "SUSPECTED-FLAKE\tflake1\t[compile_error]\t3/4\t2\t1\t-",
                    "SUSPECTED-FLAKE\tflake2\t[logic_error]\t2/4\t2\t2\t-",
                    "GAP\tgap\t[logic_error]\t0/4\t2\t3\t-",
                    "INSUFFICIENT-HISTORY\tnew\t[compile_error]\t2/2\t1\t1\t-",
                ]
            )
            harness = root / "route.sh"
            harness.write_text(
                "set -euo pipefail\n"
                "log(){ :; }\n"
                "DATE=2026-07-28\nMODEL=model\nBENCH_TIERS=smoke,core\n"
                "RESULTS_DIR=/tmp/results\nPASS=1/10\nRATE=10%\n"
                "BUILD_VERSION=vtest\nSHORT=abc123\n"
                + route,
                encoding="utf-8",
            )
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{bin_dir}:{env['PATH']}",
                    "CALLS": str(calls),
                    "CLASSIFIED": classified,
                }
            )
            proc = subprocess.run(
                ["bash", str(harness)], env=env, text=True, capture_output=True, check=False
            )
            self.assertEqual(proc.returncode, 0, proc.stderr)
            logged = calls.read_text(encoding="utf-8")
            self.assertEqual(logged.count("--type bug"), 1, logged)
            self.assertEqual(logged.count("--type note"), 2)
            self.assertEqual(logged.count("messages send public-feedback"), 1)
            self.assertEqual(logged.count("suspected-flake(s)"), 1)
            self.assertIn("history: fixture", logged)
            self.assertIn("insufficient history: new", logged)

    def test_send_titles_embed_date(self):
        """Exactly-once is conditional: archived/deleted inbox rows are excluded."""
        titles = [
            "Nightly regression: bench (2026-07-28)",
            "Nightly eval: 2 suspected-flake(s) (2026-07-28)",
            "Nightly eval: 1 non-regression failure(s) (2026-07-28)",
            "Nightly eval: 70/84 (2026-07-28)",
        ]
        for title in titles:
            self.assertRegex(title, r".*\(\d{4}-\d{2}-\d{2}\)$")


class PassTextResult(unittest.TextTestResult):
    def addSuccess(self, test):
        super().addSuccess(test)
        self.stream.writeln(f"--- PASS: {test._testMethodName}")


class PassTextRunner(unittest.TextTestRunner):
    resultclass = PassTextResult


if __name__ == "__main__":
    unittest.main(testRunner=PassTextRunner, verbosity=2)
