---
title: Mission role dispatch
sidebar_label: Mission role dispatch
---

`ailang mission role-run` executes one explicit designer, planner, executor, or
evaluator request through the existing executor adapters. This opt-in first runtime
slice makes routing observable before the durable mission state machine and
`ailang mission add` onboarding are delivered.

## Inspect a request without spending

From an isolated worktree, prepare the example using its actual revision:

```bash
jq --arg workspace "$PWD" --arg revision "$(git rev-parse HEAD)" \
  '.workspace = $workspace | .input_revision = $revision' \
  examples/mission/evaluator-request.json > /tmp/mission-request.json
ailang mission role-run --request /tmp/mission-request.json --dry-run
```

Use friendly model keys available in `ailang models list`. Update `author_models`
to identify every model that authored the material being evaluated. The example
uses an OpenAI author and a DeepSeek evaluator through Pi/OpenRouter.
Dry-run resolves the ordered candidates, origin vendors, harnesses, wire models,
request digest and registry digest. It constructs no executor, probes no provider,
and writes no receipt. It validates configuration, not provider availability.

## Execute an attended request

After inspecting the request, choose a receipt path that does not exist:

```bash
ailang mission role-run --request /tmp/mission-request.json \
  --receipt /tmp/mission-attempt-001.jsonl
```

This invokes a real agent with the adapter's existing permissions. The caller is
trusted and supplies an isolated workspace and accurate author/revision metadata.
The command does not verify the declared Git revision or create a sandbox.

The request requires version 1; mission/work-item/stage/attempt IDs; a role;
an absolute existing workspace; instructions; a revision declaration; ordered
model candidates; timeout of 1–1800 seconds; positive token and USD ceilings.
Evaluators also require author model keys. Unknown fields, trailing JSON, and
requests larger than 1 MiB are rejected. Instructions are limited to 512 KiB.

Evaluator candidates must have a different origin vendor from every declared
author. OpenRouter and Ollama are hosting routes, not origin vendors. Unknown
origin metadata is an error; model rows can declare `model_vendor` explicitly.
Identity is supported by the registry and dispatch configuration, not provider
attestation. Local GPU admission is not supported in this slice; those candidates
are recorded as skipped. A candidate requiring cost guards needs finite positive
input/output prices in the registry.

Candidates can fall through after factory, capability, or health-check failure.
Once execution starts, failure stops the attempt, because the agent may already
have edited files. There is no automatic retry over a potentially changed tree.
Inspect the receipt and workspace before issuing another attempt.

The first slice admits Claude, Codex and Pi adapters; other harnesses are skipped
until their budget contracts have been checked. The command passes deadline, token
and USD limits to the adapter. Token limits count fresh input plus output and are
checked at usage-event boundaries, so an in-flight response can overshoot. The
live USD guard estimates those tokens at registry prices; cache charges are not
included in that live estimate. Final metered cost is also checked. This is not a
hard ceiling on actual billing or a quota reservation. The existing fleet quota
guard is unchanged. Ambient `ANTHROPIC_API_KEY`/`ANTHROPIC_AUTH_TOKEN` for Claude
or `OPENAI_API_KEY` for Codex refuse those subscription routes. Child commands use
the canonical AILANG messaging store/project bindings; no inbox is consumed or
acknowledged by this command.

## Interpret receipts

Execution requires filesystem support for syncing files and their parent directory
(tested on macOS). Unsupported directory sync fails closed before dispatch.
The JSONL receipt is exclusively created with mode 0600. Events are synced before
dispatch and at completion. It records the normalized request, model plan and
registry digest, candidate rejections, selected route, adapter result, session,
finish reason, usage and cost provenance. Treat it as potentially sensitive: it
contains instructions and output. An existing receipt is never overwritten.

A zero exit status means `execution_completed`. `artifact_verified` is always
`false`: output is not an acceptance decision. Execution failures, empty output,
budget termination and receipt failures return nonzero. A crash can leave an
incomplete journal requiring inspection. This journal does not provide leases,
exactly-once execution, or duplicate detection across different receipt paths.

The four live loops continue using their existing drivers. Durable coordinator
state, resource admission, verified artifact handoffs, remote executors, mission
onboarding and workflow comparisons remain subsequent delivery slices.
