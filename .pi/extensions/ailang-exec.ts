/**
 * ailang-exec — M-AGENT-AILANG-ONLY-EXECUTION M3
 *
 * The ONE way an `ailang_only` agent executes a program: `ailang_run` submits
 * an .ail file to `ailang run --policy $AILANG_AGENT_POLICY`. The policy — not
 * the agent — decides caps, the Net allowlist and the FS sandbox; the gate
 * refuses every flag that could widen them (cmd/ailang/run_policy.go). Pair
 * with `--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run`
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
import { dirname, resolve, sep } from "node:path";

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
			const args = ["run", "--policy", gate.policyPath as string];
			if (params.args_json) args.push("--args-json", params.args_json);
			args.push(params.path);
			const r = await pi.exec("ailang", args, { timeout: 120_000 });
			const env = composeEnvelope(r.code ?? -1, r.stdout ?? "", r.stderr ?? "");
			return { content: [{ type: "text", text: JSON.stringify(env) }], details: env };
		},
	});
}
