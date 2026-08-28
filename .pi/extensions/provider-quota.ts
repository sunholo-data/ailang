/**
 * provider-quota — M-DX-PI-HARNESS addendum
 * Report provider quota/budget headroom before a session burns it.
 * OpenRouter: real budget via GET /api/v1/key (documented endpoint). The key
 * itself is never returned. Ollama: local server health; cloud quota has no
 * public API (reported as unknown — check the ollama.com dashboard).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export type QuotaStatus = "OK" | "WARN" | "CRITICAL" | "UNLIMITED" | "UNKNOWN";

/** Pure: classify usage against a limit. Thresholds: >=80 WARN, >=95 CRITICAL. */
export function classifyQuota(
	usage: number | null,
	limit: number | null,
): { pct: number; remaining: number | null; status: "OK" | "WARN" | "CRITICAL" | "UNLIMITED" | "UNKNOWN" } {
	if (limit === null || limit === undefined || !isFinite(limit) || limit <= 0) {
		if (usage === null || usage === undefined || !isFinite(usage)) {
			return { pct: 0, remaining: null, status: "UNKNOWN" };
		}
		return { pct: 0, remaining: null, status: "UNLIMITED" };
	}
	if (usage === null || !isFinite(usage)) {
		return { pct: 0, remaining: limit, status: "UNKNOWN" };
	}
	const pct = (usage / limit) * 100;
	const status = pct >= 95 ? "CRITICAL" : pct >= 80 ? "WARN" : "OK";
	return { pct, remaining: limit - usage, status };
}

/** Pure: the quota summary line for a key payload (never includes the key). */
export function summarizeOpenRouterKey(data: {
	label?: string;
	usage?: number;
	limit?: number | null;
}): string {
	const usage = typeof data.usage === "number" ? data.usage : null;
	const limit = typeof data.limit === "number" ? data.limit : null;
	if (usage === null) return "openrouter: usage unknown (unexpected API response)";
	if (limit === null) return `openrouter: $${usage.toFixed(2)} used, no limit set (UNLIMITED)`;
	const pct = (usage / limit) * 100;
	const status = pct >= 95 ? "CRITICAL" : pct >= 80 ? "WARN" : "OK";
	return `openrouter: ${status} — $${usage.toFixed(2)} of $${limit} (${pct.toFixed(1)}%) — $${(limit - usage).toFixed(2)} left`;
}

export default async function (pi: ExtensionAPI) {
	async function openRouterSummary(): Promise<string> {
		const key = process.env.OPENROUTER_API_KEY;
		if (!key) return "openrouter: OPENROUTER_API_KEY not set in this session";
		try {
			const resp = await fetch("https://openrouter.ai/api/v1/key", {
				headers: { Authorization: `Bearer ${key}` },
				signal: AbortSignal.timeout(10_000),
			});
			const data = ((await resp.json()) as { data?: { usage?: number; limit?: number | null; label?: string } }).data ?? {};
			return summarizeOpenRouterKey(data);
		} catch (e) {
			return `openrouter: query failed (${String(e)})`;
		}
	}

	async function ollamaStatus(): Promise<string> {
		try {
			const v = await fetch("http://localhost:11434/api/version", { signal: AbortSignal.timeout(5000) });
			const body = (await v.json()) as { version?: string };
			return `ollama: local server v${body.version ?? "?"} — cloud quota unknown (no public API; check ollama.com dashboard)`;
		} catch {
			return "ollama: local server not reachable";
		}
	}

	pi.registerTool({
		name: "quota_report",
		label: "Provider Quota Report",
		description:
			"Report provider budget/quota headroom: OpenRouter key usage vs limit (CRITICAL >=95%, WARN >=80%), " +
			"ollama local server status, and which provider the CURRENT session runs on. " +
			"Call before long tasks or when provider errors mention limits. Never exposes the API key.",
		parameters: (await import("typebox")).Type.Object({}),
		async execute(_id, _params, _signal, _onUpdate, ctx) {
			const current = ctx.model as { provider?: string; id?: string } | undefined;
			const or = await openRouterSummary();
			const lines = [
				`current session: ${current?.provider ?? "?"}/${current?.id ?? "?"}`,
				or,
				"ollama: local server checked (cloud quota has no public API — see ollama.com dashboard)",
			];
			return { content: [{ type: "text", text: lines.join("\n") }], details: { openrouter: or } };
		},
	});

	pi.registerCommand("quota", {
		description: "Provider quota: OpenRouter budget + ollama status + current session lane",
		handler: async (_args, ctx) => {
			const current = ctx.model as { provider?: string; id?: string } | undefined;
			const or = await openRouterSummary();
			const lane = current?.provider ? `${current.provider}/${current.id ?? "?"}` : "?";
			await ctx.ui.notify(`current session: ${lane}\n${or}`, /CRITICAL|WARN|failed|not set/.test(or) ? "warning" : "info");
		},
	});
}