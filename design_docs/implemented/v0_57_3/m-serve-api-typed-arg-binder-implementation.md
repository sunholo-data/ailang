# M-SERVE-API-TYPED-ARG-BINDER implementation

Implemented on `coordinator/task-f9a4781c` from the approved planner artifacts
at `cd75b00c` (`coordinator/task-96d9529b`). Only the two missing sprint
artifacts were restored; unrelated planner-branch history was not merged.

## Result

One `bindCallArgs` boundary converts resolved arguments before engine calls on
MCP tools/call, UI resources, REST (generic and custom routes), and A2A.
Runtime-only declared schemas retain scalars, nested records, lists, arrays,
local record aliases, labelled types and bounded unknown leaves. Integers reject
fractions, non-finite values and values outside the native integer range.
Floats retain their declared type even for integral JSON numbers. Errors name
the parameter and nested field/index in declaration order.

Typed omission zeros preserve integer/float identity; arrays use ArrayValue,
including empty arrays in optional records. Existing eval.Value objects pass
through unchanged for files, headers, raw requests and bytes. Unknown types
and undeclared record fields retain embed.FromGo conversion. Embed's public
conversion API, WebSocket calls and token-verifier calls are unchanged.

## Verification

- Focused schema/binder/MCP/REST/A2A tests pass, including the six primary
  parity rows, boolean/string rejections, nested records/lists, positional calls,
  arrays and optional arrays. A focused suite also passed with `-count=20`.
- Existing omission, multipart, file, header, raw and query-resolution regression
  tests are included in the API-server package gate.
- `go test ./internal/embed` passes.
- `go vet ./internal/apiserver` passes.
- Architecture boundaries, 800-line file limit, changelog validation and
  `git diff --check` pass.
- `make verify-examples` passes: 266 passed, 10 skipped; manifest in sync.
- Fresh repository binary: showcase type-check, five inline tests and two
  generated property tests pass. Z3 verifies twice's contract under the encoded
  mathematical integer model; this is not a machine-overflow equivalence proof.
- `make lint` cannot run: golangci-lint is absent. No tools were installed.
- `make test-changed`: API-server package passes (78.926s); broad gate fails
  the two unchanged effects tests below. M6 records passes=false for this
  acceptance exception, despite completed feature work.

The unchanged effects package separately reproduces two failures:
`TestRuntimeNestedStopUsesRemainingDeadline` (nested fixture survives stop) and
`TestProcessExec_TimeoutKillsGrandchildren` (grandchild survives timeout).
Neither effects source nor process-lifetime behavior was changed here. These
remain verification failures; they are not counted as passes.

## Independent review

The sprint-executor skill's independent evaluator reviewed schema extraction,
conversion and all four integrations. It identified the distinct array runtime
representation and missing showcase requirements; both were corrected and tested.
No remaining concrete runtime defect was reported. Its provisional 88/100 score
was conditional on final gates; it is not an unconditional PASS verdict because
lint and the broad effects gate remain unresolved.

## Execution notes

- Sequential milestones M1–M5 were committed separately. M6 contains transport
  parity tests, documentation, the showcase and manifest metadata.
- AILANG teaching prompt v0.16.11 was loaded and repeated with the freshly built
  binary. The showcase includes requires/ensures, explicit pure effects and
  inline tests, as planned.
- The array regression first failed with `array bound as *eval.ListValue`, then
  passed after preserving ArrayValue in both binding and defaults.
- Broad testing exposed inherited test-server stdlib environment state; the new
  fixture now sets its own stdlib path with t.Setenv, following existing fixtures.
- The example verifier requires an entrypoint; the showcase now has a deterministic
  main hidden from HTTP and MCP.
- Startup helper's jq/bc issues and missing Go PATH produced an unreliable initial
  verdict. Explicit Go commands used `/usr/local/go/bin` and supersede it.
- Cloud branch sync reports `skipped:no-task-id`. Work is committed locally;
  publication and merge remain with the coordinator.

The CLI handoff to sprint-evaluator was refused because no agent serves that
inbox. No force-send was attempted. The independent local review above and
coordinator output artifacts preserve the handoff.
