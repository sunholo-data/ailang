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
): string | null {
	if (acked) return null;
	if (toolName === "edit" || toolName === "write") return BLOCK_REASON;
	if (toolName === "bash") {
		return bashAllowed(bashCommand) ? null : BLOCK_REASON;
	}
	return null; // read-only and other tools are never blocked
}

/** Does the session branch show evidence of the protocol's observable steps? */
export function headlessPrerequisitesMet(branch: unknown[]): {
	met: boolean;
	missing: string[];
} {
	let claudeMdRead = false;
	let messagesChecked = false;
	for (const entry of branch) {
		const msg = (entry as { message?: { role?: string; content?: unknown } })
			.message;
		if (!msg || msg.role !== "assistant" || !Array.isArray(msg.content)) {
			continue;
		}
		for (const part of msg.content as Array<{
			type?: string;
			name?: string;
			arguments?: { path?: string; command?: string };
		}>) {
			if (part?.type !== "toolCall") continue;
			if (part.name === "read" && typeof part.arguments?.path === "string") {
				if (part.arguments.path.includes("CLAUDE.md")) claudeMdRead = true;
			}
			if (part.name === "bash" && typeof part.arguments?.command === "string") {
				if (part.arguments.command.includes("ailang messages")) {
					messagesChecked = true;
				}
				if (part.arguments.command.includes("CLAUDE.md")) claudeMdRead = true;
			}
		}
	}
	const missing: string[] = [];
	if (!claudeMdRead) missing.push("read CLAUDE.md (a read of CLAUDE.md in this session)");
	if (!messagesChecked)
		missing.push("run `ailang messages list --unread` and summarize to the user");
	return { met: missing.length === 0, missing };
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

	// The gate: block mutating tools while armed.
	pi.on("tool_call", async (event, _ctx) => {
		const reason = shouldBlock(
			event.toolName,
			event.toolName === "bash"
				? (event.input as { command?: string } | undefined)?.command
				: undefined,
			acked,
		);
		if (reason) return { block: true, reason };
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