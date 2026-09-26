/**
 * ailang-lsp-lite — M-DX-PI-HARNESS B1+B2
 * Stream B tools for writing AILANG: structured diagnostics without bash parsing,
 * and filtered builtin inventory without the firehose. Both wrap the real `ailang`
 * CLI under the Subprocess Contract (timeouts, caps, structured failures).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { spawn } from "node:child_process";

export interface Diagnostic {
	code: string; // IMP010 | TCxxx | MODxxx | EFF_* | E_* | PARxxx | UNKNOWN
	severity: "error" | "warning";
	message: string;
	file: string;
	line: number | null;
	col: number | null;
	hint: string; // trailing "— …" clause (e.g. builtinHint), "" when absent
}

const DIAG_RE = /^(Error|Warning)(?:\s*\(([^)]*)\))?:\s*(.+)$/gm;

/** Extract a leading ALL-CAPS error code ("IMP010:", "E_AI_TYPE_ERROR:") from a message body. */
function leadingCode(rest: string): { code: string; remainder: string } {
	const m = /^([A-Z][A-Z0-9_]*):\s*/.exec(rest);
	if (!m) return { code: "", remainder: rest };
	return { code: m[1], remainder: rest.slice(m[0].length) };
}

/**
 * Pure: parse `ailang check` output into structured diagnostics.
 * Handles the live shapes: `Error: type error in m (decl N): MESSAGE at FILE:LINE:COL — hint`
 * and `Error: IMP010: symbol ... — hint`, plus bare `Warning: ...` lines (V3).
 */
export function parseCheckOutput(out: string): Diagnostic[] {
	const diags: Diagnostic[] = [];
	DIAG_RE.lastIndex = 0;
	let m: RegExpExecArray | null;
	while ((m = DIAG_RE.exec(out)) !== null) {
		const severity = m[1].toLowerCase() as "error" | "warning";
		let rest = m[3] ?? "";
		// (1) hint first: "— hint" trails the span in the live format
		const dash = rest.indexOf("—");
		const hint = dash >= 0 ? rest.slice(dash + 1).trim() : "";
		if (dash >= 0) rest = rest.slice(0, dash).trim();
		// (2) code from the leading ALL-CAPS token
		const lc = leadingCode(rest);
		const code = (m[2] ?? "").trim() || lc.code;
		rest = lc.remainder;
		// (3) span at end of the pre-hint part
		const at = /\bat\s+(\S+?):(\d+):(\d+)\s*$/.exec(rest);
		let file = "";
		let line: number | null = null;
		let col: number | null = null;
		if (at) {
			file = at[1];
			line = Number(at[2]);
			col = Number(at[3]);
			rest = rest.slice(0, at.index).trim();
		}
		const message = rest.trim();
		diags.push({ code: code || "UNKNOWN", severity, message, file, line, col, hint });
	}
	return diags;
}

export interface BuiltinEntry {
	name: string;
	module: string;
	signature: string;
	[k: string]: unknown;
}

/**
 * Pure: the inventory from `ailang builtins list --json`, whichever shape the
 * binary emits. It is `{count, builtins: [...]}` today; the extension was
 * written against a bare array and every builtins_search call on the
 * ailang_only lane answered "entries.filter is not a function" (measured
 * 2026-09-16 on the first six Jobs tasks). Both shapes are accepted; anything
 * else is an error the caller can read, never an empty inventory.
 */
export function parseBuiltinsInventory(stdout: string): BuiltinEntry[] {
	const parsed = JSON.parse(stdout) as unknown;
	if (Array.isArray(parsed)) return parsed as BuiltinEntry[];
	if (parsed && typeof parsed === "object" && Array.isArray((parsed as { builtins?: unknown }).builtins)) {
		return (parsed as { builtins: BuiltinEntry[] }).builtins;
	}
	throw new Error("builtins inventory is neither an array nor {builtins: [...]}");
}

/** Pure: filter inventory by query/module, capped for context economy. */
export function filterBuiltins(
	entries: BuiltinEntry[],
	query: string | undefined,
	module: string | undefined,
	cap = 10,
): BuiltinEntry[] {
	const q = (query ?? "").toLowerCase();
	const mod = (module ?? "").toLowerCase();
	const hits = entries.filter((e) => {
		if (mod && !String(e.module ?? "").toLowerCase().includes(mod)) return false;
		if (!q) return true;
		return [e.name, e.module, e.signature, e.description]
			.some((v) => String(v ?? "").toLowerCase().includes(q));
	});
	return q ? hits.slice(0, cap) : hits; // unfiltered inventory is the caller's explicit choice
}

/** `ailang policy-tool` op=check: the path is validated in Go against the sandbox root. */
export function policyToolCheck(policyPath: string, path: string): Promise<{ ok: boolean; refused?: string; stdout?: string; stderr?: string }> {
	return new Promise((resolveP) => {
		const child = spawn("ailang", ["policy-tool", "--policy", policyPath], { stdio: ["pipe", "pipe", "pipe"] });
		let out = "";
		let err = "";
		const timer = setTimeout(() => child.kill("SIGKILL"), 40_000);
		child.stdout.on("data", (d) => (out += String(d)));
		child.stderr.on("data", (d) => (err += String(d)));
		child.on("error", (e) => { clearTimeout(timer); resolveP({ ok: false, refused: `policy-tool could not start: ${e.message}` }); });
		child.on("close", (code) => {
			clearTimeout(timer);
			try { resolveP(JSON.parse(out)); } catch { resolveP({ ok: false, refused: `policy-tool produced no response (exit ${code}): ${err.trim().slice(0, 2000)}` }); }
		});
		child.stdin.end(JSON.stringify({ op: "check", path }));
	});
}

export default async function (pi: ExtensionAPI) {
	const { Type } = await import("typebox");

	pi.registerTool({
		name: "ailang_check",
		label: "AILANG Check",
		description:
			"Type-check an AILANG file (`ailang check <path>`) and return STRUCTURED diagnostics " +
			"({code, message, file, line, col, hint}) — use this instead of bash-grepping check output.",
		parameters: Type.Object({
			path: Type.String({ description: "Path to the .ail file" }),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			void ctx;
			// Under an attached policy (the ailang_only lane) the check goes
			// through `ailang policy-tool`, which validates the path against the
			// sandbox root in Go and builds the argv itself — this extension
			// never hands an unchecked path to the CLI (M-EXECUTOR-POLICY-
			// HARDENING M4). Without a policy (a plain local pi session) the
			// direct check is unchanged.
			const policyPath = process.env.AILANG_AGENT_POLICY?.trim();
			if (policyPath) {
				const r = await policyToolCheck(policyPath, params.path);
				if (!r.ok && r.refused) {
					return { content: [{ type: "text", text: JSON.stringify({ ok: false, refused: r.refused, diagnostics: [] }) }], details: { ok: false, refused: r.refused, diagnostics: [] } };
				}
				const diagnostics = parseCheckOutput(`${r.stderr ?? ""}\n${r.stdout ?? ""}`);
				return { content: [{ type: "text", text: JSON.stringify({ ok: r.ok, diagnostics }) }], details: { ok: r.ok, diagnostics } };
			}
			// Run IN the file's directory with the bare filename, the same way
			// ailang_run does: `module x` for x.ail is then canonical wherever the
			// file lives (absolute paths trip MOD010; e2e 2026-08-28, and the
			// ailang_only Jobs batch 2026-09-16).
			const { basename, dirname, resolve } = await import("node:path");
			const abs = resolve(params.path);
			const r = await pi.exec("ailang", ["check", basename(abs)], { timeout: 30_000, cwd: dirname(abs) });
			const output = `${r.stderr ?? ""}\n${r.stdout ?? ""}`;
			const diagnostics = parseCheckOutput(output);
			return {
				content: [{ type: "text", text: JSON.stringify({ ok: r.code === 0, diagnostics }) }],
				details: { ok: r.code === 0, diagnostics },
			};
		},
	});

	pi.registerTool({
		name: "builtins_search",
		label: "Search AILANG builtins",
		description:
			"Search the REAL builtin inventory (compiled into the binary). Pass query (substring over " +
			"name/module/signature/description) and/or module (e.g. 'std/fs'); omit both for the full list. " +
			"Use this instead of guessing builtin names.",
		parameters: Type.Object({
			query: Type.Optional(Type.String()),
			module: Type.Optional(Type.String()),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			const r = await pi.exec("ailang", ["builtins", "list", "--json"], { timeout: 15_000 });
			let entries: BuiltinEntry[] = [];
			try {
				entries = parseBuiltinsInventory(r.stdout ?? "");
			} catch (e) {
				return {
					content: [{ type: "text", text: `builtins inventory unparseable (${String(e)}); ailang exit ${r.code}` }],
					details: { error: "inventory-unparseable" },
				};
			}
			const matches = filterBuiltins(entries, params.query, params.module);
			return {
				content: [{ type: "text", text: JSON.stringify({ count: matches.length, matches }) }],
				details: { count: matches.length, matches },
			};
		},
	});
}