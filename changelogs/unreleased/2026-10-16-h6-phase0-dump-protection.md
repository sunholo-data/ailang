### Fixed — executor parent is non-dumpable; pre-baked plugin hooks root-owned (audit H-6 Phase 0)

- `ailang coordinator execute-job` now marks itself non-dumpable (`PR_SET_DUMPABLE 0`) before spawning
  the agent CLI, so a same-UID agent can no longer read the job's secrets from `/proc/1/environ` or its
  memory from `/proc/$PPID/mem`. Failure is fatal and reported, never skipped
  (`proctree.DisableDump`, `ErrDumpProtectionUnsupported` off Linux).
- `Dockerfile.agent-base` keeps `/plugins` and the pre-baked `ailang_bootstrap` hooks root-owned. Only
  `/plugins/tasks` is writable by the agent user, and the per-task runtime plugin clone moved there.
- New Linux probe test spawns a same-UID child through the executor spawn path and asserts both reads are
  denied, with a control run that must leak. `TestProctreeIsALeaf` now checks the linux, darwin and windows builds (a Linux-only import used to pass on a Mac). Design: `design_docs/planned/v0_49_1/m-executor-uid-split.md`.
- Design rulings (attended 2026-10-16): D1 accepts the leaked refresh tokens as unverified rather than rotating (objects deleted, cloud on a setup-token); the cloud spike runs on a temporary dev-only Terraform job.
- D6 ruled yes: local agent CLIs on the Mac rig get a dedicated hidden service account (not `daneel`); design doc commissioned.
