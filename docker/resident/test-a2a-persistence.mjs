// Task-store durability across the stop the sweep performs (v6.40.0 M6).
//
//     node --test docker/resident/test-a2a-persistence.mjs                    # laptop
//     RESIDENT_LIB=/usr/local/bin/lib node --test test-a2a-persistence.mjs    # in-image
//
// WHY THIS EXISTS
//
// M4's idle sweep stops an instance after 30 minutes; M5a promises the caller
// "the output will appear in this conversation once it finishes — you do not
// need to ask again". Those two facts meet on this file, and before M6 they
// contradicted each other three ways, none of which any unit test could see
// because all three need a RESTART to appear:
//
//   1. The task store lived only on local disk. Cloud Run documents the
//      writable layer as reset on stop/resume, and the live logs agreed:
//      resident-7e71c9b736f2 booted three times on 2026-09-04 and
//      "restored N task(s) across restart" was logged NONE of them — including
//      the boot straight after the M5a proof run.
//   2. `pushConfigs` was never written at all. `setPushConfig` called
//      `persist()`, which serialised `tasks` and not the config it had just
//      stored — so the call read as durability and provided none, and
//      `notify()` returns early on a missing config, so losing every webhook
//      logged nothing.
//   3. Nothing reaped a restored `working` task. Its runner died with the
//      container, so it stayed `working` for ever and the caller was never
//      told — CLAUDE.md #8, the silent hang.
//
// Each test below fails on the pre-M6 code for its own reason. They assert on
// the RELOAD, never on the return of the call that wrote, because "it was
// stored" is exactly the thing that was true before and still lost the data.
import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdtempSync, rmSync, writeFileSync, mkdirSync, readFileSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const LIB = process.env.RESIDENT_LIB || new URL("./lib", import.meta.url).pathname;

let roots = [];

/** Load a FRESH copy of a2a.mjs bound to its own state dir and agent home.
 *
 * The module reads both paths at import time, and Node caches by URL, so a
 * cache-busting query is the only way to simulate the thing under test: a
 * process that has died and come back to the same directories. */
async function bootResident({ stateDir, agentHome }) {
  process.env.TASK_STATE_DIR = stateDir;
  process.env.AGENT_HOME = agentHome;
  return import(`${LIB}/a2a.mjs?boot=${randomTag()}`);
}

let counter = 0;
const randomTag = () => `${Date.now()}-${counter++}`;

function newRoots() {
  const root = mkdtempSync(join(tmpdir(), "resident-m6-"));
  roots.push(root);
  const stateDir = join(root, "local");
  const agentHome = join(root, "agent-home");
  mkdirSync(stateDir, { recursive: true });
  mkdirSync(agentHome, { recursive: true });
  return { root, stateDir, agentHome };
}

/** Simulate what Cloud Run does on stop/resume: the writable layer is gone,
 *  the mounted bucket is not. */
function stopResume(stateDir) {
  rmSync(stateDir, { recursive: true, force: true });
  mkdirSync(stateDir, { recursive: true });
}

function workingTask(id, contextId = "ctx-1") {
  return {
    kind: "task", id, contextId,
    status: { state: "working", timestamp: new Date().toISOString() },
    history: [], metadata: { model: "z-ai/glm-5.3-flash", agentName: `a${id}` },
  };
}

let openServers = [];

function webhookStub() {
  const posts = [];
  const server = createServer((req, res) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      let body = {};
      try { body = JSON.parse(Buffer.concat(chunks).toString()); } catch {}
      posts.push({ auth: req.headers.authorization || "", body });
      res.writeHead(200, { "content-type": "application/json" });
      res.end("{}");
    });
  });
  openServers.push(server);
  return {
    posts,
    listen: () => new Promise((r) => server.listen(0, "127.0.0.1", () => r(`http://127.0.0.1:${server.address().port}`))),
    close: () => new Promise((r) => server.close(r)),
  };
}

const settle = () => new Promise((r) => setTimeout(r, 120));

// A stub whose test threw before its own close() would hold the event loop and
// hang the whole FILE, which reads as a timeout rather than as the assertion
// that actually failed. Cleanup is therefore unconditional.
afterEach(async () => {
  for (const s of openServers) await new Promise((r) => s.close(r));
  openServers = [];
  for (const r of roots) rmSync(r, { recursive: true, force: true });
  roots = [];
});

describe("task state survives the stop the sweep performs", () => {
  it("restores tasks from the agent-home checkpoint when local disk was reset", async () => {
    const { stateDir, agentHome } = newRoots();

    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-sweep-1"));
    first.checkpoint();

    // The sweep stops the instance. Local disk goes; the bucket stays.
    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    const loaded = second.loadTasks();
    assert.equal(loaded.count, 1, "the checkpoint should have carried the task across the stop");
    assert.equal(loaded.source, "checkpoint", "and it should say where it came from");
    assert.ok(second.getTask("task-sweep-1"), "task-sweep-1 should be back");
  });

  it("prefers local disk when both exist, because it is the newer of the two", async () => {
    const { stateDir, agentHome } = newRoots();

    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-old"));
    first.checkpoint();
    first.adoptTask(workingTask("task-new"));   // persisted locally, not checkpointed

    const second = await bootResident({ stateDir, agentHome });
    const loaded = second.loadTasks();
    assert.equal(loaded.source, "local");
    assert.ok(second.getTask("task-new"), "the local file is ahead of the checkpoint and wins");
  });

  it("checkpoints a run the moment it starts working, not only at shutdown", async () => {
    // Learned live, and the most important test here. The first cut staged only
    // terminal states and SIGTERM; a real Cloud Run stop mid-run then recovered
    // NOTHING, because PID 1 (boot.sh) exited before the server's handler could
    // run. A durability story whose only writer is the shutdown path is one
    // missed signal away from no durability at all.
    const { stateDir, agentHome } = newRoots();
    const mod = await bootResident({ stateDir, agentHome });
    const t = mod.adoptTask({
      kind: "task", id: "task-inflight", contextId: "ctx-i",
      status: { state: "submitted", timestamp: new Date().toISOString() },
      history: [], metadata: {},
    });
    mod.setStateForTest(t, "working");

    // Nothing else has happened: no terminal state, no SIGTERM, no push config.
    const file = join(agentHome, ".resident", "tasks.json");
    assert.ok(existsSync(file), "an in-flight run must be on the mount already");
    const onMount = JSON.parse(readFileSync(file, "utf8"));
    assert.equal(onMount.tasks[0].id, "task-inflight");
    assert.equal(onMount.tasks[0].status.state, "working");
  });

  it("does not re-checkpoint every working re-assertion", async () => {
    // The interactive watcher sets `working` in a loop; a gcsfuse write per
    // iteration would cost more than the run it is protecting.
    const { stateDir, agentHome } = newRoots();
    const mod = await bootResident({ stateDir, agentHome });
    const t = mod.adoptTask({
      kind: "task", id: "task-loop", contextId: "ctx-l",
      status: { state: "submitted", timestamp: new Date().toISOString() },
      history: [], metadata: {},
    });
    const file = join(agentHome, ".resident", "tasks.json");
    mod.setStateForTest(t, "working");
    const { statSync } = await import("node:fs");
    const first = statSync(file).mtimeMs;
    for (let i = 0; i < 5; i++) mod.setStateForTest(t, "working");
    assert.equal(statSync(file).mtimeMs, first, "repeated `working` must not rewrite the mount");
  });

  it("reads a pre-M6 bare-array state file", async () => {
    // The format gained a wrapper in M6. An instance restarting onto the new
    // image must not lose the tasks the old one left behind.
    const { stateDir, agentHome } = newRoots();
    writeFileSync(join(stateDir, "tasks.json"), JSON.stringify([workingTask("task-legacy")]));

    const mod = await bootResident({ stateDir, agentHome });
    const loaded = mod.loadTasks();
    assert.equal(loaded.count, 1);
    assert.ok(mod.getTask("task-legacy"), "a legacy array file should still load");
  });
});

describe("push configs are durable, because a lost one fails silently", () => {
  it("round-trips a push config through a restart", async () => {
    const { stateDir, agentHome } = newRoots();

    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-push-1"));
    first.setPushConfig("task-push-1", { url: "https://example.invalid/hook", token: "tok-abc" });

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    const cfg = second.getPushConfig("task-push-1");
    assert.ok(cfg, "the webhook must survive — losing it breaks M5a's promise with no error anywhere");
    assert.equal(cfg.url, "https://example.invalid/hook");
    assert.equal(cfg.token, "tok-abc");
  });

  it("carries the push config in the agent-home checkpoint too", async () => {
    const { stateDir, agentHome } = newRoots();

    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-push-2"));
    first.setPushConfig("task-push-2", { url: "https://example.invalid/hook2", token: "tok-xyz" });
    first.checkpoint();

    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    assert.equal(second.getPushConfig("task-push-2")?.token, "tok-xyz");
  });

  it("refuses to overwrite a good checkpoint with an empty store", async () => {
    // The path that makes this matter: a corrupt local tasks.json makes
    // loadTasks warn and return nothing, by design. Without the guard the next
    // SIGTERM writes that emptiness over the mount and the corruption becomes
    // permanent — the one file that existed to survive this.
    const { stateDir, agentHome } = newRoots();
    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-precious"));
    first.setPushConfig("task-precious", { url: "https://example.invalid/h", token: "tok" });
    first.checkpoint();

    stopResume(stateDir);
    writeFileSync(join(stateDir, "tasks.json"), "{ this is not json");

    const second = await bootResident({ stateDir, agentHome });
    assert.equal(second.loadTasks().count, 0, "a corrupt local file loads nothing, and warns");
    assert.equal(second.checkpoint(), false, "and must not clobber the mount on the way out");

    const third = await bootResident({ stateDir, agentHome });
    rmSync(join(stateDir, "tasks.json"), { force: true });
    third.loadTasks();
    assert.ok(third.getTask("task-precious"), "the checkpoint should still hold the task");
    assert.equal(third.getPushConfig("task-precious")?.token, "tok");
  });

  it("writes the checkpoint 0600, because it carries a bearer token", async () => {
    const { stateDir, agentHome } = newRoots();
    const mod = await bootResident({ stateDir, agentHome });
    mod.adoptTask(workingTask("task-mode"));
    mod.setPushConfig("task-mode", { url: "https://example.invalid/h", token: "secret" });
    mod.checkpoint();

    const { statSync } = await import("node:fs");
    const file = join(agentHome, ".resident", "tasks.json");
    assert.ok(existsSync(file), "the checkpoint should exist");
    assert.equal(statSync(file).mode & 0o077, 0, "group and other must have no access to a file holding a token");
  });
});

describe("an interrupted run is reported, not left working for ever", () => {
  it("terminalises a restored working task and pushes it to the webhook", async () => {
    const { stateDir, agentHome } = newRoots();
    const hook = webhookStub();
    const url = await hook.listen();

    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-orphan"));
    first.setPushConfig("task-orphan", { url, token: "tok-reap" });
    first.checkpoint();

    // The container is killed mid-run. Nothing ran a shutdown hook.
    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    const reaped = second.reapOrphans();
    await settle();

    assert.equal(reaped, 1, "the working task's runner died with the container");
    assert.equal(second.getTask("task-orphan").status.state, "failed");

    assert.equal(hook.posts.length, 1, "the caller must be told — this is the silent hang M6 exists to close");
    assert.equal(hook.posts[0].auth, "Bearer tok-reap", "and it must authenticate with the restored token");
    assert.equal(hook.posts[0].body.status.state, "failed");
    const said = JSON.stringify(hook.posts[0].body);
    assert.match(said, /restart/i, "the notice should say the run was interrupted by a restart");

    await hook.close();
  });

  it("does not count reaping as activity", async () => {
    // `loadTasks` seeds idleness from the newest task so a restart is not
    // mistaken for fresh work. Reaping runs through setState, which notes
    // activity — left alone it would undo that seeding and hold the instance
    // for another 30 minutes of sweep window on every wake, for runs that had
    // already finished.
    const { stateDir, agentHome } = newRoots();
    const first = await bootResident({ stateDir, agentHome });
    const old = workingTask("task-idle");
    old.status = { state: "working", timestamp: new Date(Date.now() - 3_600_000).toISOString() };
    first.adoptTask(old);
    first.checkpoint();

    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    const seeded = second.runStats().idle_s;
    assert.ok(seeded > 3000, `idleness should carry over from the task (got ${seeded}s)`);
    second.reapOrphans();
    assert.ok(second.runStats().idle_s > 3000, "reaping is bookkeeping about a dead run, not activity");
  });

  it("leaves terminal tasks alone", async () => {
    const { stateDir, agentHome } = newRoots();
    const first = await bootResident({ stateDir, agentHome });
    const done = workingTask("task-done");
    done.status = { state: "completed", timestamp: new Date().toISOString() };
    first.adoptTask(done);
    first.checkpoint();

    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    assert.equal(second.reapOrphans(), 0, "a completed task is not an orphan");
    assert.equal(second.getTask("task-done").status.state, "completed");
  });

  it("reaps a submitted task too — it never even started", async () => {
    const { stateDir, agentHome } = newRoots();
    const first = await bootResident({ stateDir, agentHome });
    const sub = workingTask("task-sub");
    sub.status = { state: "submitted", timestamp: new Date().toISOString() };
    first.adoptTask(sub);
    first.checkpoint();

    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    assert.equal(second.reapOrphans(), 1);
    assert.equal(second.getTask("task-sub").status.state, "failed");
  });

  it("survives a webhook that is refusing connections", async () => {
    // A reap that throws would leave the instance without a task store; the
    // notice is best-effort and the terminal state is not.
    const { stateDir, agentHome } = newRoots();
    const first = await bootResident({ stateDir, agentHome });
    first.adoptTask(workingTask("task-nohook"));
    first.setPushConfig("task-nohook", { url: "http://127.0.0.1:1/gone", token: "t" });
    first.checkpoint();

    stopResume(stateDir);

    const second = await bootResident({ stateDir, agentHome });
    second.loadTasks();
    assert.equal(second.reapOrphans(), 1);
    await settle();
    assert.equal(second.getTask("task-nohook").status.state, "failed");
  });
});
