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

// Event vocabulary from ailang's own parser, so the two stay aligned:
// session, turn_start, message_update, tool_execution_start,
// tool_execution_end, message_end, turn_end, agent_end.
export function runPi({ model, prompt, thinking, tools, cwd, onEvent,
                        timeoutMs = 900000, ttftMs = 60000, idleMs = 180000 }) {
  const args = ["--mode", "json", "--model", model, "--no-session", "-p"];
  if (thinking) args.push("--thinking", thinking);
  if (Array.isArray(tools)) {
    // [] means no tools at all; a list restricts to it. Undefined leaves pi's
    // defaults alone.
    args.push(...(tools.length === 0 ? ["--no-tools"] : ["--tools", tools.join(",")]));
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
    console.log(`pi | spawn model=${model} cwd=${cwd || process.env.WORKSPACE_DIR || "/workspace"} argc=${args.length}`);
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
      // A non-zero exit with no parsed events is a harness failure; with events
      // it is usually the model or a tool, and the text is still worth keeping.
      if (code !== 0 && state.events === 0) {
        return reject(new Error(`pi exited ${code} with no events: ${state.stderr.slice(0, 500) || "(no stderr)"}`));
      }
      resolve({ ...state, exitCode: code });
    });
  });
}
