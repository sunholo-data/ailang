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

const PI_HOME = process.env.PI_HOME || "/home/ailang/.pi";
const STATE_DIR = process.env.TASK_STATE_DIR || "/home/ailang/.resident";
const STATE_FILE = `${STATE_DIR}/tasks.json`;
const AGENT_KIND = process.env.AGENT_KIND || "pi";
const DEFAULT_MODEL = process.env.DEFAULT_MODEL || "";

export const TaskState = {
  submitted: "submitted", working: "working", inputRequired: "input-required",
  completed: "completed", failed: "failed", canceled: "canceled", rejected: "rejected",
};

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

async function watch(task, target) {
  try {
    const r = await herdr.agentWait({ target, until: ["blocked", "done", "idle"] });
    // AgentInfo's field is `agent_status` (schema protocol 20), not `status`.
    const status = r?.agent?.agent_status ?? r?.agent_status ?? "unknown";
    const mapped = mapStatus(status);
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
    await herdr.agentPrompt({ target: task.metadata.agentName, text });
    setState(task, TaskState.working);
    return task;
  }

  const model = params?.metadata?.model || msg.metadata?.model || DEFAULT_MODEL;
  if (!model) throw Object.assign(new Error("no model requested and DEFAULT_MODEL is unset"), { code: -32602 });
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

  const ws = await herdr.workspaceCreate({ cwd: process.env.WORKSPACE_DIR || "/workspace", label: id.slice(0, 12) });
  const workspaceId = ws?.workspace?.workspace_id;
  if (!workspaceId) throw new Error(`workspace.create returned no workspace_id: ${JSON.stringify(ws).slice(0, 200)}`);
  const paneId = await herdr.firstPaneOf(workspaceId);
  if (!paneId) throw new Error(`no pane found in workspace ${workspaceId}`);

  await herdr.agentStart({ name: agentName, kind: AGENT_KIND, paneId, args: ["--model", model] });
  setState(task, TaskState.working);
  // Wait for the CLI to be able to take input: agent.start returns as soon as
  // the process is launched, not when it is ready.
  await herdr.waitInteractive({ target: agentName });
  await herdr.agentPrompt({ target: agentName, text });
  watch(task, agentName);
  return task;
}

// ─── agent card ──────────────────────────────────────────────────────────────
export function agentCard(baseUrl) {
  const models = (() => { try { return registeredModels().map((m) => m.ref); } catch { return []; } })();
  return {
    protocolVersion: "0.3.0",
    name: process.env.AGENT_CARD_NAME || `resident-${AGENT_KIND}`,
    description: `Resident ${AGENT_KIND} coding agent on a Cloud Run instance, supervised by herdr.`,
    url: `${baseUrl}/a2a`,
    version: process.env.AGENT_VERSION || "0.1.0",
    capabilities: { streaming: false, pushNotifications: true, stateTransitionHistory: true },
    defaultInputModes: ["text/plain"],
    defaultOutputModes: ["text/plain"],
    skills: [{
      id: "coding-agent", name: `${AGENT_KIND} coding agent`,
      description: "Runs a long-lived coding-agent session against a persistent workspace.",
      tags: ["code", "resident", AGENT_KIND],
      examples: ["Refactor the billing module and open a PR"],
    }],
    metadata: { registeredModels: models, agentKind: AGENT_KIND },
  };
}
