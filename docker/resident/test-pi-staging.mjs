// Session-store staging to the GCS mount (v6.40.0, follow-on to M6).
//
//     node --test docker/resident/test-pi-staging.mjs                    # laptop
//     RESIDENT_LIB=/usr/local/bin/lib node --test test-pi-staging.mjs    # in-image
//
// WHY THIS EXISTS
//
// M6 found that a write to gcsfuse can return success and leave no object:
// `writeFileSync` returns when the bytes reach the FUSE buffer, not when the
// object exists. It fixed the task checkpoint (ailang `ba5014074`) and did NOT
// fix `stageSessions`, which staged the whole pi session store to the SAME
// mount with a plain `cpSync` at the end of every run.
//
// The symptom of that gap is a resident that forgets a conversation after an
// idle stop — which reads as a model problem, not a storage one, so it would
// not have been looked for here.
//
// WHAT THESE TESTS CAN AND CANNOT PROVE
//
// They CANNOT prove the fsync. A tmpdir has no upload gap, which is precisely
// why every simulated restart passed before M6. Durability is asserted where
// it is observable — `verify-resident-chaos.sh` reads the OBJECT out of GCS
// after a real stop, and `test-image.sh` asserts the fsync is still in the
// source.
//
// What they DO prove is that the hand-rolled recursive copy that replaced
// `cpSync` is faithful: nested directories, overwrite of a stale copy, an
// empty store, and a non-file entry that must be skipped rather than followed
// into a bucket. Those are the ways a durability fix silently becomes a data
// loss.
import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, mkdirSync, writeFileSync, readFileSync,
         existsSync, symlinkSync, readdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const LIB = process.env.RESIDENT_LIB || new URL("./lib", import.meta.url).pathname;

let root;
let pi;

// capabilities() re-reads the capability file on every call, but the PATH to
// it is frozen at module load (`const CAP_FILE`). So TASK_STATE_DIR has to be
// set before the import — which is what beforeEach does — and after that the
// contents can be rewritten per test with no reload and no monkeypatching.
function setCapabilities({ sessionDir, agentHome }) {
  const stateDir = join(root, "state");
  mkdirSync(stateDir, { recursive: true });
  writeFileSync(join(stateDir, "capabilities.json"), JSON.stringify({
    piVersion: "test", sessionFlag: "--session-id", sessionDir, agentHome,
  }));
}

function tree(dir) {
  const out = [];
  const walk = (d, prefix) => {
    for (const ent of readdirSync(d, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      const rel = prefix ? `${prefix}/${ent.name}` : ent.name;
      if (ent.isDirectory()) walk(join(d, ent.name), rel);
      else out.push(rel);
    }
  };
  walk(dir, "");
  return out;
}

describe("stageSessions", () => {
  beforeEach(async () => {
    root = mkdtempSync(join(tmpdir(), "pi-staging-"));
    process.env.TASK_STATE_DIR = join(root, "state");
    pi = await import(`${LIB}/pi.mjs?t=${Date.now()}`);
  });
  afterEach(() => {
    rmSync(root, { recursive: true, force: true });
    delete process.env.TASK_STATE_DIR;
  });

  it("copies a nested session store into the mount, content intact", () => {
    const sessionDir = join(root, "local/sessions");
    const agentHome = join(root, "mount");
    mkdirSync(join(sessionDir, "projects/abc"), { recursive: true });
    writeFileSync(join(sessionDir, "index.json"), '{"sessions":["s1"]}');
    writeFileSync(join(sessionDir, "projects/abc/s1.json"), '{"turns":42}');
    setCapabilities({ sessionDir, agentHome });

    assert.equal(pi.stageSessions(), true);

    // Assert the FILES, not the return value. "it was staged" is exactly the
    // thing that was true before M6 and still lost the data.
    assert.deepEqual(tree(join(agentHome, "sessions")),
      ["index.json", "projects/abc/s1.json"]);
    assert.equal(readFileSync(join(agentHome, "sessions/projects/abc/s1.json"), "utf8"),
      '{"turns":42}');
  });

  it("overwrites a stale copy rather than leaving the old conversation", () => {
    const sessionDir = join(root, "local/sessions");
    const agentHome = join(root, "mount");
    mkdirSync(sessionDir, { recursive: true });
    mkdirSync(join(agentHome, "sessions"), { recursive: true });
    writeFileSync(join(agentHome, "sessions/s1.json"), '{"turns":1}');
    writeFileSync(join(sessionDir, "s1.json"), '{"turns":2}');
    setCapabilities({ sessionDir, agentHome });

    assert.equal(pi.stageSessions(), true);
    assert.equal(readFileSync(join(agentHome, "sessions/s1.json"), "utf8"), '{"turns":2}');
  });

  it("truncates when the new session file is SHORTER than the staged one", () => {
    // openSync(dest, "w") truncates; an implementation that opened "r+" to
    // fsync in place would leave a tail of the previous conversation glued to
    // the end of the new one, and it would still parse as a longer file.
    const sessionDir = join(root, "local/sessions");
    const agentHome = join(root, "mount");
    mkdirSync(sessionDir, { recursive: true });
    mkdirSync(join(agentHome, "sessions"), { recursive: true });
    writeFileSync(join(agentHome, "sessions/s1.json"), "x".repeat(500));
    writeFileSync(join(sessionDir, "s1.json"), "short");
    setCapabilities({ sessionDir, agentHome });

    assert.equal(pi.stageSessions(), true);
    assert.equal(readFileSync(join(agentHome, "sessions/s1.json"), "utf8"), "short");
  });

  it("creates the mount directory when this is the first run", () => {
    const sessionDir = join(root, "local/sessions");
    const agentHome = join(root, "mount");
    mkdirSync(sessionDir, { recursive: true });
    writeFileSync(join(sessionDir, "s1.json"), "{}");
    setCapabilities({ sessionDir, agentHome });

    assert.equal(existsSync(join(agentHome, "sessions")), false);
    assert.equal(pi.stageSessions(), true);
    assert.equal(existsSync(join(agentHome, "sessions/s1.json")), true);
  });

  it("skips a symlink rather than carrying a link entry onto the mount", () => {
    // `cpSync` copied a link through AS a link (verified: it does not
    // dereference, so nothing was ever leaked by value). That is still not
    // something to put in a bucket — gcsfuse's symlink support is its own
    // question, and boot.sh's restore is `cp -a`, which would faithfully
    // recreate a pointer to a local path that no longer exists. pi writes
    // plain JSON here; anything else is unexpected and is worth a line in the
    // log rather than a silent copy.
    const sessionDir = join(root, "local/sessions");
    const agentHome = join(root, "mount");
    mkdirSync(sessionDir, { recursive: true });
    writeFileSync(join(root, "elsewhere.txt"), "not ours to stage");
    writeFileSync(join(sessionDir, "s1.json"), "{}");
    symlinkSync(join(root, "elsewhere.txt"), join(sessionDir, "link.txt"));
    setCapabilities({ sessionDir, agentHome });

    assert.equal(pi.stageSessions(), true);
    assert.deepEqual(tree(join(agentHome, "sessions")), ["s1.json"]);
  });

  it("reports failure rather than throwing when the store is unreadable", () => {
    // The caller stages on the way out of EVERY run, including a failed one.
    // A throw here would turn a staging problem into a lost turn.
    setCapabilities({ sessionDir: join(root, "nope"), agentHome: join(root, "mount") });
    assert.equal(pi.stageSessions(), false);
  });

  it("declines when the capability probe found no session support", () => {
    setCapabilities({ sessionDir: "", agentHome: join(root, "mount") });
    assert.equal(pi.stageSessions(), false);
    assert.equal(existsSync(join(root, "mount/sessions")), false);
  });
});
