/**
 * Eval Shell Guard — bounds an eval agent's bash so one confused command cannot stall the rig.
 *
 * WHY THIS EXISTS. Measured 2026-10-02: a qwen3.8 gauntlet run (pi) wanted to read
 * std/xml.ail as a file and ran `find / -path '*stdlib*xml*'` and `find / -name ailing`.
 * pi's bash tool has NO default timeout and spawns every command detached, so when the run
 * ended the shells reparented to launchd and kept scanning the disk for 7h and 33h, in
 * disk-wait, until git on the whole machine took minutes per command.
 *
 * Two rules, eval runs only (the pi executor loads this when Task.EvalShellGuard is set):
 *   1. A `find` rooted at /, ~ or $HOME without -maxdepth is refused, with the lookup the
 *      agent actually needed: the stdlib is built into the `ailang` binary.
 *   2. Every bash call gets a time limit: $AILANG_EVAL_BASH_CAP_S (default 120). A
 *      model-supplied timeout under the cap is kept; a missing or larger one is clamped.
 *
 * The harness also reaps the run's workspace on exit (proctree.ReapWorkspace); this guard
 * stops the command, the reap catches anything else left behind.
 */

const DEFAULT_CAP_S = 120;

export function capSeconds(env: Record<string, string | undefined> = process.env): number {
	const raw = env.AILANG_EVAL_BASH_CAP_S;
	if (raw === undefined || raw === "") return DEFAULT_CAP_S;
	const n = Number(raw);
	if (!Number.isFinite(n) || n <= 0) {
		throw new Error(`eval-shell-guard: AILANG_EVAL_BASH_CAP_S must be a positive number of seconds, got '${raw}'`);
	}
	return n;
}

export function clampTimeout(requested: unknown, cap: number): number {
	return typeof requested === "number" && Number.isFinite(requested) && requested > 0 && requested <= cap
		? requested
		: cap;
}

// A find whose start point is the filesystem root or the home directory. Matches at the
// start of the command or after a separator (; && || | ( `), with optional sudo/command.
const UNBOUNDED_FIND =
	/(^|[;&|(`]\s*|\$\(\s*)(sudo\s+|command\s+)?find\s+(-[HLP]\s+)*("|')?(\/|~\/?|\$HOME\/?|\$\{HOME\}\/?)("|')?(\s|$)/;

export function isUnboundedFind(command: string, home: string = process.env.HOME ?? ""): boolean {
	const cmd = command.trim();
	let rooted = UNBOUNDED_FIND.test(cmd);
	if (!rooted && home) {
		const esc = home.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
		rooted = new RegExp(`(^|[;&|(\`]\\s*|\\$\\(\\s*)(sudo\\s+|command\\s+)?find\\s+("|')?${esc}/?("|')?(\\s|$)`).test(cmd);
	}
	return rooted && !/-maxdepth\s+\d/.test(cmd);
}

export const REFUSAL =
	"Refused: a `find` over / or the home directory scans the whole disk and stalls the machine. " +
	"The AILANG standard library is built into the `ailang` binary: look a module up with " +
	"`ailang docs std/<module>` (e.g. `ailang docs std/xml`), and search examples with " +
	"`ailang examples search \"<concept>\"`. Its source directory is $AILANG_STDLIB_PATH when that is set. " +
	"To search files, stay inside your workspace or pass -maxdepth.";

export function guidance(cap: number): string {
	return [
		"",
		"## Shell limits (eval)",
		`Every bash command is stopped after ${cap}s. Never search the whole disk (no \`find /\` or \`find ~\`).`,
		"The AILANG stdlib is built into the `ailang` binary: use `ailang docs std/<module>` to read a module's API.",
	].join("\n");
}

export default function (pi: any) {
	const cap = capSeconds();

	pi.on("tool_call", async (event: any) => {
		if (event.toolName !== "bash" || !event.input) return;
		const command = typeof event.input.command === "string" ? event.input.command : "";
		if (isUnboundedFind(command)) return { block: true, reason: REFUSAL };
		event.input.timeout = clampTimeout(event.input.timeout, cap);
	});

	pi.on("before_agent_start", async (event: any) => ({
		systemPrompt: `${event.systemPrompt ?? ""}${guidance(cap)}`,
	}));
}
