/**
 * Controller Bash Cap — gives a pi mission controller the same per-command
 * ceiling a claude controller already has.
 *
 * WHY THIS EXISTS. pi's bash tool has NO default timeout ("optional, no default
 * timeout") and runs every command detached in its own session. A claude
 * controller cannot hold one Bash call longer than 10 minutes, and the driver's
 * stall watchdog is tuned to that rhythm: it kills a controller whose session
 * transcript is flat for STALL_SAMPLES x STALL_INTERVAL (5 x 120s = 600s) while a
 * descendant has lived >= 40 minutes. Measured 2026-09-28/29 on the glm-5.3
 * fallback rung, both shapes an uncapped command produces:
 *
 *   - world iter-208: the controller `nohup`-launched a codex executor (which
 *     pi's detached spawn then reparented OUT of the controller's process tree,
 *     so the watchdog could not see its 4 MB of output) and waited on it in ONE
 *     `for i in 1..6; sleep 280` call. Transcript flat for 11 min -> killed at
 *     03:11Z. Codex finished rc=0 at 03:17Z; the work was stranded.
 *   - stapledon 17:54Z: a `find ~ -maxdepth 6` in minute one never returned.
 *     50 minutes of nothing, then the same kill.
 *
 * With a cap below the 600s window, every command returns (result or
 * `timeout:<s>`), pi appends it to the session file, and the watchdog's progress
 * arm moves. A real wedge still dies; a patient poll no longer does.
 *
 * Cap = $MISSION_PI_BASH_CAP_S (default 540). A model-supplied timeout under the
 * cap is kept; a missing or larger one is clamped. Headless-safe: it only mutates
 * the input, never blocks or prompts.
 */

const DEFAULT_CAP_S = 540;

export function capSeconds(env: Record<string, string | undefined> = process.env): number {
	const raw = env.MISSION_PI_BASH_CAP_S;
	if (raw === undefined || raw === "") return DEFAULT_CAP_S;
	const n = Number(raw);
	// Fail loud on a malformed override rather than silently running uncapped.
	if (!Number.isFinite(n) || n <= 0) {
		throw new Error(`controller-bash-cap: MISSION_PI_BASH_CAP_S must be a positive number of seconds, got '${raw}'`);
	}
	return n;
}

export function clampTimeout(requested: unknown, cap: number): number {
	return typeof requested === "number" && Number.isFinite(requested) && requested > 0 && requested <= cap
		? requested
		: cap;
}

export function guidance(cap: number): string {
	return [
		"",
		"## Command time limit (mission controller)",
		`Every bash command is stopped after ${cap}s. To wait on long work (an executor, a test suite, a CI run):`,
		"launch it in the background in ONE command (`nohup … > out.log 2>&1 &`), then poll its log or rc file",
		`in SEPARATE short commands, each well under ${cap}s. Never wait on it inside the launching command.`,
	].join("\n");
}

export default function (pi: any) {
	const cap = capSeconds();

	pi.on("tool_call", async (event: any) => {
		if (event.toolName !== "bash" || !event.input) return;
		event.input.timeout = clampTimeout(event.input.timeout, cap);
	});

	pi.on("before_agent_start", async (event: any) => ({
		systemPrompt: `${event.systemPrompt ?? ""}${guidance(cap)}`,
	}));
}
