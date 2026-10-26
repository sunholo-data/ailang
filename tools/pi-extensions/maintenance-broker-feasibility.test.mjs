/** M1 lifetime stop gate: actual SDK sandbox, no provider or model calls. */
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir, release } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const here = dirname(fileURLToPath(import.meta.url));
const repo = resolve(here, "../..");
const wrapper = join(here, "maintenance-sandbox.mjs");
const runtimeModule = join(here, "sandbox/node_modules/@anthropic-ai/sandbox-runtime/dist/index.js");
const runtimeLock = join(here, "sandbox/package-lock.json");
const canary = join(here, "maintenance-lifetime-canary.py");
const hash = data => createHash("sha256").update(data).digest("hex");
const rows = [];
let controls;
const delay = ms => new Promise(yes => setTimeout(yes, ms));
const requireLifetime = process.env.MAINTENANCE_REQUIRE_LIFETIME === "1";
assert([undefined, "1"].includes(process.env.MAINTENANCE_REQUIRE_LIFETIME), "MAINTENANCE_REQUIRE_LIFETIME must be absent or 1");
assert.equal(JSON.parse(readFileSync(join(dirname(dirname(runtimeModule)), "package.json"))).version, "0.0.71");

function system(command, args, options = {}) {
 const r = spawnSync(command, args, { cwd: repo, encoding: "utf8", timeout: 20000, maxBuffer: 65536, ...options });
 assert.ifError(r.error);
 assert.equal(r.status, 0, `${command}: ${r.stderr}`);
 return r.stdout.trim();
}
function alive(pid) {
 try { process.kill(pid, 0); return true; }
 catch (error) { if (error.code === "ESRCH") return false; throw error; }
}
function inspect(pid) {
 const status = spawnSync("/bin/ps", ["-p", String(pid), "-o", "pid=,ppid=,pgid=,stat=,comm="], { encoding: "utf8", timeout: 2000, maxBuffer: 4096 });
 assert.ifError(status.error);
 if (status.status === 1 && !status.stdout.trim() && !alive(pid)) return null;
 assert.equal(status.status, 0, status.stderr);
 const fields = status.stdout.trim().split(/\s+/);
 assert.equal(Number(fields[0]), pid);
 return { snapshot: status.stdout.trim(), state: fields[3], identity: system("/bin/ps", ["-p", String(pid), "-o", "lstart=,comm="]), live: !fields[3].startsWith("Z") };
}
function kill(pid) {
 try { process.kill(pid, "SIGKILL"); }
 catch (error) { if (error.code !== "ESRCH") throw error; }
}
async function until(check, label, milliseconds = 5000) {
 const deadline = Date.now() + milliseconds;
 while (Date.now() < deadline) { const value = check(); if (value) return value; await delay(10); }
 throw new Error(`${label}: deadline expired`);
}
function env() {
 return Object.fromEntries(Object.entries(process.env).filter(([key]) => /^(PATH|LANG|LC_[A-Z_]+|TZ|TMPDIR|TMP|TEMP)$/.test(key)));
}
function fixture() {
 const root = realpathSync(mkdtempSync(join(tmpdir(), "maintenance-owner-")));
 const app = join(root, "app"), other = join(root, "bystander"), oracle = join(root, "oracle");
 for (const p of [app, other, oracle]) mkdirSync(p);
 const policy = { schema: "maintenance.containment/1", workspaceRoot: app, protectedRoots: [oracle], readOnlyRoots: [], runtimeModule, runtimeVersion: "0.0.71", runtimeLock, runtimeLockDigest: hash(readFileSync(runtimeLock)), inferenceDomains: [] };
 const file = join(root, "policy.json"); writeFileSync(file, JSON.stringify(policy));
 return { root, app, other, file, digest: hash(readFileSync(file)), ready: join(app, "child.json") };
}
function launch(f, python, mode, escape) {
 const args = [wrapper, "--policy", f.file, "--digest", f.digest, "--network", "tools", "--", python, canary, f.ready, escape, mode];
 // Separate trusted fixture controller lets the test kill the owner itself.
 const source = `const {spawn}=require('node:child_process');
const c=spawn(process.execPath,JSON.parse(process.argv[1]),{stdio:'inherit'});
c.on('error',e=>{console.error(e.message);process.exitCode=1});
c.on('close',code=>process.exitCode=code??1);`;
 const child = spawn(process.execPath, ["-e", source, JSON.stringify(args)], { cwd: f.app, env: env(), detached: true, stdio: ["ignore", "pipe", "pipe"] });
 let output = "", closed = false, exit;
 const append = b => { output += b.toString(); if (Buffer.byteLength(output) > 65536) kill(-child.pid); };
 child.stdout.on("data", append); child.stderr.on("data", append);
 child.once("error", error => { output += error.message; closed = true; exit = -1; });
 child.once("close", (code, signal) => { closed = true; exit = { code, signal }; });
 return { child, status: () => ({ closed, exit, output }) };
}

test("M1 records the lifetime blocker; passing this control does not qualify ownership", { timeout: 60000 }, async t => {
 assert.equal(process.platform, "darwin", "this qualification is macOS-specific; no silent unsupported-platform skip");
 const python = realpathSync(system("/usr/bin/which", ["python3"]));
 const ailang = realpathSync(system("/usr/bin/which", ["ailang"]));
 const root = realpathSync(mkdtempSync(join(tmpdir(), "maintenance-reap-helper-")));
 const reap = join(root, "reap");
 try {
  system("go", ["build", "-o", reap, "./tools/pi-extensions/maintenance-feasibility-reap"]);
  const f = fixture();
  try {
   const hello = "examples/runnable/hello.ail";
   mkdirSync(join(f.app, "examples/runnable"), { recursive: true });
   writeFileSync(join(f.app, hello), readFileSync(join(repo, hello)));
   const invoke = argv => system(process.execPath, [wrapper, "--policy", f.file, "--digest", f.digest, "--network", "tools", "--", ...argv], { cwd: f.app, env: env() });
   assert.match(invoke([python, "-c", "from pathlib import Path; Path('owned.txt').write_text('OWNED_PYTHON'); print(Path('owned.txt').read_text())"]), /OWNED_PYTHON/);
   invoke([ailang, "check", hello]);
   assert.match(invoke([ailang, "run", "--caps", "IO", "--entry", "main", hello]), /Hello, AILANG!/);
   controls = { python: true, ailangCheck: true, ailangRun: true, pythonVersion: system(python, ["--version"]), compilerVersion: system(ailang, ["version"]), compilerSHA256: hash(readFileSync(ailang)) };
  } finally { rmSync(f.root, { recursive: true, force: true }); }

  for (const [escape, mode] of [["stay", "normal"], ["escape", "normal"], ["escape", "timeout"], ["escape", "owner_death"]]) {
   const f = fixture(); let ownedPID; let run; let recordedRow; let bystanderClosed = false;
   const bystander = spawn("/bin/sleep", ["30"], { cwd: f.other, stdio: "ignore" });
   bystander.once("close", () => { bystanderClosed = true; });
   try {
    run = launch(f, python, mode, escape);
    const ready = await until(() => {
     if (!existsSync(f.ready)) return;
     try { return JSON.parse(readFileSync(f.ready)); }
     catch (error) { if (!(error instanceof SyntaxError)) throw error; }
    }, `ready ${escape}/${mode}`);
    ownedPID = ready.pid;
    assert(Number.isSafeInteger(ownedPID) && ownedPID > 1);
    assert.equal(ready.sid, ownedPID); assert.deepEqual(ready.closedDescriptorRange, [0, 255]);
    assert.equal(ready.cwd, escape === "escape" ? "/" : f.app);
    assert(alive(ownedPID), "fixture died before cleanup observation");
    const before = inspect(ownedPID); assert(before?.live, "fixture was not live before cleanup");
    if (mode === "normal") { await until(() => run.status().closed, "normal leader exit"); assert.equal(run.status().exit.code, 0, run.status().output); }
    else if (mode === "timeout") { kill(-run.child.pid); await until(() => run.status().closed, "group timeout"); }
    else { kill(run.child.pid); await delay(100); }
    // Current cleanup recipe, including its cwd fallback, not a reimplementation.
    kill(-run.child.pid);
    const found = JSON.parse(system(reap, [f.app], { timeout: 5000 }));
    await delay(150);
    const after = inspect(ownedPID);
    if (after) assert.equal(after.identity, before.identity, "fixture identity changed during cleanup observation");
    const survivor = after?.live ?? false, bystanderAlive = alive(bystander.pid);
    assert(bystanderAlive, "cleanup killed an unrelated process");
    if (escape === "stay") { assert.equal(survivor, false, "cwd reaper positive control failed"); assert(found.includes(ownedPID)); }
    recordedRow = { escape, mode, ready, before, after, reaped: found, survivesExistingCleanup: survivor, bystanderAlive, controller: run.status(), policySHA256: f.digest, policy: JSON.parse(readFileSync(f.file)) };
    rows.push(recordedRow);
   } finally {
    if (ownedPID && alive(ownedPID)) kill(ownedPID); // Trusted canary PID, never production ownership.
    if (run) kill(-run.child.pid);
    kill(bystander.pid);
    if (ownedPID) await until(() => !alive(ownedPID), "emergency canary cleanup");
    if (run) await until(() => run.status().closed, "fixture controller joined");
    await until(() => bystanderClosed, "bystander joined");
    if (recordedRow) { recordedRow.emergencyCleanupVerified = true; recordedRow.bystanderJoined = true; t.diagnostic(JSON.stringify(recordedRow)); }
    rmSync(f.root, { recursive: true, force: true });
   }
  }
  assert.equal(rows.length, 4);
  const escaped = rows.filter(row => row.escape === "escape");
  if (requireLifetime) {
   // This admission assertion remains RED. The default test preserves the known
   // blocker as an expected result, following the existing nested-blocker control.
   assert.equal(escaped.some(row => row.survivesExistingCleanup), false, "M1 STOP: detached cwd-changing child survives current host cleanup");
  } else {
   assert.equal(escaped.filter(row => row.survivesExistingCleanup).length, 3, "Known blocker changed; independently requalify ownership before updating this control");
  }
 } finally { rmSync(root, { recursive: true, force: true }); }
});

test.after(() => {
 const output = process.env.MAINTENANCE_FEASIBILITY_REPORT;
 if (!output) return;
 assert(isAbsolute(output));
 const survivors = rows.filter(row => row.survivesExistingCleanup);
 const complete = rows.length === 4 && rows.every(row => row.emergencyCleanupVerified && row.bystanderJoined);
 const ownerSources = Object.fromEntries(readdirSync(join(repo, "internal/proctree")).filter(name => name.endsWith(".go") && !name.endsWith("_test.go")).sort().map(name => [name, hash(readFileSync(join(repo, "internal/proctree", name)))]));
 const report = { schema: "maintenance.broker-feasibility/1", qualified: false, backendAdmitted: false,
  acquisitionComplete: complete, requireLifetimeCleanup: requireLifetime,
  status: complete && survivors.length ? "blocked" : "incomplete", blocker: complete && survivors.length ? "escaped_descendant_survives_group_kill_and_cwd_reap" : "full_chain_not_qualified",
  platform: process.platform, architecture: process.arch, kernel: release(), node: process.version,
  runtimeVersion: "0.0.71", runtimeLockSHA256: hash(readFileSync(runtimeLock)), wrapperSHA256: hash(readFileSync(wrapper)),
  canarySHA256: hash(readFileSync(canary)), testSHA256: hash(readFileSync(fileURLToPath(import.meta.url))),
  reapHelperSHA256: hash(readFileSync(join(here, "maintenance-feasibility-reap/main.go"))), ownerSources,
  controls, rows, siblingChain: complete && survivors.length ? "not_run_after_lifetime_stop_gate" : "not_run_incomplete_probe", modelCalls: 0,
  cleanup: "Trusted fixture PID used only for emergency cleanup after observation; each child also self-expires at 30 seconds. This is not an admitted owner." };
 writeFileSync(output, JSON.stringify(report, null, 2) + "\n", { flag: "wx", mode: 0o600 });
});
