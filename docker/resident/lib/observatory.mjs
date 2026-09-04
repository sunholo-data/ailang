// Observatory session reporting for the resident agent (v6.40.0 M8).
//
// WHY THIS EXISTS ALONGSIDE otel.mjs
//
// Two planes, and the dashboard shows the second one. OTel spans correlate
// services; what an operator actually READS is a task record with its turns and
// tool calls — `POST /api/exec/sessions` creates a coordinator.TaskRecord and
// `POST /api/exec/events` hangs the transcript off it
// (internal/server/handlers_exec.go). That is how a Cloud Run agent job and a
// Claude Code session appear, and a resident run emitting only spans would be
// "traced" and still absent from the view that matters.
//
// The mapping is not a translation so much as a coincidence worth using: pi's
// NDJSON event types and the exec API's stream types describe the same thing.
//
//     pi                                     exec stream_type
//     ─────────────────────────────────────  ────────────────
//     (run start)                            turn_start
//     message_update / text_delta            text       (COALESCED — see below)
//     tool_execution_start                   tool_use
//     tool_execution_end                     tool_result
//     message_end                            turn_end
//     (a throw, or a non-zero exit)          error
//
// TEXT IS COALESCED, DELIBERATELY. pi emits one message_update per token. A
// POST each would make telemetry the bottleneck of the thing it measures, and
// would bury the tool calls — the events anyone is actually looking for — under
// thousands of one-character rows. Deltas accumulate and flush as one `text`
// event at the next tool call or at the end of the turn, which also keeps the
// transcript in the order it happened.
//
// THREE ENDPOINTS, because the executors use three and parity is the point.
//
//   /api/observatory/hooks      the SESSIONS table. What OTel spans are
//                               enriched against, and what makes this a
//                               first-class session rather than a task record
//                               that happens to exist. Its SessionStart 400s
//                               without a workspace, so that is not optional.
//   /api/exec/sessions          creates the coordinator.TaskRecord
//   /api/exec/events            the transcript hung off it (the Chat History view)
//
// claude_telemetry.sh posts to all three for a Claude Code session; a resident
// run posts the same shapes, from pi's events instead of Claude Code's hooks.
//
// Same failure posture as otel.mjs: unset URL is inert, an unreachable
// observatory drops events and warns ONCE. Reporting must never be the reason a
// turn fails.

const MAX_TOOL_INPUT = 8000; // bounded: a write tool's `content` can be a whole file
const MAX_TEXT = 100_000;

let cfg = { url: "" };
let warned = false;

export function configure({ url = "" } = {}) {
  cfg = { url: String(url || "").replace(/\/+$/, "") };
  warned = false;
  return enabled();
}

export function enabled() {
  return cfg.url !== "";
}

function warnOnce(message) {
  if (warned) return;
  warned = true;
  console.warn(`observatory | ${message}`);
}

async function post(path, body) {
  if (!enabled()) return null;
  try {
    const res = await fetch(`${cfg.url}${path}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(body),
      // Short, like the Go side's OTLP timeout: a slow dashboard must not
      // become a slow agent.
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) {
      warnOnce(`${path} returned ${res.status} — session reporting is degraded, the agent is not`);
      return null;
    }
    return await res.json().catch(() => ({}));
  } catch (e) {
    warnOnce(`${path} unreachable (${String(e.message).slice(0, 120)}) — session reporting is degraded, the agent is not`);
    return null;
  }
}

/**
 * pi's tool result, as text.
 *
 * `tool_execution_end` carries `result.content[]` with `.text` per part, not a
 * string — the same shape `flattenPiToolResult` handles in
 * internal/executor/pi/pi.go. Reading it as a string produced a transcript that
 * said a tool ran and refused to say what it returned: an empty `tool_output`
 * in the dashboard, found on the first live run (2026-09-04).
 */
function toolOutputOf(ev) {
  const r = ev.result ?? ev.output;
  if (r == null) return "";
  if (typeof r === "string") return r;
  if (Array.isArray(r.content)) {
    return r.content.map((c) => c?.text ?? "").filter(Boolean).join("\n");
  }
  // Something new. JSON beats "[object Object]", which is the shape of this
  // bug and tells a reader nothing at all.
  try {
    return JSON.stringify(r);
  } catch {
    return "";
  }
}

const INERT_RUN = {
  event() {},
  async flush() {},
  async finish() {},
};

/**
 * Open a reported run. `sessionId` is the A2A TASK id, so the record the
 * dashboard shows (`exec-<first 8>`) ties back to the task the platform holds
 * and to the `a2a.task.id` attribute carried on the spans.
 *
 * ⚠️ THE DASHBOARD RECORD IS KEYED ON THE FIRST 8 CHARACTERS.
 * `handlers_exec.go` derives `task_id = "exec-" + session_id[:8]`, so two runs
 * whose ids share a prefix collapse into ONE record — which looks exactly like
 * working telemetry, with one enormous fake session. Safe today because an A2A
 * task id defaults to `randomUUID()`, and unsafe the moment a caller passes a
 * PREFIXED `metadata.runId`: the platform's own thread ids are `aitana-<hash>`,
 * and every one of those truncates to `exec-aitana-`. Verified live 2026-09-04
 * (`m8probe-1788530540` -> `exec-m8probe-`). If runId is ever plumbed through
 * `a2a_client.converse`, put the entropy in the first 8 characters or fix the
 * truncation.
 */
export function startRun({ sessionId, workspace = "/workspace", provider = "pi" }) {
  if (!enabled()) return INERT_RUN;

  // Queued rather than awaited: startRun is called on the request path and the
  // dashboard is not allowed to add latency to it. Ordering is preserved by
  // chaining every send onto this same promise.
  let chain = Promise.all([
    post("/api/exec/sessions", { session_id: sessionId, workspace, provider }),
    hook({ event: "SessionStart", workspace, claude_version: provider }),
  ]).then(() => send({ stream_type: "turn_start", text: "resident run started" }));
  let pending = "";
  let turn = 0;

  function send(fields) {
    return post("/api/exec/events", { session_id: sessionId, turn_num: turn, ...fields });
  }
  function hook(fields) {
    return post("/api/observatory/hooks", {
      session_id: sessionId,
      timestamp: new Date().toISOString(),
      ...fields,
    });
  }
  function enqueue(fields) {
    chain = chain.then(() => send(fields)).catch(() => {});
  }
  function enqueueHook(fields) {
    chain = chain.then(() => hook(fields)).catch(() => {});
  }
  function flushText() {
    if (!pending) return;
    const text = pending.slice(0, MAX_TEXT);
    pending = "";
    enqueue({ stream_type: "text", text });
  }

  return {
    /** Feed pi's NDJSON events straight in; unknown types are ignored. */
    event(ev) {
      switch (ev?.type) {
        case "message_update":
          if (ev.assistantMessageEvent?.type === "text_delta" && ev.assistantMessageEvent.delta) {
            pending += ev.assistantMessageEvent.delta;
          }
          break;
        case "tool_execution_start":
          // Text first, so the transcript reads in the order it happened rather
          // than showing the agent narrating its own past.
          flushText();
          enqueue({
            stream_type: "tool_use",
            tool_name: ev.toolName || "unknown",
            tool_input: JSON.stringify(ev.args ?? ev.toolInput ?? {}).slice(0, MAX_TOOL_INPUT),
          });
          // tool_input/tool_response are json.RawMessage on the hooks handler,
          // so they must be JSON VALUES, not the strings the exec API takes.
          enqueueHook({
            event: "PreToolUse",
            tool_name: ev.toolName || "unknown",
            tool_use_id: ev.toolCallId || "",
            tool_input: ev.args ?? ev.toolInput ?? {},
          });
          break;
        case "tool_execution_end":
          {
            const out = toolOutputOf(ev).slice(0, MAX_TOOL_INPUT);
            enqueue({ stream_type: "tool_result", tool_name: ev.toolName || "unknown", tool_output: out });
            enqueueHook({
              event: "PostToolUse",
              tool_name: ev.toolName || "unknown",
              tool_use_id: ev.toolCallId || "",
              tool_response: out,
            });
          }
          break;
        case "message_end":
          flushText();
          turn += 1;
          break;
        default:
          break;
      }
    },

    /** Await everything queued so far. Used by tests and before finishing. */
    async flush() {
      flushText();
      await chain;
    },

    /**
     * Close the run. A failed run that reports no terminal event is
     * indistinguishable from one still going, which is the state the dashboard
     * exists to rule out.
     */
    async finish({ ok = true, error = "", usage = null } = {}) {
      flushText();
      enqueue(
        ok
          ? { stream_type: "turn_end", text: usage ? `usage: ${JSON.stringify(usage)}` : "run complete" }
          : { stream_type: "error", error_msg: String(error || "run failed") },
      );
      // Stop closes the SESSION as well as the transcript. A session that never
      // stops is a session still running, as far as the dashboard is concerned.
      enqueueHook({ event: "Stop" });
      await chain;
    },
  };
}
