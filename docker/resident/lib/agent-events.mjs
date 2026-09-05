// One normalisation of pi's NDJSON into the event vocabulary this runtime
// speaks (v6.40.0 M7).
//
// WHY THIS IS ITS OWN MODULE
//
// Two sinks now read the same stream. The observatory (M8) turns it into a
// session transcript a Sunholo operator reads; the `events` artifact (M7)
// turns it into the story the ASKING USER reads in their workbench. They are
// the same facts about the same run, and if they were normalised twice they
// could disagree — which for a transcript and the story of the same run is not
// a cosmetic problem.
//
// It also means the expensive lessons are learned once. `toolOutputOf` below
// is the second attempt: the first read `result` as a string and produced a
// transcript that said a tool ran and would not say what it returned, found on
// the first live run (2026-09-04, fixed in 810380f8d). A second copy of that
// mapping is a second copy of that bug, waiting.
//
//     pi event                                normalised type
//     ─────────────────────────────────────   ───────────────
//     message_update / text_delta             text   (COALESCED)
//     tool_execution_start                    tool_use
//     tool_execution_end                      tool_result
//     message_end                             turn_end
//
// TEXT IS COALESCED, DELIBERATELY. pi emits one message_update per token. One
// record each would bury the tool calls — the events anyone is actually
// looking for — under thousands of one-character rows, and for the observatory
// it would make telemetry the bottleneck of the thing it measures. Deltas
// accumulate and flush as one `text` record at the next tool call or at the
// end of the turn, which also keeps the sequence in the order it happened.

export const MAX_TOOL_TEXT = 8000; // a write tool's `content` can be a whole file
export const MAX_TEXT = 100_000;

/**
 * pi's tool result, as text.
 *
 * `tool_execution_end` carries `result.content[]` with `.text` per part, not a
 * string — the same shape `flattenPiToolResult` handles in
 * internal/executor/pi/pi.go. Reading it as a string produced an empty
 * `tool_output` in the dashboard: a tool that ran and returned nothing, which
 * renders as a healthy timeline and is not.
 */
export function toolOutputOf(ev) {
  const r = ev?.result ?? ev?.output;
  if (r == null) return "";
  if (typeof r === "string") return r;
  if (Array.isArray(r.content)) {
    return r.content.map((c) => c?.text ?? "").filter(Boolean).join("\n");
  }
  // Something new. JSON beats "[object Object]", which is the shape of this
  // bug and tells a reader nothing at all.
  try { return JSON.stringify(r); } catch { return ""; }
}

/**
 * A stateful normaliser over one run's event stream.
 *
 * `push` accumulates, `drain` takes what has accrued since the last call. Both
 * sinks can therefore consume at their own rate — the observatory posts as it
 * goes, the artifact rebuilds on demand — without either re-deriving the
 * mapping or holding the other up.
 *
 * `now` is injectable because a test that asserts on ordering should not also
 * have to be a test of the clock.
 */
export function createNormaliser({
  maxToolText = MAX_TOOL_TEXT,
  maxText = MAX_TEXT,
  now = () => Date.now() / 1000,
} = {}) {
  let pending = "";
  let out = [];

  function flushText() {
    if (!pending) return;
    out.push({ t: now(), type: "text", text: pending.slice(0, maxText) });
    pending = "";
  }

  return {
    /** Feed pi's NDJSON events straight in; unknown types are ignored. */
    push(ev) {
      switch (ev?.type) {
        case "message_update":
          if (ev.assistantMessageEvent?.type === "text_delta" && ev.assistantMessageEvent.delta) {
            pending += ev.assistantMessageEvent.delta;
          }
          break;
        case "tool_execution_start": {
          // Text first, so the sequence reads in the order it happened rather
          // than showing the agent narrating its own past.
          flushText();
          const args = ev.args ?? ev.toolInput ?? {};
          out.push({
            t: now(), type: "tool_use",
            tool: ev.toolName || "unknown",
            toolCallId: ev.toolCallId || "",
            // The raw object AND the capped string: the observatory's hooks
            // handler takes json.RawMessage and would reject a string, the
            // exec API and the artifact both want the string. Deriving one
            // from the other at each sink is how they drift.
            args,
            input: safeJson(args).slice(0, maxToolText),
          });
          break;
        }
        case "tool_execution_end":
          out.push({
            t: now(), type: "tool_result",
            tool: ev.toolName || "unknown",
            toolCallId: ev.toolCallId || "",
            output: toolOutputOf(ev).slice(0, maxToolText),
          });
          break;
        case "message_end":
          flushText();
          out.push({ t: now(), type: "turn_end", usage: ev.usage ?? null });
          break;
        default:
          break;
      }
    },

    /** Emit any buffered text. Call before reading a run's final sequence. */
    flush() { flushText(); },

    /** Take everything accrued since the last drain. */
    drain() { const r = out; out = []; return r; },
  };
}

function safeJson(v) {
  try { return JSON.stringify(v) ?? ""; } catch { return ""; }
}
