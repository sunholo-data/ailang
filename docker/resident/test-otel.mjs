// Tracer acceptance tests (v6.40.0 RESIDENT-P1 M8).
//
// Runs BOTH locally against the checkout and inside the built image, because
// the two answer different questions and only one of them is cheap:
//
//     node --test docker/resident/test-otel.mjs                 # laptop
//     RESIDENT_LIB=/usr/local/bin/lib node --test test-otel.mjs # in-image
//
// Every assertion here is a way the resident could join the observability
// plane and still tell you nothing — or, worse, take a working agent down for
// a reason unrelated to its job. Telemetry is the one subsystem whose failure
// must never be the user's problem, so most of these are about what happens
// when it is broken, absent or unreachable.
import { after, before, describe, it } from "node:test";
import assert from "node:assert/strict";
import { createServer } from "node:http";

const LIB = process.env.RESIDENT_LIB || new URL("./lib", import.meta.url).pathname;
const otelPath = `${LIB}/otel.mjs`;

/** A stand-in observatory. Returns every payload it was POSTed. */
function stubCollector() {
  const received = [];
  const server = createServer((req, res) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      received.push({ url: req.url, headers: req.headers, body: Buffer.concat(chunks).toString() });
      res.writeHead(200, { "content-type": "application/json" });
      res.end("{}");
    });
  });
  return {
    received,
    listen: () => new Promise((r) => server.listen(0, "127.0.0.1", () => r(`http://127.0.0.1:${server.address().port}`))),
    close: () => new Promise((r) => server.close(r)),
  };
}

const spansOf = (received) =>
  received.flatMap((r) => JSON.parse(r.body).resourceSpans.flatMap((rs) => rs.scopeSpans.flatMap((ss) => ss.spans)));
const attr = (span, key) => span.attributes?.find((a) => a.key === key)?.value;

describe("resident tracer", () => {
  let collector;
  let endpoint;

  before(async () => {
    collector = stubCollector();
    endpoint = await collector.listen();
  });
  after(async () => collector && collector.close());

  // The default state of every environment that has not opted in, including a
  // developer's laptop and every test above this one. A tracer that throws, or
  // that blocks on a connection nobody configured, would make telemetry a
  // reason the agent stops answering.
  it("is inert, and silent, when no endpoint is configured", async () => {
    const otel = await import(`${otelPath}?case=unconfigured`);
    otel.configure({ endpoint: "", serviceName: "resident-test" });
    assert.equal(otel.enabled(), false);
    const span = otel.startSpan("a2a.message/send");
    span.setAttribute("a2a.task.id", "t-1");
    span.end();
    await otel.flush();
  });

  it("exports one OTLP/HTTP JSON span per operation", async () => {
    const otel = await import(`${otelPath}?case=export`);
    otel.configure({ endpoint, serviceName: "resident-test" });
    assert.equal(otel.enabled(), true);
    otel.startSpan("a2a.message/send").end();
    await otel.flush();

    assert.ok(collector.received.length > 0, "the collector was never POSTed");
    const last = collector.received.at(-1);
    // The observatory serves POST /v1/traces and accepts protobuf OR JSON
    // (internal/observatory/otlp_receiver.go). JSON is what lets this be
    // dependency-free — see the header of lib/otel.mjs.
    assert.equal(last.url, "/v1/traces");
    assert.match(last.headers["content-type"], /application\/json/);

    const span = spansOf([last])[0];
    assert.equal(span.name, "a2a.message/send");
    assert.match(span.traceId, /^[0-9a-f]{32}$/, "traceId must be 32 hex chars");
    assert.match(span.spanId, /^[0-9a-f]{16}$/, "spanId must be 16 hex chars");
    assert.ok(Number(span.endTimeUnixNano) >= Number(span.startTimeUnixNano));
  });

  it("carries the A2A task id as an attribute", async () => {
    const otel = await import(`${otelPath}?case=taskid`);
    otel.configure({ endpoint, serviceName: "resident-test" });
    const span = otel.startSpan("a2a.tasks/get");
    span.setAttribute("a2a.task.id", "task-abc");
    span.end();
    await otel.flush();
    const span0 = spansOf([collector.received.at(-1)])[0];
    // The correlation key. Without it a resident run is a span in a trace of
    // its own, sitting BESIDE the coordinator and job traces rather than in
    // the same chain — which is the whole point of the milestone.
    assert.deepEqual(attr(span0, "a2a.task.id"), { stringValue: "task-abc" });
  });

  it("continues an inbound trace rather than starting a new one", async () => {
    const otel = await import(`${otelPath}?case=traceparent`);
    otel.configure({ endpoint, serviceName: "resident-test" });
    const traceId = "4bf92f3577b34da6a3ce929d0e0e4736";
    const parentId = "00f067aa0ba902b7";
    otel.startSpan("a2a.message/send", { traceparent: `00-${traceId}-${parentId}-01` }).end();
    await otel.flush();
    const span = spansOf([collector.received.at(-1)])[0];
    assert.equal(span.traceId, traceId, "a new traceId breaks the chain the caller started");
    assert.equal(span.parentSpanId, parentId);
  });

  it("ignores a malformed traceparent instead of failing the call", async () => {
    const otel = await import(`${otelPath}?case=badparent`);
    otel.configure({ endpoint, serviceName: "resident-test" });
    otel.startSpan("a2a.message/send", { traceparent: "not-a-traceparent" }).end();
    await otel.flush();
    const span = spansOf([collector.received.at(-1)])[0];
    assert.match(span.traceId, /^[0-9a-f]{32}$/);
    assert.ok(!span.parentSpanId || span.parentSpanId === "", "a bad parent must not be echoed as one");
  });

  it("identifies WHICH resident produced the span", async () => {
    const otel = await import(`${otelPath}?case=resource`);
    otel.configure({ endpoint, serviceName: "resident-agent", instance: "resident-abc123" });
    otel.startSpan("a2a.tasks/list").end();
    await otel.flush();
    const rs = JSON.parse(collector.received.at(-1).body).resourceSpans[0];
    const ra = (k) => rs.resource.attributes.find((a) => a.key === k)?.value?.stringValue;
    // One dashboard, many producers. A span that cannot say which box it came
    // from is indistinguishable from every other resident's.
    assert.equal(ra("service.name"), "resident-agent");
    assert.equal(ra("service.instance.id"), "resident-abc123");
  });

  it("records a failure as a failure, not an absence", async () => {
    const otel = await import(`${otelPath}?case=error`);
    otel.configure({ endpoint, serviceName: "resident-test" });
    const span = otel.startSpan("a2a.message/send");
    span.recordError(new Error("herdr socket closed"));
    span.end();
    await otel.flush();
    const span0 = spansOf([collector.received.at(-1)])[0];
    assert.equal(span0.status?.code, 2, "OTLP STATUS_CODE_ERROR");
    assert.match(span0.status?.message ?? "", /herdr socket closed/);
  });

  it("survives a collector that is not there, and says so ONCE", async () => {
    const otel = await import(`${otelPath}?case=unreachable`);
    // Port 1 is reserved and never listening — a stand-in for the dashboard
    // being down, which must never be the reason a user's question fails.
    otel.configure({ endpoint: "http://127.0.0.1:1", serviceName: "resident-test" });
    const warnings = [];
    const realWarn = console.warn;
    console.warn = (...a) => warnings.push(a.join(" "));
    try {
      for (let i = 0; i < 3; i++) otel.startSpan("a2a.message/send").end();
      await otel.flush();
      await otel.flush();
    } finally {
      console.warn = realWarn;
    }
    // Loud once is a diagnosis; loud every span is a second outage in the logs
    // of the box you are trying to read to diagnose the first.
    assert.ok(warnings.length <= 1, `expected at most one warning, got ${warnings.length}`);
  });
});
