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
export function runPi({ model, prompt, thinking, tools, cwd, onEvent, timeoutMs = 900000 }) {
  const args = ["--mode", "json", "--model", model, "--no-session", "-p"];
  if (thinking) args.push("--thinking", thinking);
  if (Array.isArray(tools)) {
    // [] means no tools at all; a list restricts to it. Undefined leaves pi's
    // defaults alone.
    args.push(...(tools.length === 0 ? ["--no-tools"] : ["--tools", tools.join(",")]));
  }
  args.push(prompt);

  return new Promise((resolve, reject) => {
    const child = spawn("pi", args, { cwd: cwd || process.env.WORKSPACE_DIR || "/workspace" });
    const state = { text: "", events: 0, toolCalls: [], usage: null, stopReason: null, stderr: "" };
    let buf = "";
    const timer = setTimeout(() => {
      child.kill("SIGTERM");
      reject(new Error(`pi timed out after ${timeoutMs}ms (${state.events} events, ${state.text.length} chars)`));
    }, timeoutMs);

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
    child.stderr.on("data", (d) => { state.stderr += d.toString().slice(0, 4000); });
    child.on("error", (e) => { clearTimeout(timer); reject(new Error(`pi spawn failed: ${e.message}`)); });
    child.on("close", (code) => {
      clearTimeout(timer);
      // A non-zero exit with no parsed events is a harness failure; with events
      // it is usually the model or a tool, and the text is still worth keeping.
      if (code !== 0 && state.events === 0) {
        return reject(new Error(`pi exited ${code} with no events: ${state.stderr.slice(0, 500) || "(no stderr)"}`));
      }
      resolve({ ...state, exitCode: code });
    });
  });
}
