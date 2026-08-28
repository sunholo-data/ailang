/**
 * provider-quota — M-DX-PI-HARNESS addendum (M5)
 * Report provider budget/quota headroom before a session burns it:
 *   - OpenRouter: GET /api/v1/key → usage vs limit (CRITICAL ≥95%, WARN ≥80%)
 *   - Ollama Cloud: GET https://ollama.com/api/usage (Bearer OLLAMA_API_KEY) —
 *     measured contract in design doc m-ollama-cloud-provider (V24/V26/V36/V50):
 *     coarse numerator, NO published denominator, so % remaining is NOT computable
 *   - Current session lane via ctx.model
 * Never exposes the API key. Subprocess-free (fetch + 10s aborts).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export type QuotaStatus = "OK" | "WARN" | "CRITICAL" | "UNLIMITED" | "UNKNOWN";

/** Pure: classify usage against a limit. Thresholds: ≥80 WARN, ≥95 CRITICAL. */
export function classifyQuota(
	usage: number | null,
	limit: number | null,
): { pct: number; remaining: number | null; status: QuotaStatus } {
	if (limit === null || limit === undefined || !isFinite(limit) || limit <= 0) {
		if (usage === null || usage === undefined || !isFinite(usage)) {
			return { pct: 0, remaining: null, status: "UNKNOWN" };
		}
		return { pct: 0, remaining: null, status: "UNLIMITED" };
	}
	if (usage === null || usage === undefined || !isFinite(usage)) {
		return { pct: 0, remaining: limit, status: "UNKNOWN" };
	}
	const pct = (usage / limit) * 100;
	const status: QuotaStatus = pct >= 95 ? "CRITICAL" : pct >= 80 ? "WARN" : "OK";
	return { pct, remaining: limit - usage, status };
}

async function openRouterSummary(): Promise<string> {
	const key = process.env.OPENROUTER_API_KEY;
	if (!key) return "OPENROUTER_API_KEY not set in this session";
	try {
		const resp = await fetch("https://openrouter.ai/api/v1/key", {
			headers: { Authorization: `Bearer ${key}` },
			signal: AbortSignal.timeout(10_000),
		});
		const data = ((await resp.json()) as { data?: { usage?: number; limit?: number | null } }).data ?? {};
		const q = classifyQuota(
			typeof data.usage === "number" ? data.usage : null,
			typeof data.limit === "number" ? data.limit : null,
		);
		const usage = typeof data.usage === "number" ? data.usage : 0;
		const remain = q.remaining !== null ? `, $${q.remaining.toFixed(2)} left` : "";
		return `${q.status} — $${usage.toFixed(2)} of $${data.limit ?? "∞"} (${q.pct.toFixed(1)}%)${remain}`;
	} catch (e) {
		return `query failed (${String(e)})`;
	}
}

/**
 * Ollama Cloud usage gauge — GET https://ollama.com/api/usage (Bearer OLLAMA_API_KEY).
 * Measured contract (design doc m-ollama-cloud-provider, V24/V26/V36/V50):
 *   - NOT proxied by the local daemon (localhost:11434/api/usage → 404); direct call required
 *   - shape: activity{cost, period{type,starting_at,ending_at}, models[{name,request_count}]}
 *     + limits{session{usage,models[]}, weekly{usage,models[]}}
 *   - usage is a COARSE numerator (stayed 0 across ~6k tokens); limits publish NO
 *     denominator — % remaining is NOT computable; measured burn: ~0.007–0.124 units
 *     per M tokens across weight classes (V36/V46/V50)
 * Inference does NOT need this key (device key via `ollama signin`); the API key comes
 * from ollama.com settings. Without it we say so rather than pretending.
 */
async function ollamaCloudUsage(): Promise<string> {
	const key = process.env.OLLAMA_API_KEY;
	if (!key) {
		return "consumption gauge needs OLLAMA_API_KEY (ollama.com → settings → API keys); device-key inference does not require it";
	}
	try {
		const resp = await fetch("https://ollama.com/api/usage", {
			headers: { Authorization: `Bearer ${key}` },
			signal: AbortSignal.timeout(10_000),
		});
		if (!resp.ok) {
			return `HTTP ${resp.status} from /api/usage`;
		}
		const d = (await resp.json()) as {
			activity?: { cost?: string; period?: { ending_at?: string } };
			limits?: { session?: { usage?: number }; weekly?: { usage?: number } };
		};
		const sess = d.limits?.session?.usage;
		const week = d.limits?.weekly?.usage;
		const cost = d.activity?.cost ?? "?";
		const periodEnd = d.activity?.period?.ending_at ?? "?";
		return `session usage ${sess ?? "?"}, weekly usage ${week ?? "?"} (coarse numerator — denominator unpublished, % remaining NOT computable); activity cost ${cost}, period ends ${periodEnd}`;
	} catch (e) {
		return `query failed (${String(e)})`;
	}
}

export default async function (pi: ExtensionAPI) {
	const { Type } = await import("typebox");

	pi.registerTool({
		name: "quota_report",
		label: "Provider Quota Report",
		description:
			"Report provider budget/quota headroom: OpenRouter key usage vs limit (CRITICAL >=95%, WARN >=80%), " +
			"Ollama Cloud usage (session/weekly numerators), and which provider the CURRENT session runs on. " +
			"Call before long tasks or when provider errors mention limits. Never exposes the API key.",
		parameters: Type.Object({}),
		async execute(_id, _params, _signal, _onUpdate, ctx) {
			const current = ctx.model as { provider?: string; id?: string } | undefined;
			const or = await openRouterSummary();
			const oc = await ollamaCloudUsage();
			const lane = current?.provider ? `${current.provider}/${current.id ?? "?"}` : "?";
			const text = `current session: ${lane}\nopenrouter: ${or}\nollama cloud: ${oc}`;
			return { content: [{ type: "text", text }], details: { openrouter: or, ollamaCloud: oc } };
		},
	});

	pi.registerCommand("quota", {
		description: "Provider quota: OpenRouter budget + Ollama Cloud usage + current session lane",
		handler: async (_args, ctx) => {
			const current = ctx.model as { provider?: string; id?: string } | undefined;
			const or = await openRouterSummary();
			const oc = await ollamaCloudUsage();
			const lane = current?.provider ? `${current.provider}/${current.id ?? "?"}` : "?";
			const text = `current session: ${lane}\nopenrouter: ${or}\nollama cloud: ${oc}`;
			await ctx.ui.notify(text, /CRITICAL|WARN|failed|not set|HTTP [45]/.test(text) ? "warning" : "info");
		},
	});
}