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
import { readFileSync, writeFileSync, mkdirSync, existsSync } from "node:fs";
import { randomUUID } from "node:crypto";
import * as herdr from "./herdr.mjs";
import { runPi, capabilities } from "./pi.mjs";

const PI_HOME = process.env.PI_HOME || "/home/ailang/.pi";
const STATE_DIR = process.env.TASK_STATE_DIR || "/home/ailang/.resident";
const STATE_FILE = `${STATE_DIR}/tasks.json`;
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

// ─── task store ──────────────────────────────────────────────────────────────
// Persisted to local disk so tasks survive the weekly auto-restart. NOT to
// $AGENT_HOME: gcsfuse has no POSIX locking and this file is rewritten often.
let tasks = new Map();
function persist() {
  try {
    mkdirSync(STATE_DIR, { recursive: true });
    writeFileSync(STATE_FILE, JSON.stringify([...tasks.values()], null, 2));
  } catch (e) { console.error(`a2a | WARN persist failed: ${e.message}`); }
}
export function loadTasks() {
  if (!existsSync(STATE_FILE)) return 0;
  try {
    for (const t of JSON.parse(readFileSync(STATE_FILE, "utf8"))) tasks.set(t.id, t);
    return tasks.size;
  } catch (e) { console.error(`a2a | WARN task state unreadable: ${e.message}`); return 0; }
}
export const getTask = (id) => tasks.get(id);
export const listTasks = () => [...tasks.values()];

function setState(task, state, message) {
  task.status = { state, timestamp: new Date().toISOString(), ...(message ? { message } : {}) };
  persist();
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
export function setPushConfig(taskId, cfg) { pushConfigs.set(taskId, cfg); persist(); }
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
export async function messageSend(params) {
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
    history: [msg], metadata: { model, agentName },
  };
  tasks.set(id, task);
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
    const sessionId = sessionIdFor(contextId);
    task.metadata.sessionId = sessionId;
    task.metadata.persistent = Boolean(capabilities().sessionFlag);
    // Fire-and-forget: the A2A call returns a task immediately and the run
    // continues in the background, reporting through task state and artifacts.
    onSession(sessionId, () => runPi({
      model: model.replace(/^openrouter\//, ""),
      prompt: text,
      sessionId,
      onEvent: (ev, st) => {
        if (ev.type === "turn_start") setState(task, TaskState.working);
        // Keep the artifact current as text arrives, so a caller polling
        // tasks/get sees progress rather than nothing until the end.
        if (st.text) attachText(task, st.text);
      },
    }))
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
      })
      .catch((e) => {
        setState(task, TaskState.failed, {
          role: "agent", messageId: randomUUID(), kind: "message",
          parts: [{ kind: "text", text: `pi run failed: ${e.message}` }],
        });
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
