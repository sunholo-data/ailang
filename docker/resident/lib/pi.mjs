// Direct pi execution in NDJSON mode (v6.40.0 M7/M9).
//
// WHY NOT THROUGH herdr: driving pi as an interactive TUI inside a herdr pane
// does not work headless. `agent.prompt` enters text without submitting it —
// context stays at 0.0% and no model call is made — and even with the prompt
// landing, herdr cannot report completion because `idle`/`done` are UI-coupled
// (Phase 0 §3). A third problem waits behind those: ailang's own
// session-protocol-gate extension branches on `ctx.hasUI`, and a TUI in a pane
// makes that true, so it would demand "a real human keypress" that never comes.
//
// `pi --mode json` sidesteps all three. It is the same invocation ailang's
// Cloud Run job executor already uses (internal/executor/pi/pi.go), it is
// headless so the gate takes its headless path, and its NDJSON carries an
// explicit `agent_end` — which finally gives this design a real terminal state
// rather than a staleness guess.
//
// herdr stays in the image for human attach (`herdr --remote`); it is simply
// no longer on the task path.
import { spawn } from "node:child_process";
import { readFileSync, existsSync, mkdirSync, readdirSync,
         openSync, writeSync, fsyncSync, closeSync } from "node:fs";
import { execFileSync } from "node:child_process";

// Capabilities are probed by boot.sh and read per run, because the server
// process was started before that probe ran and cannot have inherited it.
// Re-read rather than cache: a boot that finishes after the first call should
// still upgrade later calls from stateless to persistent.
const CAP_FILE = `${process.env.TASK_STATE_DIR || "/home/ailang/.resident"}/capabilities.json`;
// TOOL POLICY (M-AGENT-AILANG-ONLY-EXECUTION M5, D6). RESIDENT_TOOLS is a
// PROFILE — "ailang_only" (the default: read/edit/write + the AILANG gate, no
// shell), "full" (pi's own defaults), or an explicit comma list of pi tool
// names. The profile's flags come from `ailang pi tool-profile`, the ONE
// expansion internal/executor/pi uses, so the resident cannot drift from the
// fleet on what "ailang_only" means.
const PROFILE = (process.env.RESIDENT_TOOLS ?? "ailang_only").trim();
let profileArgsCache = null;
function profileArgs() {
  if (profileArgsCache) return profileArgsCache;
  if (PROFILE === "full") return (profileArgsCache = []);
  if (PROFILE === "ailang_only") {
    const out = execFileSync("ailang", ["pi", "tool-profile", "ailang_only"], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
    if (!out.includes("--no-builtin-tools")) throw new Error(`ailang pi tool-profile ailang_only returned ${JSON.stringify(out)}`);
    return (profileArgsCache = out.split(/\s+/));
  }
  const list = PROFILE.split(",").map((t) => t.trim()).filter(Boolean);
  return (profileArgsCache = list.length === 0 ? ["--no-tools"] : ["--tools", list.join(",")]);
}

/** The effective tool NAMES, for /health and the spawn log. */
export function toolPolicy() {
  const args = profileArgs();
  const i = args.indexOf("--tools");
  if (i >= 0) return args[i + 1].split(",");
  if (args.includes("--no-tools")) return [];
  return ["read", "bash", "edit", "write"]; // pi's defaults (profile full)
}
export function toolProfile() { return PROFILE; }

export function capabilities() {
  try { return JSON.parse(readFileSync(CAP_FILE, "utf8")); }
  catch { return { sessionFlag: "", sessionDir: "", agentHome: process.env.AGENT_HOME || "" }; }
}

/** Copy one file to the mount and FSYNC it before returning.
 *
 * `cpSync` and `writeFileSync` return once the bytes reach the FUSE buffer,
 * NOT once gcsfuse has uploaded the object. M6 proved the difference on this
 * exact mount: the task checkpoint's write logged success, the container was
 * destroyed moments later, and GCS never received the object (fixed in ailang
 * `ba5014074`). fsync is what makes gcsfuse finalise.
 *
 * The same fd that wrote the bytes is the one fsynced — opening a second
 * handle to fsync someone else's buffered write is not a guarantee this code
 * should be relying on.
 */
function copyFileSynced(src, dest) {
  const body = readFileSync(src);
  const fd = openSync(dest, "w");
  try { writeSync(fd, body); fsyncSync(fd); } finally { closeSync(fd); }
}

/** Recursive copy of `srcDir` INTO `destDir`, fsyncing every file written.
 *
 * Returns the number of files written. Every file is copied every time, with
 * no size/mtime skip heuristic: a false skip here means the assistant forgets
 * a conversation, which is the exact failure this function exists to prevent,
 * and it would be invisible. The cost is one fsync per session file per run —
 * worth measuring if the store grows, not worth trading correctness for now.
 */
function copyTreeSynced(srcDir, destDir) {
  mkdirSync(destDir, { recursive: true });
  let files = 0;
  for (const ent of readdirSync(srcDir, { withFileTypes: true })) {
    const src = `${srcDir}/${ent.name}`;
    const dest = `${destDir}/${ent.name}`;
    if (ent.isDirectory()) { files += copyTreeSynced(src, dest); continue; }
    // pi writes plain JSON here. A symlink or socket is unexpected, and not
    // ours to carry onto a bucket: cpSync copied a link through as a link,
    // which gcsfuse handles on its own terms and which boot.sh's `cp -a`
    // restore would faithfully recreate pointing at a path that is gone.
    // Skipped with a line in the log rather than copied invisibly.
    if (!ent.isFile()) {
      console.error(`pi | WARN skipping non-file in session store: ${src}`);
      continue;
    }
    copyFileSynced(src, dest);
    files++;
  }
  return files;
}

/** Stage the session store to the GCS mount so it survives the 7-day restart.
 *
 * Copied AFTER a run, never written to during one: gcsfuse has no POSIX
 * locking and pi rewrites its session file continuously, so pointing
 * --session-dir at the mount would corrupt exactly the state this exists to
 * keep. Same rule as the workspace, for the same reason.
 *
 * FSYNCED since 2026-09-05, for the reason on `copyFileSynced`. This ran as a
 * plain `cpSync` through all of M6 — same mount, same exposure as the
 * checkpoint bug that milestone found, and not covered by its fix. The symptom
 * would have been a resident that forgets a conversation after an idle stop,
 * which reads as a model problem rather than a storage one, so nobody would
 * have looked here.
 */
export function stageSessions() {
  const { sessionDir, agentHome } = capabilities();
  if (!sessionDir || !agentHome || !existsSync(sessionDir)) return false;
  try {
    const files = copyTreeSynced(sessionDir, `${agentHome}/sessions`);
    console.log(`pi | staged ${files} session file(s) to ${agentHome}/sessions (fsynced)`);
    return true;
  } catch (e) {
    console.error(`pi | WARN could not stage sessions to ${agentHome}: ${e.message}`);
    return false;
  }
}

// Event vocabulary from ailang's own parser, so the two stay aligned:
// session, turn_start, message_update, tool_execution_start,
// tool_execution_end, message_end, turn_end, agent_end.
export function runPi({ model, prompt, thinking, tools, cwd, onEvent, sessionId,
                        timeoutMs = 900000, ttftMs = 60000, idleMs = 180000 }) {
  const args = ["--mode", "json", "--model", model, "-p"];

  // Session handling (M10). --no-session is right for ailang's one-shot job
  // executor, which documents it as "ephemeral run (avoids ~/.pi/sessions/
  // pollution)", and WRONG for a resident: it is what made this a persistent
  // host with an amnesiac agent. --session-id creates the session if missing,
  // so the same call both starts and resumes a conversation and the caller
  // owns the identifier.
  const cap = capabilities();
  const persistent = Boolean(sessionId && cap.sessionFlag);
  if (persistent) {
    args.splice(4, 0, cap.sessionFlag, sessionId, "--session-dir", cap.sessionDir);
  } else {
    args.splice(4, 0, "--no-session");
  }
  if (thinking) args.push("--thinking", thinking);
  // TOOL POLICY — the profile (see toolPolicy above), or a per-run explicit
  // list. Default ailang_only: no bash, so `ailang run --policy` behind
  // ailang_run is a boundary and not a convenience.
  const toolPolicy = Array.isArray(tools) ? tools : toolPolicy();
  if (Array.isArray(tools)) {
    args.push(...(tools.length === 0 ? ["--no-tools"] : ["--tools", tools.join(",")]));
  } else {
    args.push(...profileArgs());
  }
  args.push(prompt);

  return new Promise((resolve, reject) => {
    // stdio: STDIN MUST BE /dev/null, NOT A PIPE.
    //
    // Go's exec.Cmd leaves Stdin nil, which the runtime wires to the null
    // device, and internal/executor/pi/pi.go relies on that. Node's spawn
    // instead defaults every stream to a pipe, so pi inherits an stdin that is
    // OPEN AND NEVER WRITTEN TO. Observed live on 2026-09-03: pi started,
    // produced not one NDJSON line, never exited, and never errored — the task
    // sat at `submitted` until the caller gave up. "ignore" is the direct
    // equivalent of Go's nil and is what makes this headless.
    const child = spawn("pi", args, {
      cwd: cwd || process.env.WORKSPACE_DIR || "/workspace",
      stdio: ["ignore", "pipe", "pipe"],
    });
    console.log(`pi | spawn model=${model} session=${persistent ? sessionId : "(stateless)"} tools=[${toolPolicy.join(",") || "none"}] cwd=${cwd || process.env.WORKSPACE_DIR || "/workspace"}`);
    const state = { text: "", events: 0, toolCalls: [], usage: null, stopReason: null, stderr: "" };
    let buf = "";

    // Three timers, mirroring pi.go's hard/TTFT/idle triple. A single 15-minute
    // hard timeout is useless as a signal: a hang and a long legitimate run look
    // identical for a quarter of an hour. TTFT converts "produced nothing" into
    // a fast, named failure — which is the only reason the stdin bug above was
    // ever visible rather than just slow.
    let settled = false;
    const timers = [];
    const clearAll = () => timers.forEach(clearTimeout);
    const fail = (msg) => {
      if (settled) return;
      settled = true;
      clearAll();
      child.kill("SIGTERM");
      reject(new Error(msg));
    };
    const timer = setTimeout(
      () => fail(`pi timed out after ${timeoutMs}ms (${state.events} events, ${state.text.length} chars)`),
      timeoutMs);
    timers.push(timer);
    timers.push(setTimeout(() => {
      if (state.events === 0) {
        fail(`pi produced no output within ${ttftMs}ms — it is not running headless. stderr: ${state.stderr.slice(0, 400) || "(none)"}`);
      }
    }, ttftMs));
    let lastEventAt = Date.now();
    const idleTimer = setInterval(() => {
      if (state.events > 0 && Date.now() - lastEventAt > idleMs) {
        clearInterval(idleTimer);
        fail(`pi went idle for ${idleMs}ms after ${state.events} events`);
      }
    }, 5000);
    timers.push(idleTimer);

    child.stdout.on("data", (d) => {
      buf += d.toString();
      let i;
      while ((i = buf.indexOf("\n")) >= 0) {
        const line = buf.slice(0, i).trim();
        buf = buf.slice(i + 1);
        if (!line) continue;
        let ev;
        try { ev = JSON.parse(line); } catch { continue; }   // tolerate non-JSON noise
        state.events++;
        lastEventAt = Date.now();
        if (state.events === 1) console.log(`pi | first event: ${ev.type}`);
        switch (ev.type) {
          case "message_update":
            if (ev.assistantMessageEvent?.type === "text_delta" && ev.assistantMessageEvent.delta) {
              state.text += ev.assistantMessageEvent.delta;
            }
            break;
          case "tool_execution_start":
            state.toolCalls.push({ id: ev.toolCallId, name: ev.toolName });
            break;
          case "message_end":
            state.usage = ev.message?.usage ?? state.usage;
            state.stopReason = ev.message?.stopReason ?? state.stopReason;
            break;
        }
        try { onEvent?.(ev, state); } catch { /* a consumer must not kill the run */ }
      }
    });
    // Surface stderr as it arrives. Buffering it until close means a process
    // that never closes takes its explanation with it.
    child.stderr.on("data", (d) => {
      const chunk = d.toString();
      state.stderr += chunk.slice(0, 4000);
      console.error(`pi | stderr: ${chunk.trim().slice(0, 500)}`);
    });
    child.on("error", (e) => { if (settled) return; settled = true; clearAll(); reject(new Error(`pi spawn failed: ${e.message}`)); });
    child.on("close", (code) => {
      if (settled) return;
      settled = true;
      clearAll();
      console.log(`pi | exit=${code} events=${state.events} chars=${state.text.length}`);
      // Stage on the way out, including on a non-zero exit: a failed turn is
      // still part of the conversation and losing it would silently rewrite
      // history.
      if (persistent) stageSessions();
      // A non-zero exit with no parsed events is a harness failure; with events
      // it is usually the model or a tool, and the text is still worth keeping.
      if (code !== 0 && state.events === 0) {
        return reject(new Error(`pi exited ${code} with no events: ${state.stderr.slice(0, 500) || "(no stderr)"}`));
      }
      resolve({ ...state, exitCode: code });
    });
  });
}
