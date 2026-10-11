# M-WORKER-FOLLOWUP validation

The official v0.54.4 Darwin ARM64 binary reproduced the leak after a restricted worker wrote `actual worker entered main`: quit and EOF returned `Ok(())` in 3 ms while its positive PID survived under PPID 1. Both uniquely owned fixtures were cleaned. `v0.54.4-policy-marker-before.json` banks this AC1 baseline.

The integrated runtime passes managed quit/EOF with actual worker-entry acknowledgement, public `Ok(())`, every recorded PID absent (including zombies), and the shared deadline. The original consumer managed and two-async probes also pass. Its async EOF case remains natural completion after the worker duration; EOF notification delivery is outside this sprint. All probes use zero live providers.

The 19-case AI probe verifies direct and generated-bin offline/help with missing or unreadable credentials and unavailable Google ADC/Ollama, strict offline execution, visible first-effect authentication failure and deterministic stubs. The M3 summary also covers local mock text/image/ref/speech/step/cache/stream APIs, once-only context/error publication, immutable auth/config/routing and named streaming with budget views.

M2 summaries bank the 14-case managed/two-async exit matrix, nested deadline, supervisor loss and bystander survival, immediate timeout/output sentinels, early CLI exits, invalid request failure receipts, standalone supervisor cleanup failures and final-exit lifeline ownership. Full integrated pre-release gates, formatting, boundaries and examples pass. The independent evaluator passes at **87/100**; its full local gates are retained by SHA256 in the report. Atomic startup PID publication passes 20 supervisor-loss repetitions.

- [Integrated gate proof](integrated-gates.json)
- [Independent evaluation](independent-evaluation-round-1.md)
- [Marker-backed cleanup](policy-marker-integrated.json)
- [Managed consumer](policy-consumer-integrated.json), [async consumer](async-consumer-integrated.json)
- [AI CLI and generated-bin probe](ai-final-integrated.json)
- [M2 checkpoints](m2-implementation-tests.json), [M3 checkpoints](m3-implementation-tests.json)

Final PR #1765 passed all platform checks and merged into dev at `e723303f4fead56b243dd9292a746baf52de0745`; main dev CI also passed. [Async output/receipt corrections](async-output-ci-corrections.json) bank deterministic failing controls, repeated successful fixes and independent focused races. [Final-source marker cleanup](policy-marker-final-source.json) and [final-source async consumer](async-consumer-final-source.json) pass.

The dev release matrix exposed an existing memory-probe baseline mismatch; two attempts exceeded the unchanged 30 MB bound. The independently reviewed matched-workload correction retains raw diagnostics and catches genuine retention at61/53 MB. Final macOS PR checks passed at-2/+9 MB and dev checks at +10/+3 MB. The Linux measurement-floor limitation remains explicit.

[Released v0.54.5 delivery](release/README.md) banks all final CI, signed-artifact and official consumer retests, brain refresh, TEST/PROD, public MCP 0.54.5 and canonical release-notice proof. No live AI providers were called. Original independent implementation score87/100 remains historical; final delivery is supported by the later records.
