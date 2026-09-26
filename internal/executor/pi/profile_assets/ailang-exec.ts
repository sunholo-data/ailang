/**
 * ailang-exec — M-AGENT-AILANG-ONLY-EXECUTION M3 / M-EXECUTOR-POLICY-HARDENING M4
 *
 * The ONE way an `ailang_only` agent touches the world:
 *
 *   ailang_run                 submits an .ail file to `ailang run --policy $AILANG_AGENT_POLICY`
 *   ailang_read / write / edit file access through `ailang policy-tool` — root-anchored in Go
 *   ailang_cli                 the allowlisted rest of the CLI, as TYPED requests (op + fields),
 *                              also through `ailang policy-tool`
 *
 * The policy — not the agent — decides caps, the Net allowlist, the FS sandbox
 * and the CLI surface. Nothing here parses the policy: the summary the model
 * is shown, and every yes/no about a path or a subcommand, comes from the Go
 * endpoint (`policy-tool`), so the tool wrapper and the runtime can never
 * disagree about what a field means. This extension composes requests and
 * relays responses; it never composes an argv for the CLI itself.
 *
 * DEFAULT-DENY (D4). With no AILANG_AGENT_POLICY the tools still register,
 * but REFUSE with a named reason — a missing tool is something a model reaches
 * around; a refusal it can read is not. A policy whose fs_sandbox contains
 * the policy file's own directory is refused the same way: an agent that can
 * edit its policy has no policy.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { readFileSync } from "node:fs";
import { execFileSync, spawn } from "node:child_process";
import { basename, dirname, resolve, sep } from "node:path";

export const POLICY_ENV = "AILANG_AGENT_POLICY";

/** Why a deployment cannot execute programs, or null when it can. */
export interface PolicyGate {
	policyPath: string | null;
	refusal: string | null;
}

/** The Go-produced summary of the resolved policy (`policy-tool` op=summary). */
export interface PolicySummary {
	security_mode: string;
	policy_digest: string;
	fs_sandbox: string;
	caps: string[];
	net_allow: string[];
	process_allow: string[];
	cli: string[];
	ops: string[];
	timeout_ms: number;
}

/** One typed request to `ailang policy-tool`. Mirrors internal/policytool.Request. */
export interface ToolRequest {
	op: string;
	path?: string;
	content?: string;
	old_text?: string;
	new_text?: string;
	module?: string;
	query?: string;
	package?: string;
	flags?: Record<string, string>;
}

/** The endpoint's one response shape. Mirrors internal/policytool.Response. */
export interface ToolResponse {
	ok: boolean;
	refused?: string;
	content?: string;
	argv?: string[];
	exit_code?: number;
	stdout?: string;
	stderr?: string;
	summary?: PolicySummary;
}

/** Runs one request against the endpoint. Injected so the composition is testable without a binary. */
export type ToolRunner = (policyPath: string, req: ToolRequest) => Promise<ToolResponse>;

/**
 * The production runner: `ailang policy-tool --policy <p>` with the request
 * on stdin and the response on stdout. The policy path is the launcher's
 * (from the gate), never a request field.
 */
export function defaultToolRunner(timeoutMs = 130_000): ToolRunner {
	return (policyPath, req) =>
		new Promise((resolveP) => {
			const child = spawn("ailang", ["policy-tool", "--policy", policyPath], { stdio: ["pipe", "pipe", "pipe"] });
			let out = "";
			let err = "";
			const timer = setTimeout(() => child.kill("SIGKILL"), timeoutMs);
			child.stdout.on("data", (d) => (out += String(d)));
			child.stderr.on("data", (d) => (err += String(d)));
			child.on("error", (e) => {
				clearTimeout(timer);
				resolveP({ ok: false, refused: `policy-tool could not start: ${e.message}` });
			});
			child.on("close", (code) => {
				clearTimeout(timer);
				try {
					resolveP(JSON.parse(out) as ToolResponse);
				} catch {
					resolveP({ ok: false, refused: `policy-tool produced no response (exit ${code}): ${err.trim().slice(0, 2000)}` });
				}
			});
			child.stdin.end(JSON.stringify(req));
		});
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
 * The D4 sandbox check needs the RESOLVED policy and lives in gateWithSummary.
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
	try {
		read(policyPath);
	} catch (e) {
		return { policyPath, refusal: `policy ${policyPath} is not readable (${(e as Error).message}) — execution refused` };
	}
	return { policyPath, refusal: null };
}

/**
 * Pure: apply the endpoint's verdict to the gate. A policy the endpoint
 * refuses to resolve is a refusal; a sandbox that contains the policy's own
 * directory is a refusal (D4: the agent could rewrite its policy).
 */
export function gateWithSummary(gate: PolicyGate, resp: ToolResponse): { gate: PolicyGate; summary: PolicySummary | null } {
	if (gate.refusal || !gate.policyPath) return { gate, summary: null };
	if (!resp.ok || !resp.summary) {
		return { gate: { policyPath: gate.policyPath, refusal: `policy ${gate.policyPath} does not resolve: ${resp.refused ?? "no summary"} — execution refused` }, summary: null };
	}
	const s = resp.summary;
	if (s.fs_sandbox && insideSandbox(s.fs_sandbox, dirname(gate.policyPath))) {
		return {
			gate: { policyPath: gate.policyPath, refusal: `policy ${gate.policyPath} lies inside its own fs_sandbox (${s.fs_sandbox}) — a program could rewrite the policy, execution refused (D4)` },
			summary: null,
		};
	}
	return { gate, summary: s };
}

/**
 * Pure: the system-prompt section that tells the model what it IS. Without
 * this the model has only the tool descriptions and a teaching prompt whose
 * recipes say `ailang check` / `ailang run` in a shell — which it does not
 * have. Measured 2026-09-16: one of five ailang_only runs read a file and
 * stopped without ever calling ailang_run.
 */
export function lanePrompt(gate: PolicyGate, summary: PolicySummary | null): string {
	const lines = [
		"## Execution lane: ailang_only",
		"You have NO shell. Your tools are ailang_read, ailang_edit, ailang_write, ailang_check, ailang_run, builtins_search, examples_search and ailang_cli — nothing else. Use builtins_search({query}) to discover std functions (listDir, readFile, split, …) instead of guessing. Use examples_search({query}) to find a working example before writing a construct you are unsure of. Use ailang_cli({op: \"iface\", module: \"std/fs\"}) for exact signatures of a module's exports before calling them.",
		"Package ceilings: a directory with an ailang.toml is a PACKAGE, and its `[effects] max` ceiling applies to EVERY module inside it — a probe program that reads files or runs git will be rejected there (`effect ceiling violation in package …`) no matter what the policy allows. Write scratch/probe programs in `.ailang-scratch/` at the sandbox root — never inside a package directory, and never anywhere else: that one directory is excluded from the commit, and you have no delete tool, so a probe left elsewhere ships in the PR. Only the package's own code goes inside the package. NEVER edit a package's `[effects] max` to make a probe or your own program pass — the ceiling is the package's public contract, and widening it is the change under review, not a workaround.",
		"The ONLY way to execute anything is `ailang_run` on an AILANG (.ail) file you have written. Do not ask for bash, do not describe commands you would run, do not stop after reading: write the program, `ailang_check` it, then `ailang_run` it.",
		"Module naming: a file named report.ail must start with `module report` (the bare file name — no directory prefix, no hyphens).",
		"Paths: every path you give a tool, and every relative path in a program (readFile, listDir), resolves against the FS SANDBOX ROOT below. Paths outside it — including symlinks that lead outside — are refused by the runtime and by the tools.",
		"Every effect a program uses must be declared in its entry function's effect row (`! {IO, FS}`); the typechecker enforces this through imports, and the gate admits the program only if the declared row is a subset of the policy below.",
	];
	if (gate.refusal || !gate.policyPath || !summary) {
		lines.push(`Execution is NOT granted in this deployment (${gate.refusal ?? "no policy"}). You can still write and type-check programs; say plainly that you cannot run them.`);
		return lines.join("\n");
	}
	lines.push(`Policy (${summary.security_mode}): allowed effects = {${summary.caps.join(", ") || "none — every program is denied"}}.`);
	if (summary.fs_sandbox) lines.push(`FS is confined to ${summary.fs_sandbox}: relative paths resolve from there, and paths outside it are refused.`);
	if (summary.caps.includes("Net") || summary.caps.includes("Stream")) lines.push(`Net is allowed only to: ${summary.net_allow.join(", ") || "(no hosts listed)"}. Redirects to other hosts are refused.`);
	else lines.push("There is no network access. Do not attempt HTTP.");
	if (summary.caps.includes("Process")) lines.push(`Process is allowed only for: ${summary.process_allow.join(", ") || "(no commands listed — every process call is refused)"} (cmd:sub narrows to a subcommand).`);
	else lines.push("There is no process/subprocess access.");
	lines.push(`ailang_cli ops available under this policy: ${summary.cli.join(", ") || "(none)"} — never run (use ailang_run); test evaluates pure tests only.`);
	lines.push(`Each run is bounded: timeout ${summary.timeout_ms} ms for the whole invocation.`);
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

/** The `policy-result: {...}` limit envelope the supervisor prints on a timeout or output cap. */
export function parseResultLine(stderr: string): Record<string, unknown> | null {
	const m = /^policy-result: (\{.*\})\s*$/m.exec(stderr);
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
	/** Set when the supervisor stopped the run: {reason: "timeout" | "output_limit" | …}. */
	limit?: Record<string, unknown> | null;
	stdout: string;
	stderr: string;
}

/**
 * Pure: compose the tool result from a finished `ailang run --policy`.
 * Denial (exit 2, decision JSON on stdout), admission (policy line on stderr)
 * and a supervisor limit (exit 3, policy-result line on stderr) are the
 * gate's documented shapes; anything else is reported as-is with
 * admitted=false so the model never mistakes a crash for a refusal.
 */
export function composeEnvelope(code: number, stdout: string, stderr: string): RunEnvelope {
	const admission = parsePolicyLine(stderr);
	const limit = parseResultLine(stderr);
	const cleanErr = stderr.replace(/^policy: \{.*\}\s*$/m, "").replace(/^policy-result: \{.*\}\s*$/m, "").trim();
	if (admission && admission.ok === true) {
		return {
			admitted: true,
			exit_code: code,
			decision: admission.decision ?? null,
			policy_digest: String(admission.policy_digest ?? ""),
			limit,
			stdout,
			stderr: cleanErr,
		};
	}
	let decision: unknown = null;
	try {
		const parsed = JSON.parse(stdout) as { decision?: unknown };
		decision = parsed.decision ?? parsed;
	} catch {
		decision = null;
	}
	return { admitted: false, exit_code: code, decision, policy_digest: "", limit, stdout: decision ? "" : stdout, stderr: cleanErr || stderr };
}

/**
 * Pure: the typed request an ailang_cli call turns into. The tool's
 * parameters ARE the request fields; nothing is parsed out of a string.
 */
export function cliRequest(params: { op: string; path?: string; module?: string; query?: string; package?: string; flags?: Record<string, string> }): ToolRequest {
	const req: ToolRequest = { op: params.op };
	if (params.path) req.path = params.path;
	if (params.module) req.module = params.module;
	if (params.query) req.query = params.query;
	if (params.package) req.package = params.package;
	if (params.flags && Object.keys(params.flags).length > 0) req.flags = params.flags;
	return req;
}

/** Text + details for a tool result. */
function result(payload: unknown) {
	return { content: [{ type: "text" as const, text: JSON.stringify(payload) }], details: payload as Record<string, unknown> };
}

/**
 * `register` is the extension body with its dependencies injected so the
 * registration path is testable without pi or typebox: `Type` builds the tool
 * parameter schemas, `env` is the process environment, `runner` talks to the
 * endpoint.
 */
export async function register(pi: ExtensionAPI, Type: TypeLike, env: Record<string, string | undefined> = process.env, runner: ToolRunner = defaultToolRunner()) {
	let gate = gateFromEnv(env);
	let summary: PolicySummary | null = null;
	if (!gate.refusal && gate.policyPath) {
		const verdict = gateWithSummary(gate, await runner(gate.policyPath, { op: "summary" }));
		gate = verdict.gate;
		summary = verdict.summary;
	}

	// Tell the model what it is. Delivered BOTH as a system-prompt section
	// and as a conversation message: measured 2026-09-16, the
	// `ollama/glm-5.3-flash:cloud` route discards the system role entirely.
	// The lane exists ONLY when a policy is attached (measured 2026-09-19: on
	// a plain `pi` with bash, "You have NO shell" talked sessions out of a
	// shell they had).
	if (!gate.refusal && summary) {
		const lane = lanePrompt(gate, summary);
		const teaching = teachingPrompt();
		const teachingSection = teaching ? `\n\n## AILANG language reference (canonical teaching prompt)\n\n${teaching}` : "";
		pi.on("before_agent_start", async (ev) => ({
			systemPrompt: `${ev.systemPrompt}\n\n${lane}${teachingSection}`,
			message: { customType: "ailang-lane", content: lane, display: false },
		}));
	}

	const refusedResult = () => result({ ok: false, admitted: false, refused: gate.refusal });
	const call = async (req: ToolRequest) => result(await runner(gate.policyPath as string, req));

	pi.registerTool({
		name: "ailang_run",
		label: "AILANG Run (policy-gated)",
		description:
			"Execute an AILANG program under the operator's policy: `ailang run --policy <policy> <path>`. " +
			"The program's declared effect row must be a subset of the policy's allowed_caps; FS stays inside " +
			"the policy's sandbox; the whole invocation is bounded by the policy's timeout. Returns " +
			"{admitted, exit_code, decision, policy_digest, limit, stdout, stderr}. " +
			"Denied programs never execute — read `decision.missing_from_policy` and narrow the program's effects. " +
			"This is the ONLY way to run code; there is no shell.",
		parameters: Type.Object({
			path: Type.String({ description: "Path to the .ail file (relative paths keep module resolution happy)" }),
			args_json: Type.Optional(Type.String({ description: "JSON arguments for the entrypoint (passed as --args-json)" })),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			void ctx;
			if (gate.refusal) return refusedResult();
			// Run IN the file's directory with the bare filename. ailang's module
			// rule (MOD010) wants `module x` for x.ail relative to the cwd.
			const abs = resolve(params.path);
			// The program must live in the sandbox (the gate refuses otherwise —
			// this is the readable reason, not the authority): running an .ail
			// from elsewhere would read sources outside the clone under the policy.
			if (summary?.fs_sandbox && !insideSandbox(summary.fs_sandbox, abs)) {
				return result({ admitted: false, refused: `program ${abs} is outside the FS sandbox ${summary.fs_sandbox}; write it inside the sandbox root and run it from there` });
			}
			const args = ["run", "--policy", gate.policyPath as string];
			if (params.args_json) args.push("--args-json", params.args_json);
			args.push(basename(abs));
			const r = await pi.exec("ailang", args, { timeout: 130_000, cwd: dirname(abs) });
			return result(composeEnvelope(r.code ?? -1, r.stdout ?? "", r.stderr ?? ""));
		},
	});

	pi.registerTool({
		name: "ailang_read",
		label: "Read (sandboxed)",
		description: "Read a file inside the policy's FS sandbox. Paths resolve from the sandbox root; anything outside it, including symlinks that lead outside, is refused. Returns {ok, content} or {ok:false, refused}.",
		parameters: Type.Object({ path: Type.String({ description: "File path, relative to the sandbox root (or absolute inside it)" }) }),
		async execute(_id, params) {
			if (gate.refusal) return refusedResult();
			return call({ op: "read", path: params.path });
		},
	});

	pi.registerTool({
		name: "ailang_write",
		label: "Write (sandboxed)",
		description: "Create or overwrite a file inside the policy's FS sandbox. Refused outside it. Returns {ok} or {ok:false, refused}.",
		parameters: Type.Object({
			path: Type.String({ description: "File path, relative to the sandbox root" }),
			content: Type.String({ description: "The complete new file content" }),
		}),
		async execute(_id, params) {
			if (gate.refusal) return refusedResult();
			return call({ op: "write", path: params.path, content: params.content });
		},
	});

	pi.registerTool({
		name: "ailang_edit",
		label: "Edit (sandboxed)",
		description: "Replace old_text with new_text in a file inside the sandbox; old_text must occur exactly once (include enough context). Returns {ok} or {ok:false, refused}.",
		parameters: Type.Object({
			path: Type.String({ description: "File path, relative to the sandbox root" }),
			old_text: Type.String({ description: "Exact text to replace (must occur exactly once)" }),
			new_text: Type.String({ description: "Replacement text" }),
		}),
		async execute(_id, params) {
			if (gate.refusal) return refusedResult();
			return call({ op: "edit", path: params.path, old_text: params.old_text, new_text: params.new_text });
		},
	});

	// The rest of the ailang CLI, as typed requests. The model never composes
	// an argv: it names an op and its fields; Go validates each field against
	// the op's schema and builds the argv inside the sandbox.
	const ops = summary?.cli ?? [];
	pi.registerTool({
		name: "ailang_cli",
		label: "AILANG CLI (policy-allowlisted)",
		description:
			`Run an allowlisted ailang operation as a typed request. ops under this policy: ${ops.join(", ") || "(none)"}. ` +
			"Field by op — check/ai_check/fmt/tree/policy_check: {path}; iface/pkg_docs: {module}; docs_search/examples_search: {query}; " +
			"examples_show/builtins_show: {module: <name>}; test: {path?, package?}; flags: {\"json\": \"\", \"limit\": \"5\"} only where the op admits them. " +
			"NOT run: execution only goes through ailang_run. Returns {ok, argv, exit_code, stdout, stderr} or {ok:false, refused}.",
		parameters: Type.Object({
			op: Type.String({ description: 'The operation, e.g. "check", "iface", "docs_search", "test"' }),
			path: Type.Optional(Type.String({ description: "In-sandbox file path (check, ai_check, fmt, tree, policy_check, test)" })),
			module: Type.Optional(Type.String({ description: 'Module path or name (iface: "std/fs"; examples_show/builtins_show: a name)' })),
			query: Type.Optional(Type.String({ description: "Search text (docs_search, examples_search)" })),
			package: Type.Optional(Type.String({ description: "Package directory for test (in-sandbox)" })),
			flags: Type.Optional(Type.Record(Type.String(), Type.String(), { description: 'Admitted flags by name; boolean flags take "" (e.g. {"json": ""})' })),
		}),
		async execute(_id, params) {
			if (gate.refusal) return refusedResult();
			return call(cliRequest(params));
		},
	});
}

// The minimal shape of typebox's Type this file uses; keeps tests free of the package.
type TypeLike = {
	Object: (props: Record<string, unknown>) => unknown;
	String: (opts?: Record<string, unknown>) => unknown;
	Array: (item: unknown, opts?: Record<string, unknown>) => unknown;
	Record: (key: unknown, value: unknown, opts?: Record<string, unknown>) => unknown;
	Optional: (schema: unknown) => unknown;
};

export default async function (pi: ExtensionAPI) {
	const { Type } = await import("typebox");
	return register(pi, Type as unknown as TypeLike);
}
