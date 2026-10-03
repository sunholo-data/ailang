### Fixed — mission controller reads an OpenRouter 402 as capacity, and a pause no longer posts a crash notice (2026-10-03)

pi text-mode prints OpenRouter's credit refusal as a bare `402: {"message":...}`. The driver's
`RUNTIME_QUOTA_SIG` knew four emitters and not this one, so a spent OpenRouter rung read as a crash:
no demotion, no re-walk to the next rung, and a "FAILED — timeout or crash" notice. `^402:` (anchored
at line start, like `^429:`) now joins the runtime-capacity signature, so a 402 demotes the rung and
re-walks the chain, and pauses as `PAUSED-NO-CAPACITY` when nothing is left. Prose that merely
mentions 402 never classifies.

A paused fire already announces itself once (the PAUSE branch's `_mc_notify`). The final rc block no
longer follows it with a second generic "iteration FAILED (rc=N)" notice, and it leaves the rcfail
episode marker alone. Process rc, thresholds, lane order and the billing reader are unchanged.

New hermetic suite `tools/launchd/test_controller_capacity.sh` (fake `pi`, zero inference) runs the real
retry loop, slot verdict and final block; `test_controller_chain.sh` now loads the demote and ration
helpers it was silently missing (63 `command not found` errors at base). The ticket's ration half
(the controller fallback skipping an over-ration OpenRouter rung) was **not reproduced** at HEAD, so no
guard was added. Design: `design_docs/planned/m-controller-capacity-admission.md`.
