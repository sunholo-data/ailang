// The shared pi-NDJSON normalisation and the `events` artifact (v6.40.0 M7).
//
//     node --test docker/resident/test-agent-events.mjs                    # laptop
//     RESIDENT_LIB=/usr/local/bin/lib node --test test-agent-events.mjs    # in-image
//
// WHY THIS EXISTS
//
// M7 shows the asking user the story of their own resident run — the commands
// it ran and what came back — where before they got one line of prose after up
// to 180 seconds of nothing. The events ride on the A2A task the platform
// already polls, so the instance's whole contribution is: normalise pi's
// NDJSON once, and keep an artifact current.
//
// "Normalise ONCE" is the load-bearing word. The observatory (M8) reads the
// same stream to build the operator's transcript. Two normalisations of one
// run can disagree, and the specific way they disagree is already on record:
// reading `tool_execution_end.result` as a string yields a transcript that
// says a tool ran and will not say what it returned (found live 2026-09-04,
// fixed in 810380f8d). A second copy of that mapping is a second copy of that
// bug. So the mapping lives in agent-events.mjs and both sinks project from
// it, and the first test below is the one that would catch it coming back.
import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const LIB = process.env.RESIDENT_LIB || new URL("./lib", import.meta.url).pathname;

const textDelta = (d) => ({ type: "message_update", assistantMessageEvent: { type: "text_delta", delta: d } });

describe("agent-events normalisation", () => {
  let ev;
  beforeEach(async () => { ev = await import(`${LIB}/agent-events.mjs?t=${Date.now()}`); });

  it("reads a MULTI-PART tool result, not [object Object] and not empty", () => {
    // THE 2026-09-04 BUG. `result.content[]` carries `.text` per part; reading
    // `result` as a string gave an empty tool_output, which renders as a
    // healthy timeline showing a tool that returned nothing.
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "tool_execution_end", toolName: "bash",
             result: { content: [{ text: "line one" }, { text: "line two" }] } });
    assert.deepEqual(n.drain(), [
      { t: 1, type: "tool_result", tool: "bash", toolCallId: "", output: "line one\nline two" },
    ]);
  });

  it("falls back to JSON for a result shape nobody has seen yet", () => {
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "tool_execution_end", toolName: "x", result: { unexpected: true } });
    assert.equal(n.drain()[0].output, '{"unexpected":true}');
  });

  it("takes a plain string result as itself", () => {
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "tool_execution_end", toolName: "x", result: "done" });
    assert.equal(n.drain()[0].output, "done");
  });

  it("coalesces text deltas into one record, not one per token", () => {
    const n = ev.createNormaliser({ now: () => 1 });
    for (const c of "Hello there") n.push(textDelta(c));
    n.flush();
    const out = n.drain();
    assert.equal(out.length, 1, "one text record, not one per character");
    assert.equal(out[0].text, "Hello there");
  });

  it("flushes text BEFORE a tool call, so the story reads in order", () => {
    // Otherwise the agent narrates its own past: the commentary that preceded
    // a command appears after its result.
    const n = ev.createNormaliser({ now: () => 1 });
    n.push(textDelta("let me look"));
    n.push({ type: "tool_execution_start", toolName: "bash", args: { command: "ls" } });
    n.push({ type: "tool_execution_end", toolName: "bash", result: "a" });
    n.flush();
    assert.deepEqual(n.drain().map((e) => e.type), ["text", "tool_use", "tool_result"]);
  });

  it("carries BOTH the raw args and the capped JSON string", () => {
    // The observatory's hooks handler takes json.RawMessage and rejects a
    // string; the exec API and the artifact both want the string. Deriving one
    // from the other at each sink is how they drift.
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "tool_execution_start", toolName: "write", args: { path: "/tmp/a", content: "x" } });
    const [e] = n.drain();
    assert.deepEqual(e.args, { path: "/tmp/a", content: "x" });
    assert.equal(e.input, '{"path":"/tmp/a","content":"x"}');
  });

  it("caps a tool input that is a whole file", () => {
    const n = ev.createNormaliser({ now: () => 1, maxToolText: 50 });
    n.push({ type: "tool_execution_start", toolName: "write", args: { content: "y".repeat(5000) } });
    assert.equal(n.drain()[0].input.length, 50);
  });

  it("drains only what is new, so a poller does not re-read the whole run", () => {
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "tool_execution_end", toolName: "a", result: "1" });
    assert.equal(n.drain().length, 1);
    assert.deepEqual(n.drain(), [], "a second drain with nothing new yields nothing");
    n.push({ type: "tool_execution_end", toolName: "b", result: "2" });
    assert.equal(n.drain().length, 1);
  });

  it("ignores event types it does not know", () => {
    const n = ev.createNormaliser({ now: () => 1 });
    n.push({ type: "something_new" });
    n.push(null);
    n.push({});
    assert.deepEqual(n.drain(), []);
  });
});

describe("the events artifact", () => {
  let a2a, root;

  beforeEach(async () => {
    root = mkdtempSync(join(tmpdir(), "a2a-story-"));
    process.env.TASK_STATE_DIR = join(root, "state");
    process.env.AGENT_HOME = join(root, "home");
    a2a = await import(`${LIB}/a2a.mjs?t=${Date.now()}`);
  });
  afterEach(() => {
    rmSync(root, { recursive: true, force: true });
    delete process.env.TASK_STATE_DIR; delete process.env.AGENT_HOME;
  });

  // The artifact is written by the run loop, which needs a live pi. These
  // assert the two pure pieces it is built from — the projection and the cap —
  // through the exported test seam, and the LIVE proof is
  // scripts/verify-resident-story.sh, which reads the artifact off a real run.
  it("projects to the lean shape: no args, no toolCallId", () => {
    // `args` is a whole second copy of `input`, and this rides in a task the
    // platform re-fetches every 2 seconds.
    const task = { id: "t1", artifacts: [] };
    a2a.pushEventsForTest(task, [
      { t: 1, type: "tool_use", tool: "bash", toolCallId: "c1", args: { command: "ls" }, input: '{"command":"ls"}' },
    ]);
    const [e] = task.artifacts[0].parts[0].data.events;
    assert.deepEqual(e, { t: 1, type: "tool_use", tool: "bash", input: '{"command":"ls"}' });
  });

  it("keeps the response artifact when the events artifact is written", () => {
    // `attachText` used to assign `task.artifacts = [one]`. With two artifacts
    // that means whichever wrote last wins, and the story would flicker in and
    // out of existence as the answer streamed.
    const task = { id: "t1", artifacts: [] };
    a2a.attachTextForTest(task, "the answer");
    a2a.pushEventsForTest(task, [{ t: 1, type: "text", text: "thinking" }]);
    a2a.attachTextForTest(task, "the answer, longer");
    assert.deepEqual(task.artifacts.map((x) => x.artifactId).sort(), ["events", "response"]);
    assert.equal(task.artifacts.find((x) => x.artifactId === "response").parts[0].text, "the answer, longer");
    assert.equal(task.artifacts.find((x) => x.artifactId === "events").parts[0].data.events.length, 1);
  });

  it("drops the OLDEST events at the cap and says it truncated", () => {
    const task = { id: "t1", artifacts: [] };
    for (let i = 0; i < 6; i++) {
      a2a.pushEventsForTest(task, [{ t: i, type: "text", text: `e${i}` }], { maxEvents: 3 });
    }
    const data = task.artifacts[0].parts[0].data;
    assert.deepEqual(data.events.map((e) => e.text), ["e3", "e4", "e5"],
      "the tail is what the WIP indicator reads and what anyone is watching");
    assert.equal(data.truncated, true);
  });

  it("stays truncated after a tick that dropped nothing", () => {
    // The flag is a property of the RUN, not of the last append. Recomputing
    // it per call would clear it on the next quiet tick and tell the reader
    // they are seeing everything. Exercised by raising the cap, which is the
    // only way a later append drops nothing after an earlier one did.
    const task = { id: "t1", artifacts: [] };
    a2a.pushEventsForTest(task, [{ t: 1, type: "text", text: "a" }, { t: 2, type: "text", text: "b" },
                                 { t: 3, type: "text", text: "c" }], { maxEvents: 2 });
    assert.equal(task.artifacts[0].parts[0].data.truncated, true);
    a2a.pushEventsForTest(task, [{ t: 4, type: "text", text: "d" }], { maxEvents: 50 });
    const data = task.artifacts[0].parts[0].data;
    assert.equal(data.events.length, 3, "nothing was dropped on this tick");
    assert.equal(data.truncated, true, "and it is still not the whole run");
  });

  it("marks a clipped field instead of silently shortening it", () => {
    const task = { id: "t1", artifacts: [] };
    a2a.pushEventsForTest(task, [{ t: 1, type: "tool_result", tool: "bash", output: "z".repeat(40) }],
                          { maxField: 10 });
    const out = task.artifacts[0].parts[0].data.events[0].output;
    assert.match(out, /^z{10}\n… \[30 more characters\]$/);
  });

  it("writes nothing at all when there is nothing new", () => {
    // A run with no tool calls must not grow an empty artifact: the platform
    // renders prose-only when the artifact is absent, and an empty one is a
    // different, worse thing — a story surface with nothing in it.
    const task = { id: "t1", artifacts: [] };
    a2a.pushEventsForTest(task, []);
    assert.deepEqual(task.artifacts, []);
  });
});
