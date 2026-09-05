// A2A surface for the resident agent (v6.40.0 RESIDENT-P1 M2, Decision 2c).
//
// The platform talks to this instance as an A2A PEER, not through a bespoke
// REST API. Every piece of vocabulary here is the specification's, not ours:
//
//   herdr `blocked`  -> TASK_STATE_INPUT_REQUIRED ("the agent requires
//                       additional user input to proceed")
//   answering        -> a follow-up message carrying the SAME task id
//   nobody connected -> push notification config: the agent POSTs task updates
//                       to a registered webhook
//
// Task ids are the platform's run ids, so the two systems share one identifier
// rather than maintaining a correlation table.
import { readFileSync, writeFileSync, mkdirSync, existsSync,
         openSync, writeSync, fsyncSync, closeSync, chmodSync } from "node:fs";
import { randomUUID } from "node:crypto";
import * as herdr from "./herdr.mjs";
import * as otel from "./otel.mjs";
import * as observatory from "./observatory.mjs";
import { runPi, capabilities } from "./pi.mjs";

const PI_HOME = process.env.PI_HOME || "/home/ailang/.pi";
const STATE_DIR = process.env.TASK_STATE_DIR || "/home/ailang/.resident";
const STATE_FILE = `${STATE_DIR}/tasks.json`;
// M6: the durable half. Local disk is reset by a stop/resume — Cloud Run
// documents the writable layer as deleted on shutdown, and M4's sweep performs
// exactly that stop after 30 idle minutes — so the mount is the only place a
// task can outlive the lifecycle the platform relies on.
const AGENT_HOME = process.env.AGENT_HOME || "/agent-home";
const CHECKPOINT_DIR = `${AGENT_HOME}/.resident`;
const CHECKPOINT_FILE = `${CHECKPOINT_DIR}/tasks.json`;
const AGENT_KIND = process.env.AGENT_KIND || "pi";
const DEFAULT_MODEL = process.env.DEFAULT_MODEL || "";
// stream = pi --mode json directly; interactive = pi as a TUI under herdr.
const DEFAULT_MODE = process.env.DEFAULT_MODE || "stream";

export const TaskState = {
  submitted: "submitted", working: "working", inputRequired: "input-required",
  completed: "completed", failed: "failed", canceled: "canceled", rejected: "rejected",
};

// ─── conversation sessions (M10) ─────────────────────────────────────────────
// A2A already has the right key: contextId is the spec's identifier for a
// conversation spanning several tasks, which is exactly a chat thread. Keying
// pi's session on it means two callers get two conversations on one instance
// with no scheme of our own, and a caller that reuses its contextId is
// resuming by definition rather than by convention.
const sessionIdFor = (contextId) =>
  `ctx-${String(contextId).toLowerCase().replace(/[^a-z0-9-]/g, "").slice(0, 48)}`;

// pi rewrites one file per session for the length of a run, so two concurrent
// turns on the SAME conversation would interleave writes and lose history. Runs
// are therefore chained per session — queued, not refused: a user sending a
// second message before the first finishes is normal chat behaviour, and an
// error there would be our problem presented as theirs. Different conversations
// still run concurrently.
const sessionChains = new Map();
function onSession(sessionId, fn) {
  const prev = sessionChains.get(sessionId) || Promise.resolve();
  const next = prev.then(fn, fn);
  sessionChains.set(sessionId, next.catch(() => {}));
  return next;
}

// ─── concurrency ceiling ─────────────────────────────────────────────────────
// The chain above serialises turns WITHIN one conversation. It does nothing
// across conversations, and each message/send spawns its own pi process, so N
// callers meant N processes on one box with nothing to stop them.
//
// This is a singleton product: Cloud Run instances do not autoscale, which is
// the point — it is what buys the stable URL and the stateful box. So the
// ceiling is real and fixed, and the only question is whether we hit it
// politely or by OOM. M0 measured the cgroup killing the CHILD while the
// container survived, so today the 5th caller silently kills somebody's run.
//
// The limit is a guess pending M6's measurement (1 GiB hosted one agent in M0;
// this box is 4 GiB), which is why it is an env var and why /health reports
// the live count. A number that refuses with its own value in the message is
// honest in a way an OOM is not.
const MAX_CONCURRENT = Number(process.env.RESIDENT_MAX_CONCURRENT_RUNS || 3);
let activeRuns = 0;

// Last time this agent did any WORK — not the last time anything touched the
// process. An idle sweep that counted health probes would never stop anything,
// because the sweep's own probe is traffic.
let lastActivityAt = Date.now();
export const noteActivity = () => { lastActivityAt = Date.now(); };
export const runStats = () => ({
  active: activeRuns,
  max: MAX_CONCURRENT,
  idle_s: activeRuns > 0 ? 0 : Math.round((Date.now() - lastActivityAt) / 1000),
});

// ─── task store ──────────────────────────────────────────────────────────────
// TWO files, and the difference is the whole of M6.
//
// LOCAL DISK is the hot path: `persist()` runs on every state change and
// gcsfuse has no POSIX locking, so the mount is the wrong place to rewrite
// continuously — the same rule M10 applies to pi's sessions.
//
// But local disk does not survive what actually happens to this instance.
// Cloud Run resets the writable layer on stop/resume, and M4's sweep stops the
// instance after 30 idle minutes, so the routine lifecycle event wipes it. The
// live evidence: resident-7e71c9b736f2 booted three times on 2026-09-04 and
// logged "restored N task(s)" on none of them, the third being the boot
// straight after the M5a proof run.
//
// So $AGENT_HOME carries a CHECKPOINT, written at the few moments that matter
// (a state worth waking someone for, a webhook registration, shutdown) rather
// than on every change. Read local first — it is the newer of the two whenever
// the process merely restarted; fall back to the checkpoint when the layer
// underneath it was reset.
let tasks = new Map();

function snapshot() {
  // Wrapped in an object because pushConfigs has to travel with the tasks: a
  // task restored without its webhook is worse than one lost outright, since
  // the caller was told an answer would arrive and nothing will ever say
  // otherwise. `v` so a later shape change can be read rather than guessed.
  return JSON.stringify({ v: 2, tasks: [...tasks.values()], push: [...pushConfigs.entries()] }, null, 2);
}

function readState(file) {
  const raw = JSON.parse(readFileSync(file, "utf8"));
  // Pre-M6 files are a bare array. An instance restarting onto this image must
  // not lose what the previous one left behind.
  if (Array.isArray(raw)) return { tasks: raw, push: [] };
  return { tasks: raw.tasks || [], push: raw.push || [] };
}

function write(file, dir, { sync = false } = {}) {
  mkdirSync(dir, { recursive: true });
  const body = snapshot();
  // 0600: this file carries the per-task push token, which is a bearer
  // credential. It grants only "report on this one task", strictly less than
  // the agent SA that owns the bucket already holds — but it is a credential
  // at rest and is written as one.
  if (!sync) { writeFileSync(file, body, { mode: 0o600 }); return; }
  // FSYNC, on the mount only, and the live run is why.
  //
  // `writeFileSync` returns once the bytes are in the FUSE buffer, NOT once
  // gcsfuse has uploaded the object. A real stop proved the difference: the
  // shutdown handler logged "staging task state", the container was destroyed
  // moments later, and GCS never received the object — the .resident/ prefix
  // did not even exist afterwards. A write that returns success and leaves no
  // object is the worst possible failure for the one file whose entire purpose
  // is surviving that stop. fsync is what makes gcsfuse finalise it.
  const fd = openSync(file, "w", 0o600);
  try {
    writeSync(fd, body);
    fsyncSync(fd);
  } finally {
    closeSync(fd);
  }
  chmodSync(file, 0o600);
}

function persist() {
  try { write(STATE_FILE, STATE_DIR); }
  catch (e) { console.error(`a2a | WARN persist failed: ${e.message}`); }
}

/** Stage the task store to the mount, so it outlives a stop.
 *
 * Best-effort by design: an instance that cannot reach its bucket should still
 * serve. Losing the checkpoint costs the notice on a later restart; refusing to
 * run costs the agent. */
export function checkpoint() {
  // An EMPTY store never overwrites a non-empty checkpoint. `loadTasks` returns
  // nothing on an unreadable local file — it warns and carries on, by design —
  // and without this guard the next shutdown would write that emptiness over
  // the mount, turning a corrupt local file into permanent data loss. Nothing
  // is lost by declining: an instance with no tasks has nothing to say.
  if (tasks.size === 0 && pushConfigs.size === 0 && existsSync(CHECKPOINT_FILE)) return false;
  try { write(CHECKPOINT_FILE, CHECKPOINT_DIR, { sync: true }); return true; }
  catch (e) { console.error(`a2a | WARN checkpoint to ${CHECKPOINT_DIR} failed: ${e.message}`); return false; }
}

export function loadTasks() {
  const source = existsSync(STATE_FILE) ? "local" : existsSync(CHECKPOINT_FILE) ? "checkpoint" : "";
  if (!source) return { count: 0, source: "" };
  const file = source === "local" ? STATE_FILE : CHECKPOINT_FILE;
  try {
    const state = readState(file);
    for (const t of state.tasks) tasks.set(t.id, t);
    for (const [id, cfg] of state.push) pushConfigs.set(id, cfg);
    // Seed idleness from the newest task rather than from boot. A restart is
    // routine (7-day ceiling), and an instance that looked freshly idle after
    // every one would be swept minutes after coming back.
    const newest = [...tasks.values()]
      .map((t) => Date.parse(t?.status?.timestamp || "") || 0)
      .reduce((a, b) => Math.max(a, b), 0);
    if (newest > 0) lastActivityAt = newest;
    return { count: tasks.size, source };
  } catch (e) {
    console.error(`a2a | WARN task state at ${file} unreadable: ${e.message}`);
    return { count: 0, source: "" };
  }
}

/** Terminalise runs whose executor died with the previous container.
 *
 * Called once at boot, where the reasoning is sound by construction: this
 * process has only just started, so nothing it restored can still be running.
 * A task left `working` is not merely untidy — M5a hands the caller off with
 * "the output will appear once it finishes", and the poll has already given up.
 * Nothing else will ever speak. That is CLAUDE.md #8's silent hang, and this is
 * the only place it can be closed.
 *
 * The notice goes out through the ordinary push path, so it reaches the same
 * receiver by the same gate as a real completion. */
export function reapOrphans() {
  const orphans = [...tasks.values()].filter(
    (t) => t?.status?.state === TaskState.submitted || t?.status?.state === TaskState.working);
  // `setState` notes activity, which is right for a live transition and wrong
  // here: reaping is bookkeeping about a run that ALREADY died, not work. Left
  // alone it would overwrite the idleness `loadTasks` deliberately seeded from
  // the newest task, and every wake would hold the instance a further 30
  // minutes for a run that had already finished.
  const seeded = lastActivityAt;
  for (const t of orphans) {
    setState(t, TaskState.failed, {
      role: "agent", messageId: randomUUID(), kind: "message",
      parts: [{ kind: "text", text:
        "This run was interrupted by a restart of the resident agent and cannot be resumed. " +
        "The instance is running again — ask the question once more." }],
    });
  }
  if (orphans.length) {
    lastActivityAt = seeded;
    console.log(`a2a | reaped ${orphans.length} task(s) interrupted by a restart`);
    checkpoint();
  }
  return orphans.length;
}

/** Put a task into the store directly.
 *
 * The seam that makes restart behaviour testable: every other way of creating a
 * task runs pi, and restart behaviour is precisely what cannot be exercised
 * that way. Every M6 defect lived in the gap between "stored" and "still there
 * after a stop", which is only reachable if a test can seed the store. */
export function adoptTask(task) { tasks.set(task.id, task); persist(); return task; }

/** Drive a state transition without running pi. Same seam, same reason as
 *  `adoptTask`: which transitions reach the mount is the thing M6 got wrong
 *  live, so it has to be assertable without an agent. */
export function setStateForTest(task, state, message) { setState(task, state, message); }

export const getTask = (id) => tasks.get(id);
export const listTasks = () => [...tasks.values()];

function setState(task, state, message) {
  noteActivity();
  const was = task.status?.state;
  task.status = { state, timestamp: new Date().toISOString(), ...(message ? { message } : {}) };
  persist();
  // M8: a transition is a fact about the run, and a run outlives the request
  // that started it — `working` -> `input-required` -> `completed` all happen
  // long after the message/send span has closed. So each is its own short span
  // PARENTED to the caller's trace, which is what keeps a resident run in the
  // same chain as the coordinator and job traces instead of beside them.
  //
  // A span rather than a log line because the acceptance criterion is that
  // these are visible in the observatory, and grepping stdout is exactly what
  // this milestone exists to stop.
  const span = otel.startSpan(`a2a.task.${state}`, {
    traceparent: task.metadata?.traceparent,
    attributes: {
      "a2a.task.id": task.id,
      "a2a.task.state": state,
      "a2a.context.id": task.contextId,
      ...(task.metadata?.model ? { "a2a.model": task.metadata.model } : {}),
      ...(task.metadata?.mode ? { "a2a.mode": task.metadata.mode } : {}),
    },
  });
  if (state === TaskState.failed) {
    const why = (message?.parts || []).filter((p) => p.kind === "text").map((p) => p.text).join(" ");
    span.recordError(new Error(why || "task failed"));
  }
  span.end();
  // M6: stage the states worth surviving a stop.
  //
  // Terminal states and `input-required` are the ones a caller may wait hours
  // on, which is exactly the window a sweep falls inside. Plus the FIRST
  // transition into `working`, which is once per run and is what makes an
  // in-flight run durable from the moment it starts.
  //
  // That last one was learned live: the first cut checkpointed only terminal
  // states and SIGTERM, and a real stop mid-run recovered NOTHING. The shutdown
  // flush is a race with the container teardown (boot.sh's TERM trap now waits
  // for this process, which is the other half of the fix) — and a durability
  // story whose only writer is the shutdown path is one signal away from
  // nothing. Repeated `working` is still excluded: the interactive watcher
  // re-asserts it in a loop and a gcsfuse write per iteration would cost more
  // than the run.
  const firstWorking = state === TaskState.working && was !== TaskState.working;
  if ((state !== TaskState.working && state !== TaskState.submitted) || firstWorking) checkpoint();
  notify(task).catch((e) => console.error(`a2a | WARN push failed for ${task.id}: ${e.message}`));
}

// ─── model registry assertion ────────────────────────────────────────────────
// pi has NO max-tokens flag: for a model absent from ~/.pi/agent/models.json it
// silently falls back to maxTokens 16384 / contextWindow 128000 / reasoning
// false. Passing an unregistered model would therefore "work" while discarding
// most of its capability, so refusing is the only honest option.
export function registeredModels() {
  const cfg = JSON.parse(readFileSync(`${PI_HOME}/agent/models.json`, "utf8"));
  return Object.entries(cfg.providers || {}).flatMap(([prov, v]) =>
    (v.models || []).map((m) => ({ provider: prov, id: m.id, ref: `${prov}/${m.id}`,
      maxTokens: m.maxTokens, contextWindow: m.contextWindow })));
}
export function assertModelRegistered(model) {
  const models = registeredModels();
  const hit = models.find((m) => m.ref === model || m.id === model);
  if (!hit) {
    const known = models.map((m) => m.ref).join(", ") || "(none)";
    throw Object.assign(new Error(
      `model ${JSON.stringify(model)} is not in the pi registry, so pi would silently run it at maxTokens=16384 / contextWindow=128000 / reasoning=false. Registered: ${known}`),
      { code: -32602 });
  }
  return hit;
}

// ─── push notifications ──────────────────────────────────────────────────────
// A2A's answer for "clients unable to maintain persistent connections" — which
// is every case that matters here, since an agent may block hours after the
// caller disconnected.
const pushConfigs = new Map();
// Checkpointed as well as persisted. Registration is the moment M5a stops
// watching and starts trusting this instance to call back, so a config that
// does not survive the next stop is the promise broken silently — `notify()`
// returns early on a missing config and logs nothing.
export function setPushConfig(taskId, cfg) { pushConfigs.set(taskId, cfg); persist(); checkpoint(); }
export function getPushConfig(taskId) { return pushConfigs.get(taskId); }
async function notify(task) {
  const cfg = pushConfigs.get(task.id);
  if (!cfg?.url) return;
  const headers = { "content-type": "application/json" };
  if (cfg.token) headers.authorization = `Bearer ${cfg.token}`;
  const res = await fetch(cfg.url, { method: "POST", headers, body: JSON.stringify(task) });
  if (!res.ok) throw new Error(`webhook ${res.status}`);
  console.log(`a2a | pushed ${task.status.state} for ${task.id}`);
}

// ─── herdr <-> A2A state mapping ─────────────────────────────────────────────
// `blocked` is the one state herdr classifies POSITIVELY. `idle` and `done` are
// UI-coupled — herdr's own contract says idle requires the tab to have been
// "seen in the focused Herdr UI" and CLI reads do not mark it seen — so
// headless they are degenerate and MUST NOT be treated as proof of completion.
// Terminal detection therefore leans on the caller's staleness sweep, exactly
// as the design's Failure Modes table assumes.
function mapStatus(agentStatus) {
  switch (agentStatus) {
    case "blocked": return TaskState.inputRequired;
    case "working": return TaskState.working;
    default: return null;                     // idle/done/unknown: not conclusive
  }
}

/** Attach the agent's recent output to the task as an A2A artifact.
 *
 * Without this a task can be started but never reports anything: herdr's
 * `idle`/`done` are UI-coupled and degenerate headless (Phase 0 §3), so the
 * caller has no way to see what the agent actually produced. Reading the pane
 * is the only honest source of truth about what happened.
 */
/** Keep the task's transcript artifact current. */
function attachText(task, text) {
  if (!text) return;
  task.artifacts = [{
    artifactId: "response", name: "agent-response",
    description: "Assistant output assembled from pi's NDJSON text deltas.",
    parts: [{ kind: "text", text: String(text).slice(-16000) }],
  }];
  persist();
}

async function captureOutput(task, target) {
  try {
    const r = await herdr.agentRead({ target, lines: 200 });
    const text = r?.output ?? r?.text ?? r?.content ?? JSON.stringify(r).slice(0, 4000);
    task.artifacts = [{
      artifactId: "transcript",
      name: "agent-transcript",
      description: "Recent terminal output from the agent pane.",
      parts: [{ kind: "text", text: String(text).slice(-8000) }],
    }];
    persist();
  } catch (e) {
    console.error(`a2a | WARN could not read transcript for ${task.id}: ${e.message}`);
  }
}

async function watch(task, target) {
  try {
    const r = await herdr.agentWait({ target, until: ["blocked", "done", "idle"] });
    // AgentInfo's field is `agent_status` (schema protocol 20), not `status`.
    const status = r?.agent?.agent_status ?? r?.agent_status ?? "unknown";
    const mapped = mapStatus(status);
    await captureOutput(task, target);
    if (mapped === TaskState.inputRequired) {
      const text = await herdr.agentRead({ target, lines: 80 }).catch(() => null);
      setState(task, TaskState.inputRequired, {
        role: "agent", messageId: randomUUID(), kind: "message",
        parts: [{ kind: "text", text: text?.output ?? text?.text ?? "agent is blocked awaiting input" }],
      });
      watch(task, target);                    // keep watching after the answer
    } else {
      // Not conclusive on its own — record what herdr said and let the caller's
      // staleness sweep decide, rather than declaring success we cannot prove.
      task.metadata = { ...(task.metadata || {}), lastAgentStatus: status };
      setState(task, TaskState.working);
      watch(task, target);
    }
  } catch (e) {
    setState(task, TaskState.failed, {
      role: "agent", messageId: randomUUID(), kind: "message",
      parts: [{ kind: "text", text: `watcher error: ${e.message}` }],
    });
  }
}

// ─── message/send ────────────────────────────────────────────────────────────
export async function messageSend(params, { traceparent = "" } = {}) {
  const msg = params?.message;
  if (!msg) throw Object.assign(new Error("params.message is required"), { code: -32602 });
  const text = (msg.parts || []).filter((p) => p.kind === "text").map((p) => p.text).join("\n");

  // Follow-up on an existing task: this is how A2A answers an input-required
  // agent, so it needs no reply mechanism of our own.
  if (msg.taskId && tasks.has(msg.taskId)) {
    const task = tasks.get(msg.taskId);
    // A follow-up on the SAME task id is A2A's mechanism for answering an
    // agent that is input-required, and that only exists in interactive mode.
    // A stream task is one turn; continuing the CONVERSATION means a new
    // message carrying the same contextId. Saying so beats routing it to a
    // herdr that is not running and letting the caller wait.
    if (task.metadata?.mode === "stream") {
      throw Object.assign(new Error(
        `task ${task.id} ran in stream mode, which is a single turn. To continue this conversation send a new message with contextId ${JSON.stringify(task.contextId)}; the session resumes from it.`),
        { code: -32602 });
    }
    await herdr.agentPrompt({ target: task.metadata.agentName, text });
    setState(task, TaskState.working);
    return task;
  }

  // When the registry holds exactly ONE model there is nothing to choose, so
  // requiring every caller to name it is friction with no safety value — the
  // registry is already the allowlist. Found live 2026-09-03: the platform's
  // own client failed with "no model requested and DEFAULT_MODEL is unset",
  // which is what ask_assistant would have done in production. Ambiguity is
  // still refused: with two or more registered and none requested, the caller
  // must say which.
  const soleModel = (() => {
    try { const m = registeredModels(); return m.length === 1 ? m[0].ref : ""; }
    catch { return ""; }
  })();
  const model = params?.metadata?.model || msg.metadata?.model || DEFAULT_MODEL || soleModel;
  if (!model) {
    const known = (() => { try { return registeredModels().map((m) => m.ref).join(", "); } catch { return "(registry unreadable)"; } })();
    throw Object.assign(new Error(
      `no model requested, DEFAULT_MODEL is unset, and the registry does not hold exactly one to fall back to. Registered: ${known}`),
      { code: -32602 });
  }
  assertModelRegistered(model);

  const id = params?.metadata?.runId || msg.taskId || randomUUID();
  const contextId = msg.contextId || randomUUID();
  const agentName = `a${id.replace(/[^a-z0-9]/gi, "").slice(0, 24).toLowerCase()}`;
  const task = {
    kind: "task", id, contextId,
    status: { state: TaskState.submitted, timestamp: new Date().toISOString() },
    // `traceparent` is stored on the task, not held in a variable: the run
    // continues after this request returns, and its later transitions must
    // still land in the trace the caller started. Persisted with the task, so
    // it survives the 7-day restart too.
    history: [msg], metadata: { model, agentName, ...(traceparent ? { traceparent } : {}) },
  };
  tasks.set(id, task);
  noteActivity();
  if (params?.configuration?.pushNotificationConfig) setPushConfig(id, params.configuration.pushNotificationConfig);
  persist();

  // ── execution ───────────────────────────────────────────────────────────
  // `stream` (default) runs pi directly in NDJSON mode. `interactive` drives it
  // as a TUI under herdr, which is the mode a human can attach to but which
  // cannot currently submit a prompt headless (M9) — so it is opt-in, not the
  // default, until that is understood.
  const mode = params?.metadata?.mode || msg.metadata?.mode || DEFAULT_MODE;
  task.metadata.mode = mode;

  if (mode === "stream") {
    if (activeRuns >= MAX_CONCURRENT) {
      throw Object.assign(new Error(
        `this agent is already running ${activeRuns} of a maximum ${MAX_CONCURRENT} concurrent turns. ` +
        `Cloud Run instances do not autoscale, so retry shortly or give this caller its own instance. ` +
        `Raise RESIDENT_MAX_CONCURRENT_RUNS only with the memory headroom to match.`),
        { code: -32000 });
    }
    const sessionId = sessionIdFor(contextId);
    task.metadata.sessionId = sessionId;
    task.metadata.persistent = Boolean(capabilities().sessionFlag);
    // Fire-and-forget: the A2A call returns a task immediately and the run
    // continues in the background, reporting through task state and artifacts.
    // M8: report the run to the observatory as a SESSION with turns and tool
    // calls, which is how an agent job and a Claude Code session appear there.
    // pi's NDJSON event types and the exec API's stream types describe the same
    // thing, so the events already flowing through onEvent are simply forwarded
    // rather than re-derived. Keyed by the A2A task id so the dashboard record
    // ties back to the task and to the a2a.task.id span attribute.
    const reported = observatory.startRun({
      sessionId: id,
      workspace: process.env.WORKSPACE_DIR || "/workspace",
      provider: `pi:${model}`,
    });
    task.metadata.reported = observatory.enabled();

    onSession(sessionId, () => { activeRuns++; return runPi({
      model: model.replace(/^openrouter\//, ""),
      prompt: text,
      sessionId,
      onEvent: (ev, st) => {
        if (ev.type === "turn_start") setState(task, TaskState.working);
        // Reporting is inside the SAME try the runner already wraps this
        // callback in ("a consumer must not kill the run"), so a dashboard
        // fault cannot take the turn down with it.
        reported.event(ev);
        // Keep the artifact current as text arrives, so a caller polling
        // tasks/get sees progress rather than nothing until the end.
        if (st.text) attachText(task, st.text);
      },
    }).finally(() => { activeRuns--; }); })
      .then((r) => {
        attachText(task, r.text);
        task.metadata.usage = r.usage ?? null;
        task.metadata.stopReason = r.stopReason ?? null;
        task.metadata.toolCalls = r.toolCalls.length;
        // agent_end gives a REAL terminal state — the thing herdr could not
        // provide headless.
        setState(task, r.exitCode === 0 ? TaskState.completed : TaskState.failed, {
          role: "agent", messageId: randomUUID(), kind: "message",
          parts: [{ kind: "text", text: r.text || `pi exited ${r.exitCode}` }],
        });
        reported.finish(r.exitCode === 0
          ? { ok: true, usage: r.usage }
          : { ok: false, error: `pi exited ${r.exitCode}` }).catch(() => {});
      })
      .catch((e) => {
        setState(task, TaskState.failed, {
          role: "agent", messageId: randomUUID(), kind: "message",
          parts: [{ kind: "text", text: `pi run failed: ${e.message}` }],
        });
        // A run that fell over must still close its session, or the dashboard
        // shows it running forever — the exact state it exists to rule out.
        reported.finish({ ok: false, error: `pi run failed: ${e.message}` }).catch(() => {});
      });
    return task;
  }

  const ws = await herdr.workspaceCreate({ cwd: process.env.WORKSPACE_DIR || "/workspace", label: id.slice(0, 12) });
  const workspaceId = ws?.workspace?.workspace_id;
  if (!workspaceId) throw new Error(`workspace.create returned no workspace_id: ${JSON.stringify(ws).slice(0, 200)}`);
  const paneId = await herdr.firstPaneOf(workspaceId);
  if (!paneId) throw new Error(`no pane found in workspace ${workspaceId}`);

  await herdr.agentStart({ name: agentName, kind: AGENT_KIND, paneId, args: ["--model", model] });
  setState(task, TaskState.working);
  await herdr.waitInteractive({ target: agentName });
  herdr
    .agentPrompt({ target: agentName, text })
    .then(async () => { await captureOutput(task, agentName); watch(task, agentName); })
    .catch(async (e) => {
      await captureOutput(task, agentName).catch(() => {});
      setState(task, TaskState.failed, {
        role: "agent", messageId: randomUUID(), kind: "message",
        parts: [{ kind: "text", text: `prompt failed: ${e.message}` }],
      });
    });
  return task;
}

// ─── agent card ──────────────────────────────────────────────────────────────
export async function readTranscript(task) {
  const target = task?.metadata?.agentName;
  if (!target) return "(no agent recorded for this task)";
  const r = await herdr.agentRead({ target, lines: 200 });
  return r?.output ?? r?.text ?? r?.content ?? JSON.stringify(r).slice(0, 4000);
}

export function agentCard(baseUrl) {
  const models = (() => { try { return registeredModels().map((m) => m.ref); } catch { return []; } })();
  return {
    protocolVersion: "0.3.0",
    name: process.env.AGENT_CARD_NAME || `resident-${AGENT_KIND}`,
    description: `Resident ${AGENT_KIND} coding agent on a Cloud Run instance, supervised by herdr.`,
    url: `${baseUrl}/a2a`,
    version: process.env.AGENT_VERSION || "0.1.0",
    capabilities: {
      streaming: false, pushNotifications: true, stateTransitionHistory: true,
      // A2A's AgentCard has NO `metadata` field — the a2a-sdk parses the card
      // into a typed model and DROPS anything not in the schema, so everything
      // we published there was invisible to every spec-compliant client,
      // including our own. Found 2026-09-03 by running the platform's client
      // against this agent for the first time. `capabilities.extensions` is
      // the spec's extension point: a uri plus free-form params.
      extensions: [
        {
          uri: "https://ailang.dev/a2a/ext/resident-registry/v1",
          description: "Models this agent will run. Anything else is refused rather than silently downgraded.",
          params: { registeredModels: models, agentKind: AGENT_KIND, defaultModel: models.length === 1 ? models[0] : (DEFAULT_MODEL || null) },
        },
        {
          uri: "https://ailang.dev/a2a/ext/resident-conversation/v1",
          description: "Reuse a contextId across message/send calls to continue one conversation; a new contextId starts a fresh one.",
          params: { key: "contextId", persistent: Boolean(capabilities().sessionFlag) },
        },
      ],
    },
    defaultInputModes: ["text/plain"],
    defaultOutputModes: ["text/plain"],
    skills: [{
      id: "coding-agent", name: `${AGENT_KIND} coding agent`,
      description: "Runs a long-lived coding-agent session against a persistent workspace.",
      tags: ["code", "resident", AGENT_KIND],
      examples: ["Refactor the billing module and open a PR"],
    }],
  };
}
