/**
 * Session Protocol Gate — M-DX-SESSION-GATE (v0.35.0)
 *
 * Mechanically backs the AGENTS.md work-routing gate for pi sessions in this
 * repo. While armed, `edit`/`write` are blocked absolutely and `bash` is
 * fail-closed to a read-only allowlist until `session_protocol_ack` succeeds.
 *
 * Ack prerequisites (design doc: m-dx-session-protocol-gate.md):
 *   - Interactive (ctx.hasUI): ctx.ui.confirm — a real human keypress (V11).
 *   - Headless: session history must show the protocol's observable steps —
 *     a tool call touching CLAUDE.md AND a bash call running `ailang messages`
 *     (V10 getBranch reconstruction pattern).
 *
 * Fail-open register (F1–F8) and platform-claim verification log live in
 * design_docs/planned/v0_35_0/m-dx-session-protocol-gate.md.
 * Tested against pi 0.84.3. Distribution: git (project-local extension).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export const BLOCK_REASON =
	"Session protocol not completed — call session_protocol_ack after reading CLAUDE.md and checking ailang messages. Feature/semantics work additionally requires an approved design doc and sprint plan.";

/**
 * Read-only bash allowlist (fail-closed). While armed, a bash command runs
 * only if EVERY segment (split on && ; ||) matches. Unlisted commands —
 * including interpreter one-liners like `python -c 'open(…)'` — are blocked,
 * never silently passed (round-2 quorum fix: denylist was a silent fallback).
 */
const BASH_ALLOW: RegExp[] = [
	/^(ls|cat|head|tail|grep|rg|find|wc|stat|file)\b/,
	/^git (status|log|diff|show|branch)\b/,
	/^ailang (check|messages|doctor|builtins|docs|prompt)\b/,
];

export function bashAllowed(command: string | undefined): boolean {
	if (typeof command !== "string" || command.trim() === "") {
		return false; // fail-closed: no/empty command is not provably read-only
	}
	// Split compound commands; every segment must be independently allowlisted
	// so `cat file && rm file` cannot smuggle a write through a read head.
	const segments = command.split(/&&|\|\||;|\|/);
	return segments.every((seg) => {
		const s = seg.trim();
		if (s === "") return true; // empty segment from trailing separator
		return BASH_ALLOW.some((re) => re.test(s));
	});
}

/**
 * Pure gate predicate — returns the block reason or null to allow.
 * Table-driven unit tests: .session-protocol-gate.test.ts
 */
export function shouldBlock(
	toolName: string,
	bashCommand: string | undefined,
	acked: boolean,
	gate?: GateContext,
): string | null {
	if (acked) return null;

	// Tier 1 is the FLOOR, not a replacement for the ack: routine work unlocks
	// once the prerequisites are verifiably met, because requiring a separate
	// self-attested call deadlocked executors that had already done the
	// protocol (design doc V6 — the model satisfied both prerequisites in three
	// turns and then sat blocked for twelve minutes).
	//
	// Tier 2 has NO auto-path. Anything not exactly "tier1" is tier 2, so a
	// missing, malformed or unrecognised context fails closed.
	if (gate?.tier === "tier1" && gate.prereqsMet === true) return null;

	if (toolName === "edit" || toolName === "write") return BLOCK_REASON;
	if (toolName === "bash") {
		return bashAllowed(bashCommand) ? null : BLOCK_REASON;
	}
	return null; // read-only and other tools are never blocked
}

/**
 * Commit attribution (M-DX-PI-HARNESS addendum): the pi-analogue of Claude Code's
 * Co-Authored-By convention. Lives in the GATE (not a separate extension) because the
 * gate's tool_call handler is the single guaranteed observer of every bash call — a
 * separate extension's handler can be skipped when the gate blocks (runner.js returns
 * on first block).
 */
export const ATTR_EMAIL = "noreply@ailang.dev";

export function attributionTrailer(model: { provider?: string; id?: string } | undefined | null): string {
	if (model?.provider && model?.id) {
		return `Co-Authored-By: pi (${model.provider}/${model.id}) <${ATTR_EMAIL}>`;
	}
	return `Co-Authored-By: pi-coding-agent <${ATTR_EMAIL}>`;
}

export function hasAttribution(command: string): boolean {
	return /Co-Authored-By:|Assisted-By:/i.test(command);
}

function hasDynamicMessage(command: string): boolean {
	return /\$\(|`/.test(command.split(/(?:-m|--message)\s/).pop() ?? "");
}

/**
 * Pure: append the trailer to the LAST -m/--message argument of a `git commit` command.
 * Returns patched/original command, or null when this isn't a commit (not our business).
 * Conservative: skips command-substitution messages and editor-flow commits (notify path).
 */
export function appendAttribution(
	command: string,
	trailer: string,
): { command: string; changed: boolean; skipped?: string } | null {
	if (!/\bgit\b[^&|;]*\bcommit\b/.test(command)) return null;
	if (hasAttribution(command)) return { command, changed: false, skipped: "already-attributed" };
	if (hasDynamicMessage(command)) {
		return { command, changed: false, skipped: "dynamic-message" };
	}
	const re = /(?:^|\s)(?:--message(?:\s*=\s*|\s+)|-[a-zA-Z]*m[a-zA-Z]*(?:\s*=\s*|\s+))(?:"([^"]*)"|'([^']*)'|([^\s"'][^\s]*))/g;
	let last: { full: string; body: string; quote: '"' | "'" | "" } | null = null;
	let m: RegExpExecArray | null;
	while ((m = re.exec(command)) !== null) {
		last = {
			full: m[0],
			body: m[1] ?? m[2] ?? m[3] ?? "",
			quote: m[1] !== undefined ? '"' : m[2] !== undefined ? "'" : "",
		};
	}
	if (!last || last.body.trim() === "") return { command, changed: false, skipped: "editor-flow" };
	const patched = command.replace(
		last.full,
		last.full.slice(0, last.full.length - (last.quote ? 1 : 0)) +
			`\n\n${trailer}` +
			(last.quote ?? ""),
	);
	return { command: patched, changed: true };
}

/**
 * Which built-in prerequisite set governs a workspace.
 *
 * M-COORDINATOR-EXECUTION-TRUST M1a. Two sets, both built in — deliberately no
 * repo-published content. A repo that could declare its own protocol could
 * declare an empty one and unlock itself; that whole surface is split out to
 * M-PACKAGE-PROTOCOL-MANIFESTS, where it can be designed with the add-only
 * bound it needs.
 *
 *   generic — the floor. Satisfiable with no CLAUDE.md and no AILANG-specific
 *             file, so a package repo the executor has never seen can pass it.
 *   ailang  — the floor PLUS a CLAUDE.md read: today's behaviour, unchanged,
 *             for the repo whose convention it actually is.
 *
 * The AILANG set is strictly stronger than the floor. That relation is asserted
 * by a test, and it is the shape a future manifest must also take: sets may ADD
 * to the floor, never subtract from it.
 */
export type PrereqSet = "generic" | "ailang";

/**
 * The workspace identity comes from the COORDINATOR (the agent config's repo /
 * workspace field), never from a file inside the clone. A repo cannot elect
 * which set governs it.
 */
export function prereqSetForWorkspace(workspace: string | undefined): PrereqSet {
	if (typeof workspace !== "string") return "generic";
	const repo = workspace.trim().toLowerCase().replace(/\.git$/, "");
	return repo === "sunholo-data/ailang" ? "ailang" : "generic";
}

/**
 * The dispatch context, read from environment the COORDINATOR sets — never from
 * anything inside the cloned workspace. `AILANG_WORK_TIER` is written by the
 * Cloud Run dispatcher from ResolveWorkTier (internal/coordinator/work_tier.go),
 * which reads the agent registry and refuses tier 1 on a direct-push dispatch.
 *
 * Anything other than exactly "tier1" is tier 2, so an absent, empty, misspelled
 * or injected value fails closed.
 */
export function dispatchTierFromEnv(env?: Record<string, string | undefined>): "tier1" | "tier2" {
	const e = env ?? (typeof process !== "undefined" ? process.env : undefined);
	return e?.AILANG_WORK_TIER === "tier1" ? "tier1" : "tier2";
}

/** Permission context for one dispatch, supplied by the coordinator. */
export interface GateContext {
	tier: "tier1" | "tier2";
	prereqsMet: boolean;
}

/**
 * Does the session branch show evidence of the protocol's observable steps?
 *
 * `set` selects which steps are required. Both sets need the two floor steps:
 * some orientation in the workspace, and an inbox check — the latter being the
 * AILANG-general part, since the message plane is what the whole ecosystem
 * shares.
 */
export function prerequisitesMet(
	branch: unknown[],
	set: PrereqSet = "ailang",
): { met: boolean; missing: string[] } {
	let oriented = false;
	let messagesChecked = false;
	let claudeMdRead = false;

	for (const entry of branch) {
		const msg = (entry as { message?: { role?: string; content?: unknown } }).message;
		if (!msg || msg.role !== "assistant" || !Array.isArray(msg.content)) continue;

		for (const part of msg.content as Array<{
			type?: string;
			name?: string;
			arguments?: { path?: string; command?: string };
		}>) {
			if (part?.type !== "toolCall") continue;

			if (part.name === "read" && typeof part.arguments?.path === "string") {
				oriented = true;
				if (part.arguments.path.includes("CLAUDE.md")) claudeMdRead = true;
			}
			if (part.name === "bash" && typeof part.arguments?.command === "string") {
				const cmd = part.arguments.command;
				if (cmd.includes("ailang messages")) messagesChecked = true;
				if (cmd.includes("CLAUDE.md")) {
					oriented = true;
					claudeMdRead = true;
				}
			}
		}
	}

	const missing: string[] = [];
	if (!oriented) missing.push("inspect the workspace (a read of a file in it)");
	if (!messagesChecked) missing.push("run `ailang messages list --unread` and summarize to the user");
	if (set === "ailang" && !claudeMdRead) missing.push("read CLAUDE.md (a read of CLAUDE.md in this session)");

	return { met: missing.length === 0, missing };
}

/**
 * Back-compat alias for the AILANG set — the shape the ack tool has always
 * checked in headless mode.
 */
export function headlessPrerequisitesMet(branch: unknown[]): { met: boolean; missing: string[] } {
	return prerequisitesMet(branch, "ailang");
}

export default async function (pi: ExtensionAPI) {
	let acked = false;

	// Reconstruct ack state from the session branch on every (re)start.
	// Fail-closed: any ambiguity re-arms the gate.
	pi.on("session_start", async (_event, ctx) => {
		acked = false;
		try {
			for (const entry of ctx.sessionManager.getBranch()) {
				const msg = (
					entry as {
						message?: { role?: string; toolName?: string; details?: { acked?: boolean } };
					}
				).message;
				if (
					msg?.role === "toolResult" &&
					msg.toolName === "session_protocol_ack" &&
					msg.details?.acked === true
				) {
					acked = true;
				}
			}
		} catch {
			acked = false;
		}
	});

	// The gate: block mutating tools while armed. When a command PASSES the gate
	// and is a git commit lacking attribution, append the trailer here — a single
	// handler sees every bash call, so patching cannot be skipped by the
	// block-short-circuit in pi's multi-handler dispatch (runner.js: returns on
	// first block). M-DX-PI-HARNESS commit-attribution lives here by design.
	pi.on("tool_call", async (event, ctx) => {
		const command = (event.toolName === "bash")
			? (event.input as { command?: string } | undefined)?.command
			: undefined;
		// The gate context is assembled from coordinator-supplied environment and
		// from THIS session's own observable history — never from repo content.
		let gate: GateContext | undefined;
		try {
			const workspace =
				typeof process !== "undefined" ? process.env?.AILANG_WORKSPACE : undefined;
			gate = {
				tier: dispatchTierFromEnv(),
				prereqsMet: prerequisitesMet(
					ctx.sessionManager.getBranch() as unknown[],
					prereqSetForWorkspace(workspace),
				).met,
			};
		} catch {
			gate = undefined; // fail closed to tier-2 semantics
		}

		const reason = shouldBlock(event.toolName, command, acked, gate);
		if (reason) return { block: true, reason };
		// Allowed bash: attribute git commits (best-effort, conservative)
		if (command && event.toolName === "bash") {
			const trailer = attributionTrailer(ctx.model as { provider?: string; id?: string } | undefined);
			const result = appendAttribution(command, trailer);
			if (result?.changed) {
				(event.input as { command: string }).command = result.command;
			}
		}
		return;
	});

	// The disarm tool. Its description IS the protocol — acknowledging
	// requires engaging with it. Prerequisites are verifiable (V10/V11).
	pi.registerTool({
		name: "session_protocol_ack",
		label: "Acknowledge Session Protocol",
		description:
			"Acknowledge completion of the AILANG session protocol, unlocking edit/write/bash-write tools for this session. " +
			"The protocol: (1) read CLAUDE.md; (2) run `ailang messages list --unread` and summarize to the user BEFORE acking; " +
			"(3) classify work via the AGENTS.md Work Routing table — feature/semantics work additionally requires " +
			"design doc -> user approval -> sprint plan -> 'execute sprint'. Calling this without doing the protocol " +
			"violates the repo's work-routing gate.",
		parameters: (await import("typebox")).Type.Object({}),
		async execute(_toolCallId, _params, _signal, _onUpdate, ctx) {
			// Interactive: a human must press confirm (V11).
			if (ctx.hasUI) {
				const confirmed = await ctx.ui.confirm(
					"Complete the session protocol?",
					"Confirm: (1) CLAUDE.md read, (2) ailang messages checked and summarized to the user, " +
						"(3) work classified per AGENTS.md Work Routing (features need approved design doc + sprint plan).",
				);
				if (!confirmed) {
					return {
						content: [
							{
								type: "text",
								text: "REFUSED: the user did not confirm the session protocol. Complete the protocol (read CLAUDE.md, check ailang messages) and try again.",
							},
						],
						details: { acked: false },
					};
				}
			} else {
				// Headless (-p/RPC): verify the protocol's observable steps in session history (V10).
				const { met, missing } = headlessPrerequisitesMet(
					ctx.sessionManager.getBranch() as unknown[],
				);
				if (!met) {
					return {
						content: [
							{
								type: "text",
								text: `REFUSED: session protocol prerequisites not met. Missing: ${missing.join("; ")}.`,
							},
						],
						details: { acked: false, missing },
					};
				}
			}
			acked = true;
			return {
				content: [
					{
						type: "text",
						text: "Session protocol acknowledged — mutating tools unlocked for this session. Work routing per AGENTS.md still applies.",
					},
				],
				details: { acked: true, at: new Date().toISOString() },
			};
		},
	});
}