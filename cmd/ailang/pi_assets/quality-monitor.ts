/**
 * quality-monitor — M-DX-QUALITY-MONITOR (v0.35.0)
 *
 * Makes quality failure modes structured, attributed, and actionable at the pi layer:
 *   Q1  empty/zero-content assistant turns → detected at turn_end, ONE corrective
 *       steer per run (silent no-op when settled), recorded via appendEntry.
 *   Q2  identical tool call repeated 3× consecutively → blocked with a reason that
 *       names the repeat and directs a different action; 3 blocks/run then notify-only.
 *   Q3  oversized tool results → bounded head+tail excerpt (≤ ~2.2KB) + narrowing
 *       directive + details.excerpted provenance — the run continues instead of dying.
 *   Q4  opt-in (PI_QUALITY_THINKING_FALLBACK=1) thinking-budget fallback after ≥2
 *       empty streams for the same model → pi.setThinkingLevel("off") once.
 *
 * Event-only: zero subprocess calls, no session-gate interplay. Detection is
 * bounded (window/strike/cap constants below); steering is capped; D1 attribution
 * discipline: records never claim a model fault without evidence — where spans are
 * unreachable, entries carry the observed facts only ("cause unknown" stays honest).
 *
 * Kill switch: PI_QUALITY_MONITOR=0 → fully inert.
 * Tested against pi 0.84.3/0.84.4. Suite member #9. Distribution: Tier 0–2 via
 * `make pi-assets` (design doc: m-dx-quality-monitor.md, sprint plan: sibling file).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// ---------- Tunables (exported for table-driven tests) ----------

export const EXCERPT_THRESHOLD_BYTES = 16 * 1024; // well under the 64KB subprocess fail-cap
export const EXCERPT_HEAD_BYTES = 1536; // head of stdout (context: what was asked)
export const EXCERPT_TAIL_BYTES = 512; // tail (errors live at the end — 2026-08-26 lesson)
export const LOOP_WINDOW_SIZE = 8; // rolling window of last tool-call hashes (D3)
export const LOOP_STRIKES = 3; // block at 3rd IDENTICAL CONSECUTIVE call
export const LOOP_BLOCK_CAP_PER_RUN = 3; // then notify-only (long-polling may be legitimate)
export const EMPTY_STEER_CAP_PER_RUN = 1; // one corrective message per failure class per run (D2)
export const MAX_LOOP_BLOCKS_PER_SESSION = 20; // hard ceiling beyond per-run caps
export const MAX_STEERS_PER_SESSION = 5; // hard ceiling total corrective messages
export const THINKING_FALLBACK_STRIKES = 2; // empty streams for the SAME model before fallback

export const KILL_SWITCH_ENV = "PI_QUALITY_MONITOR";
export const THINKING_FALLBACK_ENV = "PI_QUALITY_THINKING_FALLBACK";

// ---------- Pure helpers ----------

const RE_TAIL_FFFD = /\uFFFD+$/;
const RE_HEAD_FFFD = /^\uFFFD+/;

const encoder = new TextEncoder();
const decoder = new TextDecoder("utf-8");

export function utf8len(s: string): number {
	return encoder.encode(s).length;
}

/**
 * Canonical JSON with recursively sorted object keys, so two calls with the
 * same arguments in different key order hash equal. The hash covers the FULL
 * input — distinct args never collide by design (D3).
 */
export function canonicalize(value: unknown): string {
	if (value === undefined) return "undefined";
	if (value === null) return "null";
	if (typeof value === "number" || typeof value === "boolean") return String(value);
	if (typeof value === "string") return JSON.stringify(value);
	if (Array.isArray(value)) return `[${value.map(canonicalize).join(",")}]`;
	if (typeof value === "object") {
		const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
			a < b ? -1 : a > b ? 1 : 0,
		);
		return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${canonicalize(v)}`).join(",")}}`;
	}
	return String(value);
}

/** FNV-1a 32-bit over the canonical form (equality hashing, not crypto). */
export function stableHash(value: unknown): string {
	const s = canonicalize(value);
	let h = 0x811c9dc5;
	for (let i = 0; i < s.length; i++) {
		h ^= s.charCodeAt(i);
		h = Math.imul(h, 0x01000193);
	}
	return (h >>> 0).toString(16).padStart(8, "0");
}

export interface AssistantMsgLite {
	role?: string;
	content?: unknown;
	stopReason?: string;
	[k: string]: unknown;
}

/**
 * Q1 detector. Empty = no tool calls, no non-empty text (reasoning blocks
 * stripped by construction), stopReason stop|length. Everything else (error —
 * pi owns provider retry; aborted — the user did it; toolUse — tool loop, Q2's
 * lane) is NOT an empty-content fault.
 */
export function isEmptyAssistantTurn(message: AssistantMsgLite | undefined | null): boolean {
	if (!message || message.role !== "assistant") return false;
	if (message.stopReason !== "stop" && message.stopReason !== "length") return false;
	if (!Array.isArray(message.content)) return false;
	let sawToolCall = false;
	let sawText = false;
	for (const part of message.content as Array<{ type?: string; text?: string }>) {
		if (part?.type === "toolCall") sawToolCall = true;
		else if (part?.type === "text" && typeof part.text === "string" && part.text.trim().length > 0) {
			sawText = true;
		}
	}
	return !sawToolCall && !sawText;
}

/**
 * Q2 window: last N hashes + streak of identical consecutive calls. Window
 * clears on any distinct call (pattern restart, `grep A | grep A | grep B`).
 */
export class LoopWindow {
	private hashes: string[] = [];
	private streakHash: string | null = null;
	private streak = 0;

	size: number;
	strikes: number;

	constructor(size: number = LOOP_WINDOW_SIZE, strikes: number = LOOP_STRIKES) {
		this.size = size;
		this.strikes = strikes;
	}

	/** Push a call hash; returns the current identical-consecutive streak count. */
	push(hash: string): number {
		this.hashes.push(hash);
		if (this.hashes.length > this.size) this.hashes.shift();
		if (hash === this.streakHash) {
			this.streak += 1;
		} else {
			this.streakHash = hash;
			this.streak = 1;
		}
		return this.streak;
	}

	wouldBlock(): boolean {
		return this.streak >= this.strikes;
	}

	get countInWindow(): number {
		return this.streakHash === null
			? 0
			: this.hashes.filter((h) => h === this.streakHash).length;
	}

	reset(): void {
		this.hashes = [];
		this.streakHash = null;
		this.streak = 0;
	}
}

export function buildLoopBlockReason(toolName: string, strikes: number): string {
	return (
		`Loop guard: ${toolName} has now been called ${strikes}× with IDENTICAL arguments — ` +
		"the result cannot have changed. Do something different: vary the arguments, " +
		"or re-read the earlier result you already have before retrying."
	);
}

const EXCERPT_MARK = "\n…[quality-monitor elided]…\n";

/** UTF-8-safe head slice (never emits a split surrogate/multibyte fragment). */
export function utf8Head(text: string, bytes: number): string {
	const raw = encoder.encode(text);
	if (raw.length <= bytes) return text;
	return decoder.decode(raw.slice(0, bytes)).replace(RE_TAIL_FFFD, "");
}

/** UTF-8-safe tail slice — errors live at the end of logs. */
export function utf8Tail(text: string, bytes: number): string {
	const raw = encoder.encode(text);
	if (raw.length <= bytes) return text;
	return decoder.decode(raw.slice(raw.length - bytes)).replace(RE_HEAD_FFFD, "");
}

export interface ContentPart {
	type?: string;
	text?: string;
	[k: string]: unknown;
}

export interface ExcerptPatch {
	content: ContentPart[];
	details: { excerpted: { original_bytes: number; kept_bytes: number } };
}

/**
 * Q3 rewriter. >threshold → head(1.5KB) of the FIRST text part + tail(0.5KB) of
 * the LAST text part + narrowing directive; non-text parts (images) preserved
 * untouched. ≤threshold → null (passthrough; handler returns undefined).
 */
export function excerptContent(
	content: unknown,
	threshold = EXCERPT_THRESHOLD_BYTES,
): ExcerptPatch | null {
	if (typeof content === "string") {
		content = [{ type: "text", text: content }];
	}
	if (!Array.isArray(content)) return null;
	const parts = content as ContentPart[];
	const textParts = parts.filter((p) => p?.type === "text" && typeof p.text === "string");
	const total = textParts.reduce((n, p) => n + utf8len(p.text as string), 0);
	if (total <= threshold) return null;

	const first = textParts[0]?.text ?? "";
	const last = textParts[textParts.length - 1]?.text ?? "";
	const body =
		textParts.length === 1
			? `${utf8Head(first, EXCERPT_HEAD_BYTES)}${EXCERPT_MARK}${utf8Tail(first, EXCERPT_TAIL_BYTES)}`
			: `${utf8Head(first, EXCERPT_HEAD_BYTES)}${EXCERPT_MARK}${utf8Tail(last, EXCERPT_TAIL_BYTES)}`;
	const directive =
		`\n[result truncated by quality-monitor: ${total} bytes → shown ${utf8len(body)}; ` +
		`re-run with narrower flags/grep/tail to fetch the elided middle]`;
	const text = body + directive;
	const nonText = parts.filter((p) => !(p?.type === "text"));
	const excerptedPart: ContentPart = { type: "text", text };
	const kept = utf8len(text) + nonText.reduce((n, p) => n + utf8len(String(p?.text ?? "")), 0);
	return {
		content: [excerptedPart, ...nonText],
		details: { excerpted: { original_bytes: total, kept_bytes: kept } },
	};
}

/** Q1 corrective steer (one per run; cap enforced by the caller). */
export function buildEmptySteerMessage(): string {
	return (
		"Your previous reply contained no visible content (likely reasoning-only output). " +
		"Respond now in plain text with your actual answer — keep tool use out of the reply. " +
		"If you were about to call a tool, emit that tool call instead of an empty message."
	);
}

// ---------- Session-state reconstruction (/resume parity, D6) ----------

export interface QualityState {
	emptyTotal: number;
	steeredTotal: number;
	loopBlocksTotal: number;
	excerptsTotal: number;
	thinkingFallbacksTotal: number;
	thinkingFallbackModels: string[];
}

export function reconstructState(entries: unknown[]): QualityState {
	const state: QualityState = {
		emptyTotal: 0,
		steeredTotal: 0,
		loopBlocksTotal: 0,
		excerptsTotal: 0,
		thinkingFallbacksTotal: 0,
		thinkingFallbackModels: [],
	};
	for (const entry of entries) {
		const e = entry as { type?: string; customType?: string; data?: Record<string, unknown> };
		if (e?.type !== "custom" || typeof e.customType !== "string" || !e.customType.startsWith("quality:")) {
			continue;
		}
		const d = (e.data ?? {}) as Record<string, unknown>;
		switch (e.customType) {
			case "quality:empty_content":
				state.emptyTotal++;
				if (d.steered === true) state.steeredTotal++;
				break;
			case "quality:loop_block":
				state.loopBlocksTotal++;
				break;
			case "quality:excerpt":
				state.excerptsTotal++;
				break;
			case "quality:thinking_fallback": {
				state.thinkingFallbacksTotal++;
				const model = String(d.model ?? "");
				if (model && !state.thinkingFallbackModels.includes(model)) {
					state.thinkingFallbackModels.push(model);
				}
				break;
			}
			default:
				break;
		}
	}
	return state;
}

// ---------- Runtime ----------

interface ToolCallEvent {
	toolName: string;
	toolCallId?: string;
	input?: Record<string, unknown>;
}

interface ToolResultEvent {
	toolName: string;
	toolCallId?: string;
	content?: unknown;
	details?: Record<string, unknown>;
	isError?: boolean;
}

interface TurnEndEvent {
	turnIndex?: number;
	message?: AssistantMsgLite;
	toolResults?: unknown[];
}

export default async function (pi: ExtensionAPI) {
	// Kill switch read once — the whole extension goes inert.
	if (process.env[KILL_SWITCH_ENV] === "0") return;

	const thinkingFallbackEnabled = process.env[THINKING_FALLBACK_ENV] === "1";

	// Per-run caps reset on agent_start; session totals rebuild from entries on
	// session_start so /resume restores the hard ceilings (D6).
	const loopWindow = new LoopWindow();
	let loopBlocksThisRun = 0;
	let steersThisRun = 0;
	let loopBlocksSession = 0;
	let steersSession = 0;
	let loopCapNotified = false;
	let steerCapNotified = false;
	const emptyStreakByModel = new Map<string, number>();
	const thinkingFallbackDone = new Set<string>();

	pi.on("session_start", async (_event, ctx) => {
		try {
			const state = reconstructState(ctx.sessionManager.getEntries() as unknown[]);
			loopBlocksSession = state.loopBlocksTotal;
			steersSession = state.steeredTotal;
		} catch {
			// Fail-open for state reconstruction only: detection still works this
			// session; only the hard ceiling may be under-counted after a resume.
		}
	});

	pi.on("agent_start", async () => {
		loopBlocksThisRun = 0;
		steersThisRun = 0;
		loopCapNotified = false;
		steerCapNotified = false;
	});

	// ---- Q2: loop detection (block-then-inform) ----
	pi.on("tool_call", async (event, ctx) => {
		const ev = event as ToolCallEvent;
		const streak = loopWindow.push(stableHash({ tool: ev.toolName, input: ev.input }));
		if (streak < LOOP_STRIKES) return undefined;

		if (loopBlocksThisRun >= LOOP_BLOCK_CAP_PER_RUN || loopBlocksSession >= MAX_LOOP_BLOCKS_PER_SESSION) {
			if (!loopCapNotified) {
				loopCapNotified = true;
				try {
					ctx.ui.notify(
						`Loop guard: block cap reached — further identical calls are observed, not blocked (streak ${streak}× on ${ev.toolName})`,
						"warning",
					);
				} catch {
					/* headless: notifications unavailable — entry below is the record */
				}
			}
			pi.appendEntry("quality:loop_observed", { tool: ev.toolName, streak });
			return undefined; // notify-only: never deterministically fatal
		}

		loopBlocksThisRun++;
		loopBlocksSession++;
		pi.appendEntry("quality:loop_block", {
			tool: ev.toolName,
			streak,
			run_blocks: loopBlocksThisRun,
			hash: stableHash({ tool: ev.toolName, input: ev.input }),
			ts: new Date().toISOString(),
		});
		try {
			ctx.ui.notify(`Loop guard: blocked repeat #${loopBlocksThisRun} of ${ev.toolName} (identical args ×${streak})`, "warning");
		} catch {
			/* headless */
		}
		return { block: true, reason: buildLoopBlockReason(ev.toolName, streak) };
	});

	// ---- Q3: bounded-excerpt rewrite ----
	pi.on("tool_result", async (event) => {
		const ev = event as ToolResultEvent;
		const patch = excerptContent(ev.content);
		if (!patch) return undefined;
		pi.appendEntry("quality:excerpt", {
			tool: ev.toolName,
			original_bytes: patch.details.excerpted.original_bytes,
			kept_bytes: patch.details.excerpted.kept_bytes,
			ts: new Date().toISOString(),
		});
		return { content: patch.content, details: { ...(ev.details ?? {}), ...patch.details } };
	});

	// ---- Q1: empty-content detection + capped corrective steer ----
	pi.on("turn_end", async (event, ctx) => {
		const ev = event as TurnEndEvent;
		const msg = ev.message;
		if (!isEmptyAssistantTurn(msg)) {
			const model = String(msg?.model ?? ctx.model?.id ?? "unknown");
			emptyStreakByModel.delete(model);
			return undefined;
		}
		if (ev.toolResults && ev.toolResults.length > 0) return undefined; // tool loop lane, not this one

		const model = String(msg?.model ?? ctx.model?.id ?? "unknown");
		const streak = (emptyStreakByModel.get(model) ?? 0) + 1;
		emptyStreakByModel.set(model, streak);

		// Q4 gate first: 2+ empty streams for the same model (opt-in lane).
		if (thinkingFallbackEnabled && streak >= THINKING_FALLBACK_STRIKES && !thinkingFallbackDone.has(model)) {
			thinkingFallbackDone.add(model);
			try {
				const before = pi.getThinkingLevel();
				if (before !== "off") {
					pi.setThinkingLevel("off"); // clamped to model capabilities by pi
					pi.appendEntry("quality:thinking_fallback", {
						model,
						from_level: before,
						streak,
						ts: new Date().toISOString(),
					});
					try {
						ctx.ui.notify(`Thinking budget fallback: ${model} produced ${streak} empty streams — thinking set to off`, "warning");
					} catch {
						/* headless */
					}
				}
			} catch (e) {
				// Level control unavailable → honest record, never a silent skip.
				pi.appendEntry("quality:steer_failed", {
					kind: "thinking_fallback",
					model,
					error: String(e),
				});
			}
		}

		// Steer decision: capped 1/run (D2), hard session ceiling, silent no-op if settled.
		let steered = false;
		let why: string | undefined;
		if (steersThisRun >= EMPTY_STEER_CAP_PER_RUN) {
			why = "cap_per_run";
		} else if (steersSession >= MAX_STEERS_PER_SESSION) {
			why = "cap_per_session";
			if (!steerCapNotified) {
				steerCapNotified = true;
				try {
					ctx.ui.notify("Quality monitor: steering cap reached — empty turns recorded, no further corrections", "warning");
				} catch {
					/* headless */
				}
			}
		} else {
			try {
				if (!ctx.isIdle()) {
					steersThisRun++;
					steersSession++;
					await pi.sendUserMessage(buildEmptySteerMessage(), { deliverAs: "steer" });
					steered = true;
				} else {
					why = "settled"; // D2: silent no-op when the session is settled
				}
			} catch (e) {
				why = "steer_failed";
				pi.appendEntry("quality:steer_failed", {
					kind: "empty_content",
					model,
					error: String(e),
				});
			}
		}
		pi.appendEntry("quality:empty_content", {
			turn: ev.turnIndex,
			model,
			stop_reason: msg?.stopReason,
			streak,
			steered,
			...(why ? { why } : {}),
			ts: new Date().toISOString(),
		});
		return undefined;
	});
}