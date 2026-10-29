# Iteration 34 executor report

Built `skill_resolution_spike.sh` (400 lines), covering M1 Claude, M2 Codex and M3 pi. It implements `--only`, `--reps`, `--fence`, `--dry-run` and `SPIKE_TEST_ROOT`, fresh skill/resource tokens, ownership-ledger cleanup, parent audits, exact model argv, deadline/PGID checks, per-call evidence, final-answer-only scoring, and before/after controls. No model calls or git writes were made.

| Required gate | rc | Result |
|---|---:|---|
| `/bin/bash -n .../skill_resolution_spike.sh` | 0 | Bash 3.2.57 syntax passes |
| `shellcheck -s bash .../skill_resolution_spike.sh` | 0 | No findings; installed ShellCheck used |
| `SPIKE_TEST_ROOT=... /bin/bash ... --pin "$PWD" --v1 "$PWD" --dry-run` | 0 | All 207 planned arms emit `NOT_RUN`; 16/16 parent listings match SHA and line count |
| Mutated copy, `--only codex --reps 1 --fence off --dry-run` | 1 | Expected failure: skipped scratch cleanup detected as `SAFETY_FAILURE`; copy and deliberate leak subsequently removed |

```text
SUMMARY runid=1791873609-6f8105a9fb915f3232a78940 arms=207 verdicts=207 controls_ok=0/179 safety=OK
SUMMARY runid=1791873613-2718b2048dcd656fe89b00ba arms=50 verdicts=50 controls_ok=0/43 safety=SAFETY_FAILURE
```

The dry run's `0/179` means controls were planned and not executed; it does not claim successful inference controls. Every subject and control has a verdict. Before/after hashes and counts match for the synthetic user Claude/Agents/Codex parents, v1 parents, synthetic pin parents, existing pin skill parents, World parent, runtime parent and synthetic skill parents. The missing pi parent stays missing, the pre-existing fixture file and dangling symlink stay untouched, and every ledger-owned directory is absent after the successful run.

Full [dry-run arm table](runs/1791873609-6f8105a9fb915f3232a78940/arm-table.tsv), [verdict table](runs/1791873609-6f8105a9fb915f3232a78940/verdicts.tsv), [parent audit](runs/1791873609-6f8105a9fb915f3232a78940/audit.log), [dry-run output](dry-run-gate.log) and [mutation output](mutation-gate.log) are retained for review. The mutation changes only the COPY's EXIT cleanup to skip its scratch root. Its runtime-parent listing changes from zero to one line, with different hashes, producing nonzero exit. No user directory was repaired.

Eight offline parser fixtures pass (rc 0): exact pair, mixed resource origins, ignoring tokens in echoed/tool content, rejecting a token-bearing negative answer, rejecting manual search, accepting declared-file Codex reads and extracting pi terminal assistant content. These tests invoke only Python. An additional harmless local-shell PGID-runner check returned rc 1 because `/bin/ps` is denied here: **UNINFORMATIVE UNDER SANDBOX**, not a runner pass/fail. No CLI inference was attempted.

Deviations and limits:

- Test-root mode additionally redirects pin marker parents into `<test-root>/pin`: the initial dry run reached a denied mkdir in the real pin `.agents/skills` (rc 1, **UNINFORMATIVE UNDER SANDBOX**). Existing pin parents are still observed. Successful fixture runs validate mutation safety and arm planning, not skill discovery. The separate sandbox finding is retained in `sandbox-finding.log`.
- UTC epoch seconds plus the required 12 random bytes form the runid. Marker names lowercase the premise and stay within the valid 64-character skill-name limit; each repetition has new names and independently generated tokens.
- Python 3 is a required, checked dependency for structured JSON scoring. Known marker-file reads are allowed; searches and unrecognised tool/path operations invalidate discovery. Raw traces remain available for the controller to review catalog provenance; token matches alone do not establish universal precedence.
- Every scenario gets explicit readability probes for each location and pre/post negative controls; named C+/C-, K+/K- and I+/I- baseline controls are also included. This exceeds the minimal control count to cover cwd/root/flag changes and token rotation.
- SC2016 is disabled only around the two fenced write-fixture commands and the fenced cwd/argv command: literal `$1`/`$@` must expand in the child. Other initial ShellCheck findings were fixed.

Real CLI auth, fence behaviour, native catalogs, provider responses and deadline/group signalling require the controller's outside-sandbox run. Auth and resolution results are therefore unmeasured, the sprint JSON remains unchanged with `passes: null`, and M4/Phase 3b decisions remain with the controller/Mark. Dry-run fixture versions are not measured; real runs bank numbered execution-time help and version output. No installed skills, credentials, driver or release configuration were changed.

## Controller addendum (fleet iteration 34, outside the sandbox)

The executor's gates were reproduced outside the sandbox: `bash -n` passed, `shellcheck` passed, and a
real `--dry-run --reps 1` planted into and removed from the real user parents (rc 0, every parent
sha-identical, 161 planned arms). The live runs then exposed eleven instrument defects that no
sandboxed run could have shown. The controller repaired them in place; each is listed in
[findings.md](findings.md) § "Instrument repairs". The script as committed is the repaired version.
After the first concurrent edit, each subsequent live run executed a frozen copy of the script.
Distilled per-call evidence is tracked under `runs/`. Raw streams stay on the rig at
`~/.ailang/state/fleet-i34-spike-raw/`, because the claude init events list personal addresses.
