/**
 * ailang-exec — M-AGENT-AILANG-ONLY-EXECUTION M3
 *
 * The ONE way an `ailang_only` agent executes a program: `ailang_run` submits
 * an .ail file to `ailang run --policy $AILANG_AGENT_POLICY`. The policy — not
 * the agent — decides caps, the Net allowlist and the FS sandbox; the gate
 * refuses every flag that could widen them (cmd/ailang/run_policy.go). Pair
 * with `--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run,builtins_search,examples_search,ailang_cli`
 * (`ailang pi tool-profile ailang_only`) and `bash` is gone, so the gate is a
 * boundary rather than a convenience. `ailang_check` lives in ailang-lsp-lite.
 *
 * DEFAULT-DENY (D4). With no AILANG_AGENT_POLICY the tool still registers, but
 * REFUSES with a named reason — a missing tool is something a model reaches
 * around; a refusal it can read is not. A policy whose fs_sandbox contains
 * the policy file's own directory is refused the same way: an agent that can
 * edit its policy has no policy.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { basename, dirname, resolve, sep } from "node:path";

export const POLICY_ENV = "AILANG_AGENT_POLICY";

/** Why a deployment cannot execute programs, or null when it can. */
export interface PolicyGate {
	policyPath: string | null;
	refusal: string | null;
}

/** Minimal TOML read of `fs_sandbox = "..."` — the one field the load check needs. */
export function fsSandboxOf(policyToml: string): string | null {
	const m = /^\s*fs_sandbox\s*=\s*"([^"]*)"/m.exec(policyToml);
	return m ? m[1] : null;
}

/** Pure: is `dir` equal to or inside `sandbox`? (both resolved) */
export function insideSandbox(sandbox: string, dir: string): boolean {
	const s = resolve(sandbox);
	const d = resolve(dir);
	return d === s || d.startsWith(s.endsWith(sep) ? s : s + sep);
}

/**
 * Pure given a reader: decide whether this deployment may execute at all.
 * - unset env            → refusal (not granted)
 * - unreadable policy    → refusal (names the path)
 * - sandbox ⊇ policy dir → refusal (D4: the agent could rewrite its own policy)
 */
export function gateFromEnv(
	env: Record<string, string | undefined>,
	read: (p: string) => string = (p) => readFileSync(p, "utf8"),
): PolicyGate {
	const policyPath = env[POLICY_ENV]?.trim() || null;
	if (!policyPath) {
		return {
			policyPath: null,
			refusal: `program execution is not granted in this deployment (${POLICY_ENV} is unset) — write the .ail file and ask the operator to attach a policy`,
		};
	}
	let toml: string;
	try {
		toml = read(policyPath);
	} catch (e) {
		return { policyPath, refusal: `policy ${policyPath} is not readable (${(e as Error).message}) — execution refused` };
	}
	const sandbox = fsSandboxOf(toml);
	if (sandbox && insideSandbox(sandbox, dirname(policyPath))) {
		return {
			policyPath,
			refusal: `policy ${policyPath} lies inside its own fs_sandbox (${sandbox}) — a program could rewrite the policy, execution refused (D4)`,
		};
	}
	return { policyPath, refusal: null };
}

/** Minimal TOML read of the fields the prompt names. */
export function policySummary(policyToml: string): { caps: string[]; sandbox: string | null; net: string[]; process: string[]; cli: string[] | null } {
	const list = (key: string): string[] | null => {
		const m = new RegExp(`^\\s*${key}\\s*=\\s*\\[([^\\]]*)\\]`, "m").exec(policyToml);
		if (!m) return null;
		return m[1].split(",").map((x) => x.trim().replace(/^"|"$/g, "")).filter(Boolean);
	};
	return {
		caps: list("allowed_caps") ?? [],
		sandbox: fsSandboxOf(policyToml),
		net: list("net_allow") ?? [],
		process: list("process_allow") ?? [],
		// null = key absent → the tool's documented default set; [] = the operator
		// listed nothing → every subcommand is refused.
		cli: list("cli_allow"),
	};
}

/**
 * The `ailang` subcommands an ailang_only agent may invoke when the policy
 * carries no `cli_allow`: read-only, or writing only the file they are given
 * inside the sandbox (fmt). Audited 2026-09-16 against all 94 top-level
 * subcommands: everything that reaches the message plane, the registry, a
 * provider, the coordinator, or spends (messages, coordinator, mission,
 * install/publish, eval-*, brain/cache, …) is out; `run`/`test`/`exec`/`repl`/
 * `replay`/`watch`/`select-best` execute programs and are refused even when
 * listed — execution only goes through ailang_run's gate.
 */
export const CLI_DEFAULT_ALLOW: readonly string[] = [
	"check", "ai-check", "iface", "fmt", "test", "docs:search", "examples", "builtins",
	"pkg-docs", "tree", "prompt", "agent-prompt", "devtools-prompt", "policy-check", "axioms", "version",
];
// `test` is NOT gate-only: the test runner evaluates PURE code only
// (internal/testing/pure_cluster.go refuses a test whose dependency has effects;
// there is no --caps flag), so it cannot reach FS/Net/Process whatever the
// policy says — and `ailang test --package .` was the first thing a package
// agent asked for (ailang-packages#64).
export const CLI_GATE_ONLY: readonly string[] = ["run", "exec", "repl", "replay", "watch", "select-best"];

export interface CliDecision { ok: boolean; reason?: string; }

/**
 * Pure: may `ailang argv...` run under this policy? Allowlist entries are
 * `cmd` or `cmd:sub` (process_allow syntax). Any argv token that names a path
 * must stay inside the sandbox — `fmt ../../etc/x` is a write outside it.
 */
export function cliDecision(argv: readonly string[], allow: readonly string[] | null, sandbox: string | null): CliDecision {
	if (argv.length === 0) return { ok: false, reason: "argv is empty" };
	const [cmd, sub] = argv;
	if (CLI_GATE_ONLY.includes(cmd)) {
		return { ok: false, reason: `\`ailang ${cmd}\` executes programs; the only execution route is the ailang_run tool (policy-gated)` };
	}
	const list = allow ?? CLI_DEFAULT_ALLOW;
	const permitted = list.some((e) => e === cmd || (sub !== undefined && e === `${cmd}:${sub}`));
	if (!permitted) {
		return { ok: false, reason: `\`ailang ${cmd}${sub ? " " + sub : ""}\` is not in the policy's cli_allow (${list.join(", ") || "empty"})` };
	}
	if (sandbox) {
		for (const tok of argv.slice(1)) {
			if (tok.startsWith("-")) continue;
			if (!(tok.startsWith("/") || tok.includes("/") || tok.startsWith(".") || tok.endsWith(".ail"))) continue;
			if (!insideSandbox(sandbox, resolve(sandbox, tok))) return { ok: false, reason: `path ${tok} is outside the FS sandbox ${sandbox}` };
		}
	}
	return { ok: true };
}

/**
 * Pure: the system-prompt section that tells the model what it IS. Without
 * this the model has only the tool descriptions and a teaching prompt whose
 * recipes say `ailang check` / `ailang run` in a shell — which it does not
 * have. Measured 2026-09-16: one of five ailang_only runs read a file and
 * stopped without ever calling ailang_run.
 */
export function lanePrompt(gate: PolicyGate, read: (p: string) => string = (p) => readFileSync(p, "utf8")): string {
	const lines = [
		"## Execution lane: ailang_only",
		"You have NO shell. Your tools are read, edit, write, ailang_check, ailang_run, builtins_search, examples_search and ailang_cli — nothing else. Use builtins_search({query}) to discover std functions (listDir, readFile, split, …) instead of guessing. Use examples_search({query}) to find a working example before writing a construct you are unsure of. Use ailang_cli({argv: [\"iface\", \"std/fs\"]}) for exact signatures of a module's exports before calling them.",
		"Package ceilings: a directory with an ailang.toml is a PACKAGE, and its `[effects] max` ceiling applies to EVERY module inside it — a probe program that reads files or runs git will be rejected there (`effect ceiling violation in package …`) no matter what the policy allows. Write scratch/probe programs OUTSIDE any package directory (e.g. at the sandbox root); only the package's own code goes inside it. NEVER edit a package's `[effects] max` to make a probe or your own program pass — the ceiling is the package's public contract, and widening it is the change under review, not a workaround.",
		"The ONLY way to execute anything is `ailang_run` on an AILANG (.ail) file you have written. Do not ask for bash, do not describe commands you would run, do not stop after reading: write the program, `ailang_check` it, then `ailang_run` it.",
		"Module naming: a file named report.ail must start with `module report` (the bare file name — no directory prefix, no hyphens).",
		"Paths: AILANG resolves every relative path in a program (readFile, listDir, exec's working directory) against the FS SANDBOX ROOT below, not against the file's location. Write paths relative to that root (or absolute paths inside it).",
		"Every effect a program uses must be declared in its entry function's effect row (`! {IO, FS}`); the typechecker enforces this through imports, and the gate admits the program only if the declared row is a subset of the policy below.",
	];
	if (gate.refusal || !gate.policyPath) {
		lines.push(`Execution is NOT granted in this deployment (${gate.refusal ?? "no policy"}). You can still write and type-check programs; say plainly that you cannot run them.`);
		return lines.join("\n");
	}
	let sum = { caps: [] as string[], sandbox: null as string | null, net: [] as string[], process: [] as string[], cli: null as string[] | null };
	try { sum = policySummary(read(gate.policyPath)); } catch { /* the tool will refuse; the prompt stays generic */ }
	lines.push(`Policy: allowed effects = {${sum.caps.join(", ") || "none — every program is denied"}}.`);
	if (sum.sandbox) lines.push(`FS is confined to ${sum.sandbox}: relative paths resolve from there, subprocesses run from there, and paths outside it are rejected at run time.`);
	if (sum.caps.includes("Net")) lines.push(`Net is allowed only to: ${sum.net.join(", ") || "(no hosts listed)"}.`);
	else lines.push("There is no network access. Do not attempt HTTP.");
	if (sum.caps.includes("Process")) lines.push(`Process is allowed only for: ${sum.process.join(", ") || "(no commands listed — every process call is refused)"} (cmd:sub narrows to a subcommand).`);
	else lines.push("There is no process/subprocess access.");
	lines.push(`ailang_cli may run only these subcommands: ${(sum.cli ?? CLI_DEFAULT_ALLOW).join(", ") || "(none)"} — never run (use ailang_run); test evaluates pure tests only.`);
	lines.push("A denial names `missing_from_policy`: narrow the program's effects instead of retrying the same thing. Programs are ordinary AILANG modules with `export func main() -> () ! {…}`; use std/fs, std/io, std/string for what you would have done with shell tools.");
	return lines.join("\n");
}

/**
 * The canonical AILANG teaching prompt (`ailang prompt`, the ACTIVE version —
 * never a written-down copy), for the system role. An ailang_only agent has to
 * write AILANG and nothing else, and on the coordinator/Jobs path nothing else
 * teaches it: the eval harness folds this same prompt into every run, but a
 * registry task carries only the repo's AGENTS.md. Measured 2026-09-16: without
 * it a model rewrote a valid file into `import std/fs { listDir }`. ~23k tokens,
 * provider-cached; empty string when `ailang prompt` fails (the tool still
 * works, the model just guesses — the load log says so).
 */
export function teachingPrompt(run: (cmd: string, args: string[]) => string = (c, a) => execFileSync(c, a, { encoding: "utf8", maxBuffer: 4 * 1024 * 1024, stdio: ["ignore", "pipe", "pipe"] })): string {
	if ((process.env.AILANG_LANE_TEACHING ?? "1") === "0") return "";
	try {
		return run("ailang", ["prompt"]).trim();
	} catch (e) {
		console.error(`ailang-exec: teaching prompt unavailable (${(e as Error).message}); the model will not be taught AILANG syntax`);
		return "";
	}
}

/** The `policy: {...}` admission line `ailang run --policy` prints on stderr. */
export function parsePolicyLine(stderr: string): Record<string, unknown> | null {
	const m = /^policy: (\{.*\})\s*$/m.exec(stderr);
	if (!m) return null;
	try {
		return JSON.parse(m[1]) as Record<string, unknown>;
	} catch {
		return null;
	}
}

export interface RunEnvelope {
	admitted: boolean;
	exit_code: number;
	decision: unknown;
	policy_digest: string;
	stdout: string;
	stderr: string;
}

/**
 * Pure: compose the tool result from a finished `ailang run --policy`.
 * Denial (exit 2, decision JSON on stdout) and admission (policy line on stderr)
 * are the gate's two documented shapes; anything else is reported as-is with
 * admitted=false so the model never mistakes a crash for a refusal.
 */
export function composeEnvelope(code: number, stdout: string, stderr: string): RunEnvelope {
	const admission = parsePolicyLine(stderr);
	if (admission && admission.ok === true) {
		return {
			admitted: true,
			exit_code: code,
			decision: admission.decision ?? null,
			policy_digest: String(admission.policy_digest ?? ""),
			stdout,
			stderr: stderr.replace(/^policy: \{.*\}\s*$/m, "").trim(),
		};
	}
	let decision: unknown = null;
	try {
		const parsed = JSON.parse(stdout) as { decision?: unknown };
		decision = parsed.decision ?? parsed;
	} catch {
		decision = null;
	}
	return { admitted: false, exit_code: code, decision, policy_digest: "", stdout: decision ? "" : stdout, stderr };
}

export default async function (pi: ExtensionAPI) {
	const { Type } = await import("typebox");
	const gate = gateFromEnv(process.env);

	// Tell the model what it is (pure text from the policy; nothing secret).
	// Delivered BOTH as a system-prompt section and as a conversation message:
	// measured 2026-09-16, the `ollama/glm-5.3-flash:cloud` route discards the
	// system role entirely (deepseek via ollama and OpenRouter glm honour it),
	// so a system-prompt-only injection would silently vanish on the rig's
	// default pi model. The teaching prompt's shell recipes stay; this section
	// says they do not apply here.
	const lane = lanePrompt(gate);
	// The teaching prompt goes in the SYSTEM role only (it is large); the lane
	// section goes both ways because some routes drop the system role.
	const teaching = gate.refusal ? "" : teachingPrompt();
	const teachingSection = teaching ? `\n\n## AILANG language reference (canonical teaching prompt)\n\n${teaching}` : "";
	pi.on("before_agent_start", async (ev) => ({
		systemPrompt: `${ev.systemPrompt}\n\n${lane}${teachingSection}`,
		message: { customType: "ailang-lane", content: lane, display: false },
	}));

	pi.registerTool({
		name: "ailang_run",
		label: "AILANG Run (policy-gated)",
		description:
			"Execute an AILANG program under the operator's policy: `ailang run --policy <policy> <path>`. " +
			"The program's declared effect row must be a subset of the policy's allowed_caps; FS stays inside " +
			"the policy's sandbox. Returns {admitted, exit_code, decision, policy_digest, stdout, stderr}. " +
			"Denied programs never execute — read `decision.missing_from_policy` and narrow the program's effects. " +
			"This is the ONLY way to run code; there is no shell.",
		parameters: Type.Object({
			path: Type.String({ description: "Path to the .ail file (relative paths keep module resolution happy)" }),
			args_json: Type.Optional(Type.String({ description: "JSON arguments for the entrypoint (passed as --args-json)" })),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			void ctx;
			if (gate.refusal) {
				const text = JSON.stringify({ admitted: false, refused: gate.refusal });
				return { content: [{ type: "text", text }], details: { admitted: false, refused: gate.refusal } };
			}
			// Run IN the file's directory with the bare filename. ailang's module
			// rule (MOD010) wants `module x` for x.ail relative to the cwd; an
			// absolute path makes the canonical path the whole absolute prefix,
			// and the first six Jobs tasks (2026-09-16) burned turns cycling
			// through `module tmp/ailang-only/x`, `module x`, `module workspace/…`.
			const abs = resolve(params.path);
			const args = ["run", "--policy", gate.policyPath as string];
			if (params.args_json) args.push("--args-json", params.args_json);
			args.push(basename(abs));
			const r = await pi.exec("ailang", args, { timeout: 120_000, cwd: dirname(abs) });
			const env = composeEnvelope(r.code ?? -1, r.stdout ?? "", r.stderr ?? "");
			return { content: [{ type: "text", text: JSON.stringify(env) }], details: env };
		},
	});

	// The rest of the ailang CLI, allowlisted by the same policy file. The
	// model reaches the binary ONLY through this tool: argv is an array (no
	// shell), the cwd is the sandbox root, paths must stay inside it, and the
	// program-executing subcommands are refused whatever the list says.
	const policyToml = gate.policyPath ? (() => { try { return readFileSync(gate.policyPath as string, "utf8"); } catch { return ""; } })() : "";
	const cliSum = policySummary(policyToml);
	pi.registerTool({
		name: "ailang_cli",
		label: "AILANG CLI (policy-allowlisted)",
		description:
			"Run an allowlisted `ailang <subcommand>` — iface (exact export signatures), fmt, ai-check (type-check + Z3 verification), " +
			"test (pure tests only; --package for a package), docs search, examples, builtins, pkg-docs, tree, prompt, policy-check. NOT run: execution only goes through ailang_run. " +
			"argv is passed as an array with no shell; paths must stay inside the FS sandbox. Returns {ok, exit_code, stdout, stderr}.",
		parameters: Type.Object({
			argv: Type.Array(Type.String(), { description: 'Subcommand and its arguments, e.g. ["iface", "std/fs"] or ["ai-check", "report.ail"]' }),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			void ctx;
			if (gate.refusal) {
				const text = JSON.stringify({ ok: false, refused: gate.refusal });
				return { content: [{ type: "text", text }], details: { ok: false, refused: gate.refusal } };
			}
			const d = cliDecision(params.argv, cliSum.cli, cliSum.sandbox);
			if (!d.ok) {
				const text = JSON.stringify({ ok: false, refused: d.reason });
				return { content: [{ type: "text", text }], details: { ok: false, refused: d.reason } };
			}
			const cwd = cliSum.sandbox ?? process.cwd();
			const r = await pi.exec("ailang", params.argv, { timeout: 60_000, cwd });
			const cap = (t: string) => (t.length > 64_000 ? t.slice(0, 64_000) + "\n…[truncated]" : t);
			const out = { ok: (r.code ?? -1) === 0, exit_code: r.code ?? -1, stdout: cap(r.stdout ?? ""), stderr: cap(r.stderr ?? "") };
			return { content: [{ type: "text", text: JSON.stringify(out) }], details: out };
		},
	});
}
