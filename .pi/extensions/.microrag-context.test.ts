/**
 * Table-driven unit tests for microrag-context pure functions (M-DX-MICRORAG-CONTEXT).
 * Run: node --experimental-strip-types --test .pi/extensions/.microrag-context.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 *
 * Gate matrix, envelope parsing, synthetic queries, buffer state, and the
 * engine-parity contract with the claude-code shim (design doc Testing §2).
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import {
	KILL_SWITCH_ENV,
	ENABLE_ENV,
	ENGINE_TIMEOUT_MS,
	PROMPT_CAP_CHARS,
	MIN_PROMPT_CHARS,
	QUERY_CAP_CHARS,
	HITS_CAP,
	HIT_EXCERPT_CHARS,
	CORPUS_NAMESPACES,
	gateDecision,
	promptWarrantsLookup,
	parseEngineEnvelope,
	syntheticErrorQuery,
	extractCheckErrors,
	selectInjectionQuery,
	markAttempted,
	corporaToNamespacesCsv,
	buildEngineArgs,
	extractHitsFromEnvelope,
	reconstructLastError,
	type CheckBufferState,
} from "./microrag-context.ts";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));

// ---------- R1/R2 gate matrix ----------

test("gateDecision: kill switch wins over everything", () => {
	assert.equal(gateDecision({ kill: "0", enabled: "1" }), "inert");
	assert.equal(gateDecision({ kill: "0", enabled: undefined }), "inert");
});

test("gateDecision: unset/empty/falsy engine env → inert (R1 — diverges from engine default deliberately)", () => {
	assert.equal(gateDecision({ kill: undefined, enabled: undefined }), "inert");
	assert.equal(gateDecision({ kill: undefined, enabled: "" }), "inert");
	assert.equal(gateDecision({ kill: undefined, enabled: "0" }), "inert");
	assert.equal(gateDecision({ kill: undefined, enabled: "false" }), "inert");
	assert.equal(gateDecision({ kill: undefined, enabled: "FALSE" }), "inert");
});

test("gateDecision: explicit truthy env → active", () => {
	assert.equal(gateDecision({ kill: undefined, enabled: "1" }), "active");
	assert.equal(gateDecision({ kill: "1", enabled: "1" }), "active");
});

test("constants match the sprint plan budgets", () => {
	assert.equal(ENGINE_TIMEOUT_MS, 5000);
	assert.equal(PROMPT_CAP_CHARS, 4096); // shim parity: microrag_userprompt.sh caps at 4096 chars
	assert.equal(MIN_PROMPT_CHARS, 20); // shim parity: AILANG_MICRORAG_USERPROMPT_MIN_LEN default
	assert.equal(QUERY_CAP_CHARS, 512);
	assert.equal(HITS_CAP, 3);
	assert.equal(HIT_EXCERPT_CHARS, 1024);
});

// ---------- R7 pre-gate ----------

test("promptWarrantsLookup: pure commands and trivial prompts never trigger", () => {
	assert.equal(promptWarrantsLookup(undefined), false);
	assert.equal(promptWarrantsLookup(""), false);
	assert.equal(promptWarrantsLookup("   \n  "), false);
	assert.equal(promptWarrantsLookup("/sprint-start m-dx"), false);
	assert.equal(promptWarrantsLookup("ok"), false);
	assert.equal(promptWarrantsLookup("yes please"), false);
	assert.equal(promptWarrantsLookup("what is the weather today in tokyo"), false); // generic, short
});

test("promptWarrantsLookup: >8 words or AILANG token triggers", () => {
	assert.equal(promptWarrantsLookup("how do I concat strings in AILANG?"), true);
	assert.equal(promptWarrantsLookup("fix the .ail file that fails to parse"), true);
	assert.equal(
		promptWarrantsLookup("now I need you to look at this failing benchmark result and tell me what happened"),
		true,
	);
});

test("promptWarrantsLookup: min-length + cap boundaries (shim parity)", () => {
	// 5000 chars of one non-word token: no words >8, no AILANG token → no embedding
	// spend on a semantic-free paste (the shim would have queried it; we pre-gate)
	assert.equal(promptWarrantsLookup("x".repeat(5000)), false);
	// shim's 20-char floor dominates trivial short text EVEN over the words gate
	// (engine enforces a min length — passing it a 17-char query buys nothing)
	assert.equal(promptWarrantsLookup("a b c d e f g h i"), false); // 17 chars, below floor
	assert.equal(promptWarrantsLookup("can you look at this failing benchmark result quickly"), true); // 53 chars, 9 words → words gate
});

// ---------- R4 envelope parsing ----------

const LIVE_HIT = `{"injection":{"injection_text":"━━━ 🧠 μRAG [ailang-builtins] ━━━\\n→ cli\\n  [ns:ailang-builtins] [version:v0.16.6] [module:std/string] [effect:pure]\\n  concat_String: (string, string) -> string\\n━━━━━━━━━━━━━━━\\n","snippet_id":"6e1df4c4baaea2d01520d0ad6ab2abde","tokens":55,"ns":"ailang-builtins","score":0.8124941348452831,"microrag_state":"on"},"microrag_state":"on","reason":"injected"}`;

test("parseEngineEnvelope: live hit shape → inject with text + provenance", () => {
	const e = parseEngineEnvelope(LIVE_HIT);
	assert.equal(e.inject, true);
	assert.match(e.content, /🧠 μRAG/);
	assert.match(e.content, /concat_String/);
	assert.equal(e.reason, "injected");
	assert.equal(e.meta?.ns, "ailang-builtins");
	assert.equal(e.meta?.snippetId, "6e1df4c4baaea2d01520d0ad6ab2abde");
});

test("parseEngineEnvelope: below_floor / dryrun / empty / garbage → no inject, exit-0 semantics", () => {
	assert.equal(parseEngineEnvelope('{"microrag_state":"on","reason":"below_floor"}').inject, false);
	assert.equal(parseEngineEnvelope("").inject, false);
	assert.equal(parseEngineEnvelope("").reason, "empty");
	assert.equal(parseEngineEnvelope(undefined).inject, false);
	assert.equal(parseEngineEnvelope("<stderr noise>").reason, "unparseable");

	// shim parity: injection_text empty → no inject even if reason claims injected
	assert.equal(
		parseEngineEnvelope('{"injection":{"injection_text":""},"reason":"injected"}').inject,
		false,
	);
});

// ---------- R9 synthetic error query ----------

test("syntheticErrorQuery: path-stripped message + fix intent (code-citation template measured below floor)", () => {
	const q = syntheticErrorQuery(["IMP010"], "Export statement is not allowed here");
	assert.equal(q, "Export statement is not allowed here — how do I fix this in AILANG?");

	const q2 = syntheticErrorQuery(["TC010"], "failed to load /tmp/x.ail: parse errors in /tmp/x.ail: bad token");
	assert.doesNotMatch(q2, /\/tmp/); // paths stripped — embedding noise (measured)
	assert.match(q2, /bad token — how do I fix this in AILANG\?$/);

	const longMsg = "y".repeat(2000);
	const q3 = syntheticErrorQuery(["TC010"], longMsg);
	assert.ok(q3.length <= QUERY_CAP_CHARS);
});

test("syntheticErrorQuery: no codes or empty message → no query, no injection", () => {
	assert.equal(syntheticErrorQuery([], "boom"), "");
	assert.equal(syntheticErrorQuery(["IMP010"], undefined), ""); // empty message → no garbage query
});

// ---------- R8 buffer extraction from ailang_check details ----------

test("extractCheckErrors: error diagnostics buffered, warnings ignored", () => {
	const details = {
		ok: false,
		diagnostics: [
			{ code: "IMP010", severity: "error", message: "Export statement is not allowed here", file: "x.ail", line: 3, col: 1, hint: "" },
			{ code: "UNU001", severity: "warning", message: "unused binding", file: "x.ail", line: 5, col: 7, hint: "" },
		],
	};
	const b = extractCheckErrors(details);
	assert.ok(b);
	assert.deepEqual(b.codes, ["IMP010"]);
	assert.equal(b.firstMessage, "Export statement is not allowed here");
});

test("extractCheckErrors: warning-only / passing / missing details → null", () => {
	assert.equal(extractCheckErrors({ ok: true, diagnostics: [{ code: "UNU001", severity: "warning" }] }), null);
	assert.equal(extractCheckErrors({ ok: true, diagnostics: [] }), null);
	assert.equal(extractCheckErrors(undefined), null);
	assert.equal(extractCheckErrors({}), null);
});

// ---------- M2 selection state ----------

test("selectInjectionQuery: unserved error buffer wins over prompt intent (error-recovery first)", () => {
	const state: CheckBufferState = {
		buffer: { codes: ["IMP010"], firstMessage: "Export not allowed", at: "t1" },
		served: false,
	};
	const sel = selectInjectionQuery(state, "help with parsing");
	assert.ok(sel.query);
	assert.equal(sel.from, "error");
});

test("selectInjectionQuery: served buffer falls back to prompt intent; nothing → null", () => {
	const served: CheckBufferState = {
		buffer: { codes: ["IMP010"], firstMessage: "x", at: "t1" },
		served: true,
	};
	assert.equal(selectInjectionQuery(served, "how do I pattern match in AILANG?").from, "intent");
	assert.equal(selectInjectionQuery(served, "unrelated but long prompt without triggers").from, null);

	const empty: CheckBufferState = { buffer: null, served: true };
	assert.equal(selectInjectionQuery(empty, "unrelated but long prompt without triggers").from, null);
});

test("selectInjectionQuery: >PROMPT_CAP_CHARS prompts are capped (shim parity: 4096)", () => {
	const paste = ("explain this AILANG code paste ").repeat(300); // 10200 chars
	const sel = selectInjectionQuery({ buffer: null, served: true }, paste);
	assert.equal(sel.from, "intent");
	assert.ok((sel.query as string).length <= PROMPT_CAP_CHARS);
	assert.ok((sel.query as string).endsWith("…"));
});

test("selectInjectionQuery: attempt clears (serve-on-attempt — no retry loops on below-floor)", () => {
	const state: CheckBufferState = {
		buffer: { codes: ["MOD010"], firstMessage: "module path", at: "t1" },
		served: false,
	};
	const after = markAttempted(state);
	assert.equal(after.served, true);
	assert.equal(selectInjectionQuery(after, "how do I concat strings in AILANG?").from, "intent");
});

// ---------- R10 corpus map / tool plumbing ----------

test("corporaToNamespacesCsv: known corpora map, unknown/omitted → engine default", () => {
	assert.equal(corporaToNamespacesCsv(undefined), undefined);
	assert.equal(corporaToNamespacesCsv("syntax"), "ailang-syntax");
	assert.equal(corporaToNamespacesCsv("builtins"), "ailang-builtins");
	assert.equal(corporaToNamespacesCsv("docs"), "ailang-docs"); // absent corpus degrades below-floor
	assert.equal(corporaToNamespacesCsv("nonexistent"), undefined);
});

// ---------- engine invocation contract ----------

test("buildEngineArgs: prompt via @tmpfile, session not leaked into args", () => {
	const args = buildEngineArgs("/tmp/xyz/p.txt");
	assert.deepEqual(args, ["micro-rag", "user-prompt", "--prompt", "@/tmp/xyz/p.txt"]);
	const withNs = buildEngineArgs("/tmp/xyz/p.txt", "ailang-docs");
	assert.deepEqual(withNs, ["micro-rag", "user-prompt", "--prompt", "@/tmp/xyz/p.txt", "--namespaces", "ailang-docs"]);
});

test("extractHitsFromEnvelope: cap 3 hits, ≤1KB excerpt each, back-flat fields", () => {
	const envelope = {
		hits: [
			{ tier: "simhash", score: 0.9, content: "a".repeat(2000), source: "std/string" },
			{ tier: "fts", score: 0.5, content: "b", source: "x" },
			{ tier: "fts", score: 0.4, content: "c", source: "y" },
			{ tier: "fts", score: 0.3, content: "d", source: "z" },
		],
	};
	const hits = extractHitsFromEnvelope(envelope);
	assert.equal(hits.length, HITS_CAP);
	assert.ok(hits[0].content.length <= HIT_EXCERPT_CHARS + 3); // + ellipsis
	assert.equal(hits[0].source, "std/string");
});

// ---------- shim parity (design doc Testing §2) ----------

test("shim parity: claude-code userprompt shim invokes the same engine route + flags", () => {
	const shim = readFileSync(join(here, "../../.claude/skills/microrag/frontends/claude-code/microrag_userprompt.sh"), "utf8");
	assert.match(shim, /micro-rag user-prompt --prompt/);
	assert.match(shim, /AILANG_MICRORAG_USERPROMPT_MIN_LEN/);
	assert.match(shim, /injection\.injection_text/);
});

// ---------- /resume reconstruction ----------

test("reconstructLastError: last unserved buffer restored, served entries never re-served", () => {
	const entries = [
		{ type: "custom", customType: "microrag:last_error", data: { codes: ["E_OLD"], firstMessage: "old", at: "t0", served: true } },
		{ type: "custom", customType: "quality:empty_content", data: {} },
		{ type: "custom", customType: "microrag:last_error", data: { codes: ["E_NEW"], firstMessage: "new", at: "t1", served: false } },
	];
	const s = reconstructLastError(entries);
	assert.ok(s.buffer);
	assert.deepEqual(s.buffer?.codes, ["E_NEW"]);
	assert.equal(s.served, false);

	const onlyServed = reconstructLastError([entries[0]!]);
	assert.ok(onlyServed.buffer); // retained for audit, but…
	assert.equal(onlyServed.served, true); // …never re-served
});

