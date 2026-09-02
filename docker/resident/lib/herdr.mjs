// herdr socket-API client (v6.40.0 RESIDENT-P1 M2).
//
// Speaks the socket API directly rather than shelling out to the CLI: the
// bundled schema (`herdr api schema --json`) is authoritative for method names
// and parameters, where CLI flag spellings would be guesswork. Framing is
// newline-delimited JSON — {id, method, params} in, {id, result|error} out.
//
// Method and enum names below come from schema protocol 20, schema_version 1.
import net from "node:net";
import { randomUUID } from "node:crypto";

const SOCKET = () => process.env.HERDR_SOCKET_PATH || `${process.env.HOME}/.config/herdr/herdr.sock`;

// herdr's own lifecycle vocabulary (AgentStatus in the schema).
export const AGENT_STATUS = ["idle", "working", "blocked", "done", "unknown"];

export function call(method, params = {}, { timeoutMs = 15000 } = {}) {
  return new Promise((resolve, reject) => {
    const id = `resident:${randomUUID()}`;
    const c = net.createConnection(SOCKET());
    let buf = "";
    const done = (fn, arg) => { clearTimeout(t); c.destroy(); fn(arg); };
    const t = setTimeout(() => done(reject, new Error(`herdr ${method} timed out after ${timeoutMs}ms`)), timeoutMs);
    c.on("connect", () => c.write(JSON.stringify({ id, method, params }) + "\n"));
    c.on("error", (e) => done(reject, new Error(`herdr socket: ${e.message}`)));
    c.on("data", (d) => {
      buf += d.toString();
      let i;
      while ((i = buf.indexOf("\n")) >= 0) {
        const line = buf.slice(0, i); buf = buf.slice(i + 1);
        if (!line.trim()) continue;
        let msg;
        try { msg = JSON.parse(line); } catch { continue; }
        if (msg.id !== id) continue;             // ignore unrelated events on the stream
        if (msg.error) return done(reject, new Error(`herdr ${method}: ${msg.error.message || JSON.stringify(msg.error)}`));
        return done(resolve, msg.result);
      }
    });
  });
}

// Liveness. `api snapshot` is the ONLY honest probe: `herdr status` exits 0
// whether or not the server runs, and list calls return successful EMPTY
// results with no server, indistinguishable from a healthy idle one.
export async function snapshot() { return call("session.snapshot", {}, { timeoutMs: 5000 }); }

export async function workspaceCreate({ cwd, label, env } = {}) {
  return call("workspace.create", { cwd: cwd ?? null, label: label ?? null, env: env ?? {}, focus: false });
}

// agent.start requires an EXISTING pane; it never creates or moves layout.
export async function agentStart({ name, kind, paneId, args = [], timeoutMs = 30000 }) {
  return call("agent.start", { name, kind, pane_id: paneId, args, timeout_ms: timeoutMs });
}

// agent.prompt WITHOUT a `wait` object does not reliably submit: observed live
// 2026-09-02 with the prompt sitting in pi's input box and context stuck at
// 0.0%, so no model call was ever made and the task hung at `working`.
//
// The socket-api docs are explicit: "agent.prompt accepts an optional `wait`
// object with `until` and `timeout_ms`; this submits the prompt and starts the
// wait in one request, avoiding a race between separate calls." So the wait
// object is what makes it a submission, and it removes a race we would
// otherwise have to handle ourselves.
//
// The same paragraph documents a refusal we MUST NOT swallow: "If the resolved
// agent is already blocked, agent.prompt returns agent_blocked without sending
// input." Ignoring that is how a prompt silently vanishes.
export async function agentPrompt({ target, text, until = ["idle", "blocked", "done"], timeoutMs = 600000 }) {
  const r = await call(
    "agent.prompt",
    { target, text, wait: { until, timeout_ms: timeoutMs } },
    { timeoutMs: timeoutMs + 10000 },
  );
  const type = r?.type ?? r?.result?.type;
  if (type === "agent_blocked" || r?.agent_blocked) {
    throw Object.assign(
      new Error(`agent ${target} was already blocked; herdr refused to send the prompt (agent_blocked)`),
      { agentBlocked: true },
    );
  }
  return r;
}

// agent.start RETURNS BEFORE THE AGENT CAN ACCEPT INPUT. Prompting straight
// after it fails with "not an active named agent" — observed live 2026-09-02 on
// the first real prompt, with herdr already reporting agents:1. AgentInfo
// carries `interactive_ready` and `launch_pending` for exactly this, so wait on
// them rather than sleeping a guessed interval.
export async function waitInteractive({ target, timeoutMs = 90000 }) {
  const deadline = Date.now() + timeoutMs;
  let last = null;
  while (Date.now() < deadline) {
    try {
      const r = await agentGet({ target });
      const a = r?.agent ?? r;
      last = { ready: a?.interactive_ready, pending: a?.launch_pending, status: a?.agent_status };
      if (a?.interactive_ready === true && a?.launch_pending !== true) return last;
    } catch (e) {
      last = { error: e.message };          // the agent may not be listed yet
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error(`agent ${target} never became interactive within ${timeoutMs}ms (last: ${JSON.stringify(last)})`);
}

// Blocks server-side until the agent reaches one of `until`. This is why the
// watcher needs no polling loop.
export async function agentWait({ target, until, timeoutMs = 3600000 }) {
  return call("agent.wait", { target, until, timeout_ms: timeoutMs }, { timeoutMs: timeoutMs + 5000 });
}

export async function agentGet({ target }) { return call("agent.get", { target }); }

export async function paneList() { return call("pane.list", {}); }

// WorkspaceInfo carries active_tab_id but NO pane id, and neither does
// TabInfo (schema protocol 20) — so the pane a new workspace opened with is
// found by listing panes and filtering on workspace_id, not read off the
// create response.
export async function firstPaneOf(workspaceId) {
  const r = await paneList();
  const pane = (r?.panes || []).find((p) => p.workspace_id === workspaceId);
  return pane?.pane_id ?? null;
}

export async function agentRead({ target, source = "recent_unwrapped", lines = 400, format = "text" }) {
  return call("agent.read", { target, source, lines, format, strip_ansi: true });
}
