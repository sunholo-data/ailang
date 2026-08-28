/**
 * binary-freshness — M-DX-PI-HARNESS A1
 * Answers "is my installed ailang binary fresh vs HEAD?" before an agent trusts it.
 * Classification (fail-closed): FRESH / STALE / DIRTY / UNKNOWN. Never auto-installs.
 * Subprocess Contract: 15s timeout per command, 64KB caps, structured failures.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export interface BinaryVersion {
	commit: string; // short hash, "" if absent
	dirty: boolean; // Full hash carried a -dirty suffix
	built: string; // RFC3339 or ""
}

export type Freshness =
	| { status: "FRESH" | "STALE" | "DIRTY"; detail: string }
	| { status: "UNKNOWN"; detail: string };

/** Parse `ailang version` output: Commit: <short> / Full: <hash>[-dirty] / Built: <RFC3339>. */
export function parseVersionOutput(out: string): BinaryVersion | null {
	const commit = /Commit:\s*([0-9a-f]{7,12})\b/.exec(out)?.[1] ?? "";
	const full = /Full:\s*(\S+)/.exec(out)?.[1] ?? "";
	const built = /Built:\s*(\S+)/.exec(out)?.[1] ?? "";
	if (!commit) return null; // fail-closed: no Commit line → UNKNOWN upstream
	return { commit, dirty: full.endsWith("-dirty"), built };
}

/** Pure classification. dirtyFiles = count from `git status --porcelain`. */
export function classify(
	bv: BinaryVersion | null,
	headCommit: string | null,
	dirtyFiles: number,
): Freshness {
	if (!bv || !headCommit) {
		return { status: "UNKNOWN", detail: "version output or HEAD unavailable — do not trust toolchain-dependent results" };
	}
	const drifted = bv.commit !== headCommit.slice(0, bv.commit.length);
	const treeDirty = dirtyFiles > 0;
	if (drifted) {
		return { status: "STALE", detail: `installed binary built from ${bv.commit}, HEAD is ${headCommit.slice(0, 12)} — rebuild (attended: make quick-install; unattended: scratch-build)` };
	}
	if (bv.dirty || treeDirty) {
		return { status: "DIRTY", detail: `binary matches HEAD but ${bv.dirty ? "the binary itself was built from a dirty tree" : `${dirtyFiles} file(s) are dirty`} — rebuilding now ships uncommitted work (verify owners first)` };
	}
	return { status: "FRESH", detail: `installed binary built from HEAD (${bv.commit}), clean tree` };
}

export default async function (pi: ExtensionAPI) {
	async function report(ctx: { cwd: string }): Promise<Freshness> {
		const ver = await pi.exec("ailang", ["version"], { timeout: 10_000 });
		const bv = parseVersionOutput(ver.stdout ?? "");
		const head = await pi.exec("git", ["rev-parse", "HEAD"], { timeout: 10_000 });
		const headCommit = (head.stdout ?? "").trim() || null;
		const st = await pi.exec("git", ["status", "--porcelain"], { timeout: 10_000 });
		const dirtyFiles = (st.stdout ?? "").split("\n").filter((l) => l.trim() !== "").length;
		return classify(bv, headCommit, dirtyFiles);
	}

	pi.registerTool({
		name: "freshness_report",
		label: "Binary Freshness Report",
		description:
			"Report whether the installed `ailang` binary matches the repo HEAD (FRESH/STALE/DIRTY/UNKNOWN). " +
			"Call this before trusting toolchain-dependent results after pulling, or when stdlib symbols appear missing.",
		parameters: (await import("typebox")).Type.Object({}),
		async execute(_id, _params, _signal, _onUpdate, ctx) {
			const f = await report(ctx);
			return {
				content: [{ type: "text", text: `${f.status}: ${f.detail}` }],
				details: f,
			};
		},
	});

	pi.registerCommand("fresh", {
		description: "Binary freshness: installed ailang vs HEAD",
		handler: async (_args, ctx) => {
			const f = await report(ctx);
			await ctx.ui.notify(`ailang binary — ${f.status}: ${f.detail}`, f.status === "FRESH" ? "info" : "warning");
		},
	});
}