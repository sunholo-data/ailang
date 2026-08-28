/**
 * builtin-sprint — M-DX-PI-HARNESS A4
 * /builtin-finish chains the post-builtin ceremony (golden refresh → doctor →
 * inventory count → CHANGELOG pointer) that was previously rediscovered per sprint.
 * Every step honors the Subprocess Contract (timeouts, structured failures).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export interface ChainStep {
	name: string;
	cmd: string;
	args: string[];
	env?: Record<string, string>;
	timeoutMs: number;
}

export interface ChainResult {
	step: string;
	ok: boolean;
	detail: string; // e.g. "exit 0", "TIMEOUT", inventory count
	inventoryCount?: number;
}

/** Pure: the ceremony chain, in order, with Subprocess Contract timeouts. */
export function ceremonySteps(): ChainStep[] {
	return [
		{ name: "golden-refresh", cmd: "go", args: ["test", "./internal/pipeline", "-run", "TestBuiltinTypes_GoldenSnapshot"], env: { UPDATE_GOLDEN: "1" }, timeoutMs: 120_000 },
		{ name: "stdlib-freeze", cmd: "make", args: ["freeze-stdlib"], timeoutMs: 120_000 },
		{ name: "stdlib-verify", cmd: "make", args: ["verify-stdlib"], timeoutMs: 60_000 },
		{ name: "doctor-builtins", cmd: "ailang", args: ["doctor", "builtins"], timeoutMs: 30_000 },
		{ name: "builtins-inventory", cmd: "ailang", args: ["builtins", "list", "--json"], timeoutMs: 15_000 },
	];
}

export default function (pi: ExtensionAPI) {
	pi.registerCommand("builtin-finish", {
		description: "Post-builtin ceremony: golden refresh + doctor builtins + inventory count",
		handler: async (_args, ctx) => {
			const results: ChainResult[] = [];
			for (const step of ceremonySteps()) {
				try {
					const r = await pi.exec(step.cmd, step.args, { timeout: step.timeoutMs });
					if (step.name === "builtins-inventory") {
						let count = -1;
						try {
							count = (JSON.parse(r.stdout ?? "") as unknown[]).length;
						} catch { /* non-JSON output → count unknown */ }
						results.push({ step: step.name, ok: r.code === 0, detail: r.code === 0 ? `inventory: ${count} builtins` : `exit ${r.code}`, inventoryCount: count });
					} else {
						results.push({ step: step.name, ok: r.code === 0 && !r.killed, detail: r.killed ? "TIMEOUT" : `exit ${r.code}` });
					}
				} catch (e) {
					results.push({ step: step.name, ok: false, detail: String(e instanceof Error ? e.message : e) });
				}
			}
			const lines = results.map((r) => `${r.ok ? "✓" : "✗"} ${r.step}: ${r.detail}`);
			lines.push("CHANGELOG: add the entry under changelogs/v0.32-current.md [Unreleased]");
			await ctx.ui.notify(lines.join("\n"), results.every((r) => r.ok) ? "info" : "warning");
		},
	});
}