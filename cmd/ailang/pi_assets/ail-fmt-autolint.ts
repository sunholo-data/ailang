/**
 * ail-fmt-autolint — M-DX-PI-HARNESS addendum (motoko port, measured-first)
 *
 * Port of motoko's `fmt` extension profile: after a successful write/edit of a
 * .ail file, run `ailang fmt --write` so every AILANG file an agent saves is
 * canonically formatted. motoko measured this technique in A/B eval arms;
 * the repo's Claude Code shells already get it via .claude/ fmt hooks — this
 * makes parity in pi.
 *
 * Prepush note: gofmt-formatted Go + fmt-formatted .ail both pass; the prepush
 * gate and this extension are complementary layers of the same doctrine.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export function isAilPath(path: string | undefined): boolean {
	return typeof path === "string" && /\.ail$/i.test(path.trim());
}

export default function (pi: ExtensionAPI) {
	// path of the most recent write/edit, keyed by toolCallId
	const pending = new Map<string, string>();

	pi.on("tool_call", async (event, ctx) => {
		if (event.toolName !== "write" && event.toolName !== "edit") return;
		const p = (event.input as { path?: string } | undefined)?.path;
		if (isAilPath(p)) {
			const abs = p && p.startsWith("/") ? p : `${ctx.cwd}/${p}`;
			pending.set(event.toolCallId, abs);
		}
		return; // never blocks — fmt is a post-save nicety
	});

	pi.on("tool_execution_end", async (event, _ctx) => {
		const path = pending.get(event.toolCallId ?? "");
		pending.delete(event.toolCallId ?? "");
		if (!path || event.isError) return;
		if (event.toolName !== "write" && event.toolName !== "edit") return;
		try {
			const r = await pi.exec("ailang", ["fmt", "--write", path], { timeout: 15_000 });
			// Silent on success: formatted output is the point, not the notification.
			// Surface only genuine failures so a broken fmt install is visible.
			if (r.code !== 0) {
				await _ctx?.ui?.notify?.(
					`[ail-fmt] ailang fmt --write ${path} exited ${r.code}: ${(r.stderr ?? "").slice(0, 200)}`,
					"warning",
				);
			}
		} catch (e) {
			await _ctx?.ui?.notify?.(`[ail-fmt] failed: ${String(e).slice(0, 200)}`, "warning");
		}
		void event;
	});
}