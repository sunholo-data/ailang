// Observatory session reporting (v6.40.0 RESIDENT-P1 M8).
//
//     node --test docker/resident/test-observatory.mjs                       # laptop
//     RESIDENT_LIB=/usr/local/bin/lib node --test test-observatory.mjs       # in-image
//
// WHY THIS EXISTS ALONGSIDE test-otel.mjs
//
// Spans and sessions are different planes and the dashboard shows the second
// one. An agent job and a Claude Code session appear in the observatory as
// TASK RECORDS with turns and tool calls (server/handlers_exec.go creates a
// coordinator.TaskRecord per exec session); OTel spans are the correlation
// between services, not the thing an operator reads. A resident that emitted
// only spans would satisfy "is traced" and still be absent from the view that
// matters — which is the whole complaint M8 was written to fix.
import { after, before, describe, it } from "node:test";
import assert from "node:assert/strict";
import { createServer } from "node:http";

const LIB = process.env.RESIDENT_LIB || new URL("./lib", import.meta.url).pathname;
const obsPath = `${LIB}/observatory.mjs`;

function stubObservatory() {
  const posts = [];
  const server = createServer((req, res) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      let body = {};
      try {
        body = JSON.parse(Buffer.concat(chunks).toString());
      } catch {}
      posts.push({ url: req.url, body });
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify({ task_id: "exec-abc12345", status: "created" }));
    });
  });
  return {
    posts,
    listen: () => new Promise((r) => server.listen(0, "127.0.0.1", () => r(`http://127.0.0.1:${server.address().port}`))),
    close: () => new Promise((r) => server.close(r)),
  };
}

const eventsAt = (posts) => posts.filter((p) => p.url === "/api/exec/events").map((p) => p.body);
const sessionsAt = (posts) => posts.filter((p) => p.url === "/api/exec/sessions").map((p) => p.body);

describe("observatory session reporting", () => {
  let obs;
  let url;

  before(async () => {
    obs = stubObservatory();
    url = await obs.listen();
  });
  after(async () => obs && obs.close());
  const fresh = () => obs.posts.splice(0, obs.posts.length);

  it("is inert when no observatory is configured", async () => {
    const o = await import(`${obsPath}?case=off`);
    o.configure({ url: "" });
    assert.equal(o.enabled(), false);
    const run = o.startRun({ sessionId: "t-1", workspace: "/workspace", provider: "pi" });
    run.event({ type: "tool_execution_start", toolName: "bash", toolCallId: "c1" });
    await run.finish({ ok: true });
  });

  it("opens a session the dashboard can list", async () => {
    const o = await import(`${obsPath}?case=session`);
    o.configure({ url });
    fresh();
    const run = o.startRun({ sessionId: "task-abc12345", workspace: "/workspace", provider: "pi" });
    await run.flush();
    const s = sessionsAt(obs.posts)[0];
    // session_id is the A2A TASK id, so the record the dashboard shows
    // (`exec-<first 8>`) can be tied back to the task the platform holds and to
    // the a2a.task.id attribute on the spans.
    assert.deepEqual(s, { session_id: "task-abc12345", workspace: "/workspace", provider: "pi" });
    await run.finish({ ok: true });
  });

  it("does NOT make an HTTP request per text delta", async () => {
    const o = await import(`${obsPath}?case=chatty`);
    o.configure({ url });
    fresh();
    const run = o.startRun({ sessionId: "t-chatty", workspace: "/w", provider: "pi" });
    // pi emits one message_update per token. A POST each would make telemetry
    // the bottleneck of the thing it is measuring, and would bury the tool
    // calls — which are the interesting events — under thousands of text rows.
    for (let i = 0; i < 500; i++) {
      run.event({ type: "message_update", assistantMessageEvent: { type: "text_delta", delta: "x" } });
    }
    await run.flush();
    const events = eventsAt(obs.posts);
    assert.ok(events.length <= 2, `500 deltas produced ${events.length} event posts`);
    const text = events.filter((e) => e.stream_type === "text");
    assert.equal(text.length, 1, "deltas should coalesce into one text event");
    assert.equal(text[0].text.length, 500, "and must not lose any of the text");
    await run.finish({ ok: true });
  });

  it("reports a tool call as its own event, promptly", async () => {
    const o = await import(`${obsPath}?case=tools`);
    o.configure({ url });
    fresh();
    const run = o.startRun({ sessionId: "t-tools", workspace: "/w", provider: "pi" });
    run.event({ type: "message_update", assistantMessageEvent: { type: "text_delta", delta: "thinking" } });
    run.event({ type: "tool_execution_start", toolName: "bash", toolCallId: "c1", args: { command: "ls" } });
    await run.flush();
    const events = eventsAt(obs.posts);
    const tool = events.find((e) => e.stream_type === "tool_use");
    assert.ok(tool, "no tool_use event was reported");
    assert.equal(tool.tool_name, "bash");
    assert.equal(tool.session_id, "t-tools");
    assert.match(tool.tool_input, /ls/);
    // Ordering matters in a transcript: text said BEFORE a tool call must not
    // arrive after it, or the dashboard shows the agent narrating its own past.
    const order = events.map((e) => e.stream_type);
    assert.ok(order.indexOf("text") < order.indexOf("tool_use"), `out of order: ${order.join(",")}`);
    await run.finish({ ok: true });
  });

  it("bounds a tool input rather than shipping whatever it was given", async () => {
    const o = await import(`${obsPath}?case=bounds`);
    o.configure({ url });
    fresh();
    const run = o.startRun({ sessionId: "t-big", workspace: "/w", provider: "pi" });
    run.event({ type: "tool_execution_start", toolName: "write", toolCallId: "c1", args: { content: "y".repeat(200000) } });
    await run.flush();
    const tool = eventsAt(obs.posts).find((e) => e.stream_type === "tool_use");
    assert.ok(tool.tool_input.length < 20000, `tool_input was ${tool.tool_input.length} bytes`);
    await run.finish({ ok: true });
  });

  it("closes the run with a terminal event, success or failure", async () => {
    const o = await import(`${obsPath}?case=terminal`);
    o.configure({ url });

    fresh();
    const good = o.startRun({ sessionId: "t-ok", workspace: "/w", provider: "pi" });
    await good.finish({ ok: true, usage: { input: 10, output: 20 } });
    assert.ok(eventsAt(obs.posts).some((e) => e.stream_type === "turn_end"));

    fresh();
    const bad = o.startRun({ sessionId: "t-bad", workspace: "/w", provider: "pi" });
    await bad.finish({ ok: false, error: "pi exited 1" });
    const err = eventsAt(obs.posts).find((e) => e.stream_type === "error");
    // A failed run that reports no terminal event is indistinguishable from one
    // still running, which is the state the whole dashboard exists to rule out.
    assert.ok(err, "a failed run reported no error event");
    assert.match(err.error_msg, /pi exited 1/);
  });

  it("survives an observatory that is not there, and says so ONCE", async () => {
    const o = await import(`${obsPath}?case=down`);
    o.configure({ url: "http://127.0.0.1:1" });
    const warnings = [];
    const realWarn = console.warn;
    console.warn = (...a) => warnings.push(a.join(" "));
    try {
      const run = o.startRun({ sessionId: "t-down", workspace: "/w", provider: "pi" });
      run.event({ type: "tool_execution_start", toolName: "bash", toolCallId: "c1" });
      await run.flush();
      await run.finish({ ok: true });
    } finally {
      console.warn = realWarn;
    }
    assert.ok(warnings.length <= 1, `expected at most one warning, got ${warnings.length}`);
  });
});
