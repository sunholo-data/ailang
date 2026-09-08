// Resident agent HTTP surface (v6.40.0 RESIDENT-P1 M2).
//
// Three routes, and only one of them is ours to design:
//   GET  /health                     operational, not part of any protocol
//   GET  /.well-known/agent.json     A2A discovery — same path the platform serves
//   POST /a2a                        A2A JSON-RPC 2.0
//
// There is deliberately no /panes API. The platform talks to this instance as
// an A2A peer (design Decision 2c); herdr's vocabulary stays inside the
// container instead of leaking into a bespoke platform-facing protocol.
import { createServer } from "node:http";
import * as herdr from "./lib/herdr.mjs";
import * as pi from "./lib/pi.mjs";
import * as a2a from "./lib/a2a.mjs";
import * as otel from "./lib/otel.mjs";
import * as observatory from "./lib/observatory.mjs";
import { verify, authConfig } from "./lib/auth.mjs";

const PORT = Number(process.env.RESIDENT_PORT || 8080);
const STARTED = Date.now();
const loaded = a2a.loadTasks();
// The SOURCE is logged, not just the count. "local" means the process
// restarted under a filesystem that survived; "checkpoint" means the writable
// layer was reset — a stop/resume, which is what M4's sweep does — and the
// mount is why anything came back at all. Told apart in the log because they
// were indistinguishable while the second silently returned nothing.
if (loaded.count) console.log(`server | restored ${loaded.count} task(s) from ${loaded.source} across restart`);

// M8: join the observability plane the agent jobs and the coordinator are
// already on. Unset endpoint = inert, which is the normal state of a laptop
// and of any environment that has not opted in.
otel.configure({
  endpoint: process.env.OTEL_EXPORTER_OTLP_ENDPOINT || "",
  serviceName: process.env.OTEL_SERVICE_NAME || "resident-agent",
  instance: process.env.RESIDENT_INSTANCE_NAME || "",
});
// The plane an operator actually reads: runs appear as task records with turns
// and tool calls, the same shape as an agent job and a Claude Code session.
observatory.configure({ url: process.env.AILANG_OBSERVATORY_URL || "" });
console.log(
  `server | telemetry: traces=${otel.enabled() ? "on" : "OFF"} observatory=${observatory.enabled() ? "on" : "OFF"}`,
);

const json = (res, code, body) => {
  res.writeHead(code, { "content-type": "application/json" });
  res.end(JSON.stringify(body, null, 2));
};
const rpcError = (res, id, code, message, httpCode = 200) =>
  json(res, httpCode, { jsonrpc: "2.0", id: id ?? null, error: { code, message } });

async function health() {
  const herdrEnabled = process.env.RESIDENT_ENABLE_HERDR === "1";
  let h;
  if (!herdrEnabled) h = { ok: true, disabled: true };
  else try {
    const snap = await herdr.snapshot();
    const s = snap?.snapshot ?? {};
    h = { ok: true, protocol: s.protocol, version: s.version, agents: (s.agents || []).length, panes: (s.panes || []).length };
  } catch (e) { h = { ok: false, error: String(e.message).slice(0, 200) }; }
  // herdr being off is a configuration, not a fault: it is no longer on the
  // task path, so its absence must not make the instance look unhealthy.
  let models = null;
  try { models = a2a.registeredModels().map((m) => m.ref); } catch { models = null; }
  return {
    healthy: h.ok && Array.isArray(models) && models.length > 0,
    uptime_s: Math.round((Date.now() - STARTED) / 1000),
    herdr: h,
    models: models === null ? { error: "registry unreadable" } : { count: models.length, ids: models },
    // Reported so persistence is OBSERVABLE rather than inferred. A resident
    // whose conversation silently resets looks identical to one that works
    // until someone asks it a second question, and "did it remember" is not a
    // thing an operator should have to test by hand to find out.
    // What the agent can actually DO, reported rather than implied. While
    // `bash` is in this list the AILANG program allowlist is a convenience and
    // not a containment boundary — the container and the only-dir mount are.
    tools: pi.toolPolicy(),
    // Load, so an operator can see whether the ceiling is being reached before
    // a user reports "it refused me". Instances do not autoscale, so this is
    // the whole capacity there is.
    runs: a2a.runStats(),
    extensions: process.env.RESIDENT_PI_EXTENSIONS === "1" ? "enabled" : "disabled",
    session: (() => {
      const c = pi.capabilities();
      return { persistent: Boolean(c.sessionFlag), pi: c.piVersion || null, dir: c.sessionDir || null };
    })(),
    tasks: a2a.listTasks().length,
  };
}

async function readBody(req) {
  const chunks = [];
  for await (const c of req) chunks.push(c);
  return Buffer.concat(chunks).toString();
}

const server = createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host || "localhost"}`);
  const base = process.env.PUBLIC_BASE_URL || `http://${req.headers.host || "localhost"}`;

  // /livez is the ONLY unauthenticated route: a bare liveness signal carrying
  // no information, so infrastructure probes work without a token. Everything
  // else — including the agent card, which lists registered models — requires a
  // verified caller.
  if (req.method === "GET" && url.pathname === "/livez") {
    res.writeHead(200, { "content-type": "text/plain" });
    return res.end("ok");
  }

  // Cloud Run instances (Preview) do not enforce run.invoker at the edge, so
  // this instance runs with public ingress and does the check itself. Failures
  // are uniform: never reveal whether the audience, the signature or the
  // allowlist was the problem.
  let caller;
  try {
    caller = await verify(req.headers.authorization);
  } catch (e) {
    console.error(`server | auth refused: ${e.message}`);
    res.writeHead(401, { "content-type": "application/json" });
    return res.end(JSON.stringify({ error: "unauthorized" }));
  }

  if (req.method === "GET" && (url.pathname === "/health" || url.pathname === "/")) {
    const h = await health();
    return json(res, h.healthy ? 200 : 503, h);
  }
  // Both spellings: the platform serves /.well-known/agent.json, while newer
  // A2A drafts use agent-card.json. Serving both costs nothing and avoids a
  // discovery failure that would look like the agent being down.
  if (req.method === "GET" && (url.pathname === "/.well-known/agent.json" || url.pathname === "/.well-known/agent-card.json")) {
    return json(res, 200, a2a.agentCard(base));
  }
  if (req.method !== "POST" || url.pathname !== "/a2a") {
    return json(res, 404, { error: "not found" });
  }

  let body;
  try { body = JSON.parse(await readBody(req)); }
  catch { return rpcError(res, null, -32700, "parse error"); }
  const { id, method, params } = body || {};
  if (body?.jsonrpc !== "2.0" || !method) return rpcError(res, id, -32600, "invalid request");

  // One span per A2A method, continuing the CALLER'S trace when it sent a
  // traceparent — that is what puts a resident run in the same chain as the
  // coordinator and job traces rather than in a trace of its own (M8).
  const span = otel.startSpan(`a2a.${method}`, {
    traceparent: req.headers.traceparent,
    attributes: { "rpc.system": "jsonrpc", "rpc.method": method, "peer.identity": caller?.email || "unknown" },
  });
  // tasks/* address an existing task by id; message/send MAKES one, so its id
  // is only known from the result and is attached there.
  const inboundTaskId = params?.id ?? params?.taskId;
  if (inboundTaskId) span.setAttribute("a2a.task.id", inboundTaskId);

  try {
    switch (method) {
      case "message/send": {
        // The caller's trace context is threaded into the TASK, not just this
        // span: the run's later transitions happen after this request returns.
        const result = await a2a.messageSend(params, { traceparent: span.traceparent() });
        if (result?.id) span.setAttribute("a2a.task.id", result.id);
        if (result?.status?.state) span.setAttribute("a2a.task.state", result.status.state);
        return json(res, 200, { jsonrpc: "2.0", id, result });
      }
      case "tasks/get": {
        const t = a2a.getTask(params?.id);
        return t ? json(res, 200, { jsonrpc: "2.0", id, result: t })
                 : rpcError(res, id, -32001, `task not found: ${params?.id}`);
      }
      case "tasks/transcript": {
        // Not an A2A method: an operator affordance for a Preview product where
        // completion cannot be detected reliably. Namespaced so it is obvious
        // it is ours, and it reads the pane rather than inventing state.
        const t = a2a.getTask(params?.id);
        if (!t) return rpcError(res, id, -32001, `task not found: ${params?.id}`);
        const out = await a2a.readTranscript(t);
        return json(res, 200, { jsonrpc: "2.0", id, result: { taskId: t.id, transcript: out } });
      }
      case "tasks/list":
        return json(res, 200, { jsonrpc: "2.0", id, result: { tasks: a2a.listTasks() } });
      case "tasks/pushNotificationConfig/set":
        a2a.setPushConfig(params?.taskId, params?.pushNotificationConfig);
        return json(res, 200, { jsonrpc: "2.0", id, result: { taskId: params?.taskId, pushNotificationConfig: params?.pushNotificationConfig } });
      case "tasks/pushNotificationConfig/get":
        return json(res, 200, { jsonrpc: "2.0", id, result: a2a.getPushConfig(params?.taskId) ?? null });
      default:
        return rpcError(res, id, -32601, `method not found: ${method}`);
    }
  } catch (e) {
    // A herdr socket failure must surface as a visible JSON-RPC error, never a
    // hang and never a success with an empty result.
    console.error(`server | ${method} failed: ${e.stack || e.message}`);
    span.recordError(e);
    return rpcError(res, id, e.code ?? -32603, e.message);
  } finally {
    span.end();
  }
});

const cfg = authConfig();
server.listen(PORT, "0.0.0.0", () => {
  console.log(
    `server | listening on ${PORT} (livez public; health, A2A card and JSON-RPC require OIDC) ` +
      `| audience=${cfg.audience || "UNSET"} allowed=${cfg.allowed.length || "UNSET"}`,
  );
  // M6: close out runs that died with the previous container. AFTER listen, so
  // an instance woken by a caller is already answering while it does this —
  // the reap posts outbound webhooks and must never delay readiness.
  a2a.reapOrphans();
});

// M6: Cloud Run sends SIGTERM and gives roughly ten seconds before the
// writable layer goes. That window is the last chance to stage anything a
// caller is still waiting on. `checkpoint()` is best-effort and synchronous, so
// a failure here costs the notice and never the shutdown.
for (const sig of ["SIGTERM", "SIGINT"]) {
  process.on(sig, () => {
    console.log(`server | ${sig} — staging task state before shutdown`);
    a2a.checkpoint();
    server.close(() => process.exit(0));
    // A connection that will not drain must not hold the instance past the
    // window; the checkpoint is already written by this point either way.
    setTimeout(() => process.exit(0), 5000).unref();
  });
}
