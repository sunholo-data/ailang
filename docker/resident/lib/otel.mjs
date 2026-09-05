// OTLP tracing for the resident agent (v6.40.0 RESIDENT-P1 M8).
//
// WHY THIS IS HAND-WRITTEN AND NOT @opentelemetry/sdk-node
//
// The observatory's receiver accepts protobuf OR JSON on POST /v1/traces
// (internal/observatory/otlp_receiver.go), and OTLP/HTTP+JSON is a flat,
// stable, fully specified document shape. Emitting it directly costs ~150
// lines and no dependencies; the SDK costs a node_modules tree inside an image
// whose Dockerfile pins herdr by checksum and pi by version precisely because
// it does not accept an unpinned supply chain. Every other file in this
// container is dependency-free plain .mjs, and there is no package.json to
// hang a lockfile on.
//
// The trade is real and worth naming: no auto-instrumentation, no metrics, no
// context propagation beyond the one header we parse. This module deliberately
// does the smallest thing that puts a resident run in the same chain as the
// coordinator and job traces, which is what M8 asked for.
//
// TELEMETRY MUST NEVER BE THE REASON A TURN FAILS.
//
// That is the rule the whole file is shaped around. An unset endpoint is the
// normal state of a laptop and of any environment that has not opted in, so it
// is inert rather than an error. An unreachable collector is a dashboard
// outage, not an agent outage, so spans are dropped and the fact is logged
// ONCE — logging it per span turns someone else's outage into a flood in the
// logs you are reading to diagnose it.

const NS_PER_MS = 1_000_000n;

let cfg = { endpoint: "", serviceName: "resident-agent", instance: "" };
let queue = [];
let warned = false;
let flushTimer = null;

const hex = (bytes) => {
  const b = new Uint8Array(bytes);
  crypto.getRandomValues(b);
  return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
};

const nowNano = () => BigInt(Date.now()) * NS_PER_MS;

/** OTLP wants a typed value object, not a bare scalar. */
const anyValue = (v) => {
  if (typeof v === "number") return Number.isInteger(v) ? { intValue: String(v) } : { doubleValue: v };
  if (typeof v === "boolean") return { boolValue: v };
  return { stringValue: String(v) };
};
const kv = (obj) => Object.entries(obj).map(([key, value]) => ({ key, value: anyValue(value) }));

// W3C traceparent: version-traceId-spanId-flags, all lowercase hex, fixed
// widths. Anything else is ignored rather than repaired: echoing a parent id
// we could not parse would put this span in a trace that does not exist, which
// is worse than starting a clean one.
const TRACEPARENT = /^00-([0-9a-f]{32})-([0-9a-f]{16})-[0-9a-f]{2}$/;
function parseTraceparent(header) {
  const m = TRACEPARENT.exec(String(header || "").trim());
  return m ? { traceId: m[1], parentSpanId: m[2] } : null;
}

/**
 * Point the tracer at a collector. Called once at startup; calling it with an
 * empty endpoint is how a deployment opts out, and is not an error.
 */
export function configure({ endpoint = "", serviceName = "resident-agent", instance = "" } = {}) {
  cfg = { endpoint: String(endpoint || "").replace(/\/+$/, ""), serviceName, instance };
  queue = [];
  warned = false;
  return enabled();
}

export function enabled() {
  return cfg.endpoint !== "";
}

/** A span that costs nothing when telemetry is off, so callers never branch. */
const INERT = {
  setAttribute() {
    return this;
  },
  addEvent() {
    return this;
  },
  recordError() {
    return this;
  },
  end() {},
  traceparent: () => "",
};

/**
 * Start a span. `traceparent` continues an inbound trace — that is what makes
 * a resident run appear IN the caller's chain rather than beside it.
 */
export function startSpan(name, { traceparent = "", attributes = {}, kind = 2 } = {}) {
  if (!enabled()) return INERT;

  const parent = parseTraceparent(traceparent);
  const traceId = parent?.traceId ?? hex(16);
  const spanId = hex(8);
  const span = {
    traceId,
    spanId,
    ...(parent ? { parentSpanId: parent.parentSpanId } : {}),
    name,
    kind,
    startTimeUnixNano: String(nowNano()),
    attributes: kv(attributes),
    events: [],
  };

  return {
    setAttribute(key, value) {
      if (value !== undefined && value !== null) span.attributes.push({ key, value: anyValue(value) });
      return this;
    },
    // Lifecycle transitions (working, blocked, failed) are events on the run's
    // span, not separate spans: they are moments in one run, and modelling them
    // as spans would imply a duration they do not have.
    addEvent(eventName, attrs = {}) {
      span.events.push({ name: eventName, timeUnixNano: String(nowNano()), attributes: kv(attrs) });
      return this;
    },
    recordError(err) {
      const message = String(err?.message ?? err);
      span.status = { code: 2, message }; // STATUS_CODE_ERROR
      span.events.push({
        name: "exception",
        timeUnixNano: String(nowNano()),
        attributes: kv({ "exception.message": message, "exception.type": err?.name || "Error" }),
      });
      return this;
    },
    end() {
      span.endTimeUnixNano = String(nowNano());
      queue.push(span);
      scheduleFlush();
    },
    /** Pass downstream so a call this span makes stays in the same trace. */
    traceparent: () => `00-${traceId}-${spanId}-01`,
  };
}

// Batched rather than one request per span: a turn produces several spans and
// three POSTs on the request path would add latency to the thing being
// measured. `unref` so a pending flush never holds the process open — an
// instance that will not exit is how a restart becomes a stuck restart.
function scheduleFlush() {
  if (flushTimer || !enabled()) return;
  flushTimer = setTimeout(() => {
    flushTimer = null;
    flush().catch(() => {});
  }, 2000);
  if (typeof flushTimer.unref === "function") flushTimer.unref();
}

/** Send everything queued. Never throws, never rejects. */
export async function flush() {
  if (!enabled() || queue.length === 0) return;
  const spans = queue;
  queue = [];

  const payload = {
    resourceSpans: [
      {
        resource: {
          attributes: kv({
            "service.name": cfg.serviceName,
            ...(cfg.instance ? { "service.instance.id": cfg.instance } : {}),
          }),
        },
        scopeSpans: [{ scope: { name: "resident-agent" }, spans }],
      },
    ],
  };

  try {
    const res = await fetch(`${cfg.endpoint}/v1/traces`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        // Accepted by the receiver alongside its own header, and the form
        // every OTLP exporter uses. Absent when ingest auth is off, which is
        // the deployed default.
        ...(process.env.AILANG_OTLP_INGEST_TOKEN
          ? { "X-AILANG-Ingest-Token": process.env.AILANG_OTLP_INGEST_TOKEN }
          : {}),
      },
      body: JSON.stringify(payload),
      signal: AbortSignal.timeout(3000), // matches internal/telemetry/otel.go
    });
    if (!res.ok) warnOnce(`observatory returned ${res.status} for ${spans.length} span(s)`);
  } catch (e) {
    warnOnce(`traces not exported (${String(e.message).slice(0, 120)}) — telemetry is degraded, the agent is not`);
  }
}

function warnOnce(message) {
  if (warned) return;
  warned = true;
  console.warn(`otel | ${message}`);
}
