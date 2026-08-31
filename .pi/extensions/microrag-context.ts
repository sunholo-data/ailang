/**
 * microrag-context — M-DX-MICRORAG-CONTEXT (v0.35.0)
 *
 * μRAG as a first-class pi extension: the harness this repo's daily work and rig
 * evaluations run in finally gets the retrieval frontend every other harness has.
 * Three lanes, all wrapping the existing `ailang micro-rag` engine (never
 * re-implements retrieval — CLAUDE.md principle 1):
 *
 *   M1  prompt-intent injection — `before_agent_start` runs the user prompt through
 *       `micro-rag user-prompt`; above-floor hits are injected as a TRAILING message.
 *       System prompt is never touched (D1 — provider prefix cache stays byte-stable).
 *   M2  error-triggered injection — `ailang_check` structured diagnostics (V12) are
 *       buffered; on the model's NEXT attempt a dense synthetic query
 *       (`AILANG error <CODE>: <message> — how to fix`, D3) is served. Max depth 1,
 *       clear-on-ATTEMPT (no retry loops), newest wins. This lane exists in no other
 *       frontend anywhere.
 *   M3  on-demand `microrag_search` tool — returns the engine envelope back-flat
 *       (cap 3 hits, ≤1KB excerpt each), description includes a paraphrase example.
 *
 * Gate matrix (sprint plan R1/R2 — deliberately DIVERGES from the engine/shim
 * default-on: pi is the rig's eval path, so unset env must preserve today's
 * no-microrag baseline; `--microrag on/off` already forces the env both ways):
 *   - AILANG_MICRORAG_ENABLED: "1"/truthy → active; unset/empty/"0"/"false" → the
 *     whole extension is INERT (nothing registers, no subprocess ever spawned).
 *     This is what the eval harness's --microrag off arm forces — arm parity.
 *   - PI_MICRORAG_CONTEXT=0    → whole extension inert (total kill switch).
 *   - PI_MICRORAG_INJECT=0     → M1/M2 off, `microrag_search` stays.
 *   - PI_MICRORAG_TOOL=0       → tool off, M1/M2 stays.
 *
 * Engine-call contract (sprint plan R3–R7): prompt handed over via `--prompt @tmpfile`
 * (capped at 4096 chars, shim parity); session key injected as
 * `AILANG_MICRORAG_SESSION=pi:<session>` through an `env` argv wrapper because
 * pi.exec offers no env option — WITHOUT it every engine call would derive a fresh
 * pid-scoped key and the dedup ledger (V9) would never dedup across turns
 * (measured: pi spawn inherits parent env verbatim, dist/core/exec.js).
 * 5s timeout (claude-code hook precedent is 4s); timeout/unparseable engine →
 * structured no-op + one notify per run — engine FAILURE notifies, floor-miss
 * never does (A11).
 *
 * Tested against pi 0.84.3/0.84.4. Suite member #11. Distribution: Tier 0–2 via
 * `make pi-assets` (design doc: m-dx-microrag-context.md, sprint plan: sibling file).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// ---------- Tunables (exported for table-driven tests) ----------

export const KILL_SWITCH_ENV = "PI_MICRORAG_CONTEXT";
export const INJECT_KILL_ENV = "PI_MICRORAG_INJECT";
export const TOOL_KILL_ENV = "PI_MICRORAG_TOOL";
export const ENABLE_ENV = "AILANG_MICRORAG_ENABLED";
export const SESSION_ENV = "AILANG_MICRORAG_SESSION";
export const TIMEOUT_ENV = "PI_MICRORAG_TIMEOUT_MS";

export const ENGINE_TIMEOUT_MS = 5000; // shim parity: CC UserPromptSubmit hooks run this command at 4s
export const PROMPT_CAP_CHARS = 4096; // shim parity: microrag_userprompt.sh caps the prompt identically
export const MIN_PROMPT_CHARS = 20; // shim parity: AILANG_MICRORAG_USERPROMPT_MIN_LEN default
export const QUERY_CAP_CHARS = 512; // synthetic error query cap
export const HITS_CAP = 3; // M3: cap hits sent back to the model
export const HIT_EXCERPT_CHARS = 1024; // per-hit pretty excerpt cap (M3)
export const MAX_CODES_IN_BUFFER = 8; // dedup'd error codes per buffer

export const CORPUS_NAMESPACES: Record<string, string> = {
	syntax: "ailang-syntax",
	builtins: "ailang-builtins",
	docs: "ailang-docs", // absent corpus degrades to below_floor → empty hits (R10)
};

/** Entry type for the one-entry error buffer (State Management pattern, V9-adjacent). */
export const LAST_ERROR_ENTRY = "microrag:last_error";
/** customType of injected trailing messages (D1 — never a systemPrompt edit). */
export const INJECT_CUSTOM_TYPE = "microrag-inject";

const AILANG_TOKEN_RE = /\bailang\b|\B\.ail\b/i; // ".ail" after whitespace has no leading \b (dot is non-word) — hence \B

// ---------- Pure helpers ----------

/**
 * Gate matrix (sprint plan R1/R2). Returns "inert" (extension must not register
 * anything nor spawn anything) or "active".
 */
export function gateDecision(opts: {
	kill: string | undefined;
	enabled: string | undefined;
}): "inert" | "active" {
	if (opts.kill === "0") return "inert";
	const v = (opts.enabled ?? "").trim();
	if (v === "" || v === "0" || v.toLowerCase() === "false") return "inert";
	return "active";
}

/**
 * M1 pre-gate (R7): pure commands and trivial prompts never trigger a subprocess.
 * Requires BOTH the shim-parity min length AND (the design doc's >8 words OR an
 * AILANG token).
 */
export function promptWarrantsLookup(prompt: string | undefined): boolean {
	if (typeof prompt !== "string") return false;
	const p = prompt.trim();
	if (p.length < MIN_PROMPT_CHARS) return false;
	if (p.startsWith("/")) return false; // pure commands like /sprint-start
	if (p.split(/\s+/).length > 8) return true;
	return AILANG_TOKEN_RE.test(p);
}

export interface EngineEnvelope {
	inject: boolean;
	content: string;
	reason: string; // injected | below_floor | unparseable | empty | engine_error
	meta?: { ns?: string; score?: number; snippetId?: string; tokens?: number };
}

/**
 * R4 envelope parser. Hit: `{"injection":{"injection_text",…},"reason":"injected"}`
 * (exit 0); miss: `{"reason":"below_floor"}` (exit 0). Shim parity: an empty
 * injection_text is treated as no-inject even when reason claims injected.
 */
export function parseEngineEnvelope(stdout: string | undefined | null): EngineEnvelope {
	const s = (stdout ?? "").trim();
	if (s === "") return { inject: false, content: "", reason: "empty" };
	let parsed: {
		reason?: string;
		injection?: {
			injection_text?: string;
			snippet_id?: string;
			tokens?: number;
			ns?: string;
			score?: number;
		};
	};
	try {
		parsed = JSON.parse(s);
	} catch {
		return { inject: false, content: "", reason: "unparseable" };
	}
	const text = (parsed.injection?.injection_text ?? "").trim();
	if (parsed.reason === "injected" && text !== "") {
		return {
			inject: true,
			content: text,
			reason: "injected",
			meta: {
				ns: parsed.injection?.ns,
				score: parsed.injection?.score,
				snippetId: parsed.injection?.snippet_id,
				tokens: parsed.injection?.tokens,
			},
		};
	}
	return { inject: false, content: "", reason: parsed.reason ?? "below_floor" };
}

/**
 * R9 synthetic dense query (D3). Errors are the highest per-token-value query
 * shape (ADR-002). MEASURED 2026-08-31 (pre-implementation e2e + engine probes):
 * code-CITATION templates ("AILANG error PAR015: …") fall BELOW the engine's
 * embedding floor, while dense fix-intent phrasing of the diagnostic MESSAGE
 * clears it (hits the Let Bindings chunk where the cited-code form missed).
 * So the query embeds the path-stripped message plus fix intent; codes ride the
 * entries/audit, not the embedding.
 */
export function syntheticErrorQuery(codes: string[], firstMessage: string | undefined): string {
	if (codes.length === 0) return "";
	// File-path tokens are embedding noise (ADR-002's diagnosis, measured) — strip.
	const msg = (firstMessage ?? "")
		.replace(/\S*\/\S*/g, " ")
		.replace(/\s+/g, " ")
		.trim();
	if (msg === "") return ""; // no clean intent-bearing text → no query, no injection
	const core = `${msg} — how do I fix this in AILANG?`;
	return core.length <= QUERY_CAP_CHARS ? core : `${core.slice(0, QUERY_CAP_CHARS - 1)}…`;
}

export interface CheckBuffer {
	codes: string[];
	firstMessage: string;
	at: string; // ISO timestamp
}

/**
 * R8: extract the error-severity diagnostics from the shipped `ailang_check`
 * tool shape (`details: { ok, diagnostics: [{code, severity, message, …}] }`).
 * Warnings never buffer (fix-guidance value lives with errors); passing checks
 * and missing/malformed details yield null.
 */
export function extractCheckErrors(details: unknown): CheckBuffer | null {
	if (details === null || typeof details !== "object") return null;
	const diags = (details as { diagnostics?: unknown }).diagnostics;
	if (!Array.isArray(diags)) return null;
	const codes: string[] = [];
	let firstMessage = "";
	for (const d of diags as Array<{ severity?: string; code?: string; message?: string }>) {
		if (d?.severity !== "error") continue;
		const code = String(d.code ?? "").trim();
		if (code === "") continue;
		if (!codes.includes(code)) codes.push(code);
		if (firstMessage === "" && typeof d.message === "string" && d.message.trim() !== "") {
			firstMessage = d.message.trim();
		}
	}
	if (codes.length === 0) return null;
	return { codes: codes.slice(0, MAX_CODES_IN_BUFFER), firstMessage, at: new Date().toISOString() };
}

export interface CheckBufferState {
	buffer: CheckBuffer | null;
	served: boolean;
}

export type InjectionSelection = { query: null; from: null } | { query: string; from: "error" | "intent" };

/**
 * M2 selection: an UNSERVED error buffer wins over prompt intent
 * (error-recovery > intent — the little-coder ordering). The prompt path goes
 * through the same pre-gate as M1. One query max per turn.
 */
export function selectInjectionQuery(
	state: CheckBufferState,
	prompt: string | undefined,
): InjectionSelection {
	if (state.buffer && !state.served) {
		const q = syntheticErrorQuery(state.buffer.codes, state.buffer.firstMessage);
		if (q !== "") return { query: q, from: "error" };
	}
	if (promptWarrantsLookup(prompt)) {
		const p = (prompt as string).trim();
		return { query: p.length > PROMPT_CAP_CHARS ? `${p.slice(0, PROMPT_CAP_CHARS - 1)}…` : p, from: "intent" };
	}
	return { query: null, from: null };
}

/**
 * Serve-on-ATTEMPT: the buffer is consumed the moment its query fires, injection
 * hit or miss (a below-floor result must not spawn a retry loop).
 */
export function markAttempted(state: CheckBufferState): CheckBufferState {
	return { buffer: state.buffer, served: true };
}

/** R10: corpus knob → engine `--namespaces` CSV; unknown/omitted → engine default. */
export function corporaToNamespacesCsv(corpus: string | undefined): string | undefined {
	if (!corpus) return undefined;
	return CORPUS_NAMESPACES[corpus]; // undefined for unknown corpora = engine default
}

/** Engine invocation contract (R3): prompt via @tmpfile, never an argv-embedded string. */
export function buildEngineArgs(tmpPath: string, namespacesCsv?: string): string[] {
	const args = ["micro-rag", "user-prompt", "--prompt", `@${tmpPath}`];
	if (namespacesCsv) args.push("--namespaces", namespacesCsv);
	return args;
}

/**
 * `env` argv wrapper — REQUIRED (see module doc): pi.exec has no env option, and
 * without an explicit AILANG_MICRORAG_SESSION the engine derives a fresh
 * pid-scoped key per call, so the session ledger (V9) would never dedup.
 */
export function buildEngineCommand(sessionId: string, args: string[]): { command: string; argv: string[] } {
	return { command: "env", argv: [`${SESSION_ENV}=pi:${sessionId}`, "ailang", ...args] };
}

/**
 * M3 hit projection: envelope hits back-flat, capped, excerpted. `details`
 * carries the full envelope for audit (A12); the model text gets the pretty view.
 */
export function extractHitsFromEnvelope(envelope: unknown): Array<{
	tier: string;
	score: number;
	content: string;
	source: string;
}> {
	const hits = (envelope as { hits?: unknown } | null)?.hits;
	if (!Array.isArray(hits)) return [];
	return hits.slice(0, HITS_CAP).map((h) => {
		const hit = (h ?? {}) as { tier?: string; score?: number; content?: string; source?: string };
		const content = typeof hit.content === "string" ? hit.content : "";
		return {
			tier: String(hit.tier ?? "?"),
			score: Number(hit.score ?? 0),
			content: content.length > HIT_EXCERPT_CHARS ? `${content.slice(0, HIT_EXCERPT_CHARS)}…` : content,
			source: String(hit.source ?? ""),
		};
	});
}

/** /resume reconstruction: last buffer entry wins; served entries never re-serve. */
export function reconstructLastError(entries: unknown[]): CheckBufferState {
	let state: CheckBufferState = { buffer: null, served: true };
	for (const e of entries as Array<{ type?: string; customType?: string; data?: Record<string, unknown> }>) {
		if (e?.type !== "custom" || e.customType !== LAST_ERROR_ENTRY) continue;
		const d = (e.data ?? {}) as { codes?: string[]; firstMessage?: string; at?: string; served?: boolean };
		state = {
			buffer: {
				codes: Array.isArray(d.codes) ? d.codes.map(String) : [],
				firstMessage: String(d.firstMessage ?? ""),
				at: String(d.at ?? ""),
			},
			served: d.served === true,
		};
	}
	return state;
}

// ---------- Wire-up ----------

interface BeforeAgentStartEvent {
	prompt?: string;
}

interface ToolResultEvent {
	toolName?: string;
	details?: unknown;
}

export default async function (pi: ExtensionAPI) {
	// Kill switch first: whole extension inert (R2).
	if (process.env[KILL_SWITCH_ENV] === "0") return;

	// R1: explicit engine env is required. Unset/off → nothing registers, zero
	// subprocesses for the whole process lifetime — arm parity with --microrag off.
	const gate = gateDecision({
		kill: process.env[KILL_SWITCH_ENV],
		enabled: process.env[ENABLE_ENV],
	});
	if (gate === "inert") return;

	const injectEnabled = process.env[INJECT_KILL_ENV] !== "0";
	const toolEnabled = process.env[TOOL_KILL_ENV] !== "0";
	const timeoutMs = Math.max(250, Number(process.env[TIMEOUT_ENV] ?? "") || ENGINE_TIMEOUT_MS);

	// Session key for the engine's dedup ledger (V9). pi has no top-level
	// getSessionId; capture it from session_start's ctx. pid fallback keeps the
	// key process-stable — never a transient per-call pid (which would gut dedup).
	let sessionId = `pid-${process.pid}`;
	let bufferState: CheckBufferState = { buffer: null, served: true };
	let failuresNotified = false; // engine-failure notify: once per run

	const { mkdtemp, writeFile, rm } = await import("node:fs/promises");
	const { tmpdir } = await import("node:os");
	const { join } = await import("node:path");

	/** One engine call under the Subprocess Contract (timeout, cap, tmpfile, cleanup). */
	const callEngineUserPrompt = async (query: string, namespacesCsv: string | undefined): Promise<EngineEnvelope> => {
		const dir = await mkdtemp(join(tmpdir(), "microrag-"));
		const p = join(dir, "prompt.txt");
		await writeFile(p, query.slice(0, PROMPT_CAP_CHARS), "utf8");
		try {
			const { command, argv } = buildEngineCommand(sessionId, buildEngineArgs(p, namespacesCsv));
			const r = await pi.exec(command, argv, { timeout: timeoutMs });
			if (r.code !== 0) return { inject: false, content: "", reason: "engine_error" };
			return parseEngineEnvelope(r.stdout);
		} catch {
			return { inject: false, content: "", reason: "engine_error" };
		} finally {
			await rm(dir, { recursive: true, force: true }).catch(() => {});
		}
	};

	pi.on("session_start", async (_event, ctx) => {
		sessionId = ctx.sessionManager.getSessionId();
		bufferState = { buffer: null, served: true };
		try {
			bufferState = reconstructLastError(ctx.sessionManager.getEntries() as unknown[]);
		} catch {
			// fail-open: detection works; only /resume parity of the buffer is lost
		}
	});

	pi.on("agent_start", async () => {
		// failure-notify is once-per-run; buffers are per-session-by-design
		failuresNotified = false;
	});

	// ---- M2 input side: buffer the newest structured check error ----
	pi.on("tool_result", async (event) => {
		const ev = event as ToolResultEvent;
		if (ev.toolName !== "ailang_check" || !injectEnabled) return;
		const b = extractCheckErrors(ev.details);
		if (!b) return;
		bufferState = { buffer: b, served: false };
		pi.appendEntry(LAST_ERROR_ENTRY, { ...b, served: false });
	});

	// ---- M1/M2 output side: inject on the model's next attempt ----
	pi.on("before_agent_start", async (event, ctx) => {
		if (!injectEnabled) return;
		const ev = event as BeforeAgentStartEvent;
		const sel = selectInjectionQuery(bufferState, ev.prompt);
		let from: "error" | "intent" | null = null;
		let query: string | null = null;
		if (sel.from === "error" && sel.query !== null) {
			bufferState = markAttempted(bufferState); // serve-on-attempt: no retry loops
			pi.appendEntry(LAST_ERROR_ENTRY, { ...bufferState.buffer, served: true });
			from = "error";
			query = sel.query;
		} else if (sel.from === "intent" && sel.query !== null) {
			from = "intent";
			query = sel.query;
		}
		if (query === null) return;

		const envelope = await callEngineUserPrompt(query);
		if (!envelope.inject) {
			// A11: engine *failure* notifies (once per run); floor-miss is silent.
			if ((envelope.reason === "engine_error" || envelope.reason === "empty" || envelope.reason === "unparseable") && !failuresNotified) {
				failuresNotified = true;
				try {
					ctx.ui.notify(`μRAG engine call failed (${envelope.reason}) — injection skipped this turn`);
				} catch {
					// headless has no ui — entry still banked below
				}
			}
			pi.appendEntry("microrag:skip", { query, from, reason: envelope.reason, at: new Date().toISOString() });
			return;
		}
		// D1: trailing message only — never a systemPrompt edit. Engine content
		// already carries the 🧠 μRAG provenance marker (grep-able in any transcript).
		pi.appendEntry("microrag:injected", {
			query,
			from,
			ns: envelope.meta?.ns,
			score: envelope.meta?.score,
			snippetId: envelope.meta?.snippetId,
			tokens: envelope.meta?.tokens,
			at: new Date().toISOString(),
		});
		return {
			message: {
				customType: INJECT_CUSTOM_TYPE,
				content: envelope.content,
				display: true,
			},
		};
	});

	// ---- M3: on-demand search tool ----
	if (!toolEnabled) return;
	const { Type } = await import("typebox");
	pi.registerTool({
		name: "microrag_search",
		label: "μRAG search",
		description:
			"Search the AILANG knowledge corpora (μRAG engine). Pass a natural-language query about AILANG " +
			"syntax, builtins, or docs — e.g. paraphrase what you need ('how do I read a file in AILANG?') " +
			"BEFORE guessing API names. Returns scored hits with source references. Optional corpus: " +
			"syntax | builtins | docs (default: syntax+builtins).",
		parameters: Type.Object({
			query: Type.String({ description: "Dense natural-language question about AILANG (not keywords)" }),
			corpus: Type.Optional(
				Type.Union([Type.Literal("syntax"), Type.Literal("builtins"), Type.Literal("docs")], {
					description: "Restrict to one corpus; omit for the default pair",
				}),
			),
		}),
		async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
			void ctx;
			const envelope = await callEngineUserPrompt(params.query, corporaToNamespacesCsv(params.corpus));
			if (envelope.reason === "engine_error") {
				return {
					content: [{ type: "text", text: "μRAG engine call failed (timeout/crash) — no hits this time." }],
					details: { error: "engine_error", query: params.query },
				};
			}
			// Above-floor single-hit injections via the tool path: expose the engine
			// text directly, mirroring M1's payload shape.
			if (envelope.inject) {
				return {
					content: [{ type: "text", text: envelope.content }],
					details: { hits: [{ tier: "injection", score: envelope.meta?.score, content: envelope.content.slice(0, HIT_EXCERPT_CHARS), source: envelope.meta?.ns ?? "" }], reason: envelope.reason },
				};
			}
			// Structured skip — below-floor is a result, not an error (A11/A12).
			return {
				content: [{ type: "text", text: JSON.stringify({ hits: [] }) }],
				details: { hits: [], reason: envelope.reason, query: params.query },
			};
		},
	});
}