# M-WORKER-FOLLOWUP validation

This directory banks acceptance evidence for the policy worker and lazy AI follow-up release. A successful test of ordinary process groups alone does not establish supervised policy cleanup.

Before implementation, the official v0.54.4 Darwin ARM64 runtime (bc078dc1b41308542bbccd3417bf040283e817df) reproduced the managed policy leak: quit and EOF returned Ok(()) in 4 ms and 3 ms while a separately grouped policy worker remained with PPID 1. Both exclusive fixtures were cleaned by positive, fixture-owned PID; no provider calls. The reproducer JSON is retained alongside this document.

Pending final evidence: acknowledgement before success, startup/nested/supervisor-loss cases, async consumer gate, lazy AI API coverage and generated bin offline/help, full/race/lint/CI checks, evaluator result and official released artifact checks. No release acceptance has been claimed yet.
