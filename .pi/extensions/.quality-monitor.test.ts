/**
 * Table-driven unit tests for quality-monitor pure detectors (M-DX-QUALITY-MONITOR).
 * Run: node --experimental-strip-types --test .pi/extensions/.quality-monitor.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import {
	canonicalize,
	stableHash,
	isEmptyAssistantTurn,
	LoopWindow,
	buildLoopBlockReason,
	utf8Head,
	utf8Tail,
	excerptContent,
	buildEmptySteerMessage,
	reconstructState,
	EXCERPT_THRESHOLD_BYTES,
	EXCERPT_HEAD_BYTES,
	EXCERPT_TAIL_BYTES,
	LOOP_WINDOW_SIZE,
	LOOP_STRIKES,
	LOOP_BLOCK_CAP_PER_RUN,
	EMPTY_STEER_CAP_PER_RUN,
	MAX_LOOP_BLOCKS_PER_SESSION,
	MAX_STEERS_PER_SESSION,
	THINKING_FALLBACK_STRIKES,
} from "./quality-monitor.ts";

// ---------- stableHash / canonicalize ----------

test("stableHash: key order does not affect the hash (canonical input)", () => {
	assert.equal(stableHash({ a: 1, b: "x" }), stableHash({ b: "x", a: 1 }));
	assert.equal(
		stableHash({ tool: "bash", input: { command: "ls", timeout: 5 } }),
		stableHash({ input: { timeout: 5, command: "ls" }, tool: "bash" }),
	);
});

test("stableHash: distinct inputs never collide", () => {
	const base = { tool: "bash", input: { command: "cat big.log" } };
	assert.notEqual(stableHash(base), stableHash({ ...base, input: { command: "cat big2.log" } }));
	assert.notEqual(stableHash(base), stableHash({ tool: "read", input: { command: "cat big.log" } }));
	assert.notEqual(
		stableHash({ tool: "read", input: { path: "/a", offset: 1 } }),
		stableHash({ tool: "read", input: { path: "/a", offset: 2 } }),
	);
});

test("canonicalize: nested arrays/undefined/null are stable", () => {
	const x = { a: [1, { z: undefined, y: null }], b: undefined };
	assert.equal(canonicalize(x), canonicalize({ b: undefined, a: [1, { y: null, z: undefined }] }));
});

// ---------- isEmptyAssistantTurn (Q1) ----------

const reasoningOnly = (stopReason = "stop") => ({
	role: "assistant",
	content: [{ type: "thinking", thinking: "long deliberation..." }],
	stopReason,
});
const emptyContent = (stopReason = "stop") => ({ role: "assistant", content: [], stopReason });
const whitespaceOnly = (stopReason = "stop") => ({
	role: "assistant",
	content: [{ type: "text", text: "   \n\t" }],
	stopReason,
});
const withText = (stopReason = "stop") => ({
	role: "assistant",
	content: [{ type: "text", text: "Here is the answer." }],
	stopReason,
});
const withToolCall = (stopReason = "toolUse") => ({
	role: "assistant",
	content: [{ type: "toolCall", id: "t1", name: "bash", arguments: { command: "ls" } }],
	stopReason,
});

test("isEmptyAssistantTurn: empty arms flagged (zero-content class, 2026-08-26 incident)", () => {
	for (const msg of [reasoningOnly(), emptyContent(), whitespaceOnly()]) {
		assert.equal(isEmptyAssistantTurn(msg), true, JSON.stringify(msg).slice(0, 60));
	}
	// 'length' with no text = reasoning burned the whole budget (reasoning-token burn)
	assert.equal(isEmptyAssistantTurn(reasoningOnly("length")), true);
});

test("isEmptyAssistantTurn: non-empty arms never flagged", () => {
	assert.equal(isEmptyAssistantTurn(withText()), false);
	assert.equal(isEmptyAssistantTurn(withToolCall()), false); // toolUse lane = Q2's, not Q1's
	assert.equal(isEmptyAssistantTurn(withText("toolUse")), false);
});

test("isEmptyAssistantTurn: stopReason partition — error/aborted/missing never flagged", () => {
	// pi owns provider-error retry (V7); aborted is the user's action; missing means
	// malformed/unknown — D1: never attribute without evidence
	assert.equal(isEmptyAssistantTurn(emptyContent("error")), false);
	assert.equal(isEmptyAssistantTurn(emptyContent("aborted")), false);
	assert.equal(isEmptyAssistantTurn({ role: "assistant", content: [] }), false);
});

test("isEmptyAssistantTurn: role/name guards", () => {
	assert.equal(isEmptyAssistantTurn(undefined), false);
	assert.equal(isEmptyAssistantTurn(null), false);
	assert.equal(isEmptyAssistantTurn({ role: "user", content: [], stopReason: "stop" }), false);
	// tool-call-free but stopReason error → not ours
	assert.equal(isEmptyAssistantTurn({ role: "assistant", content: [{ type: "text", text: "" }], stopReason: "error" }), false);
});

// ---------- LoopWindow (Q2) ----------

test("LoopWindow: block on 3rd identical consecutive call (D3 + example t12/t14/t16)", () => {
	const w = new LoopWindow();
	w.push("h1"); // t12 — pass
	assert.equal(w.wouldBlock(), false);
	w.push("h1"); // t14 — pass
	assert.equal(w.wouldBlock(), false);
	w.push("h1"); // t16 — BLOCK
	assert.equal(w.wouldBlock(), true);
});

test("LoopWindow: distinct args never block; pattern restart clears the streak", () => {
	const w = new LoopWindow();
	w.push("a");
	w.push("a");
	w.push("b"); // distinct → streak resets
	assert.equal(w.wouldBlock(), false);
	w.push("a"); // restart like `grep A | grep A | grep B | grep A`
	assert.equal(w.wouldBlock(), false);
	w.push("a");
	w.push("a");
	assert.equal(w.wouldBlock(), true); // only after 3 consecutive again
});

test("LoopWindow: window eviction keeps memory bounded (size 8)", () => {
	const w = new LoopWindow();
	for (let i = 0; i < LOOP_WINDOW_SIZE * 10; i++) w.push(`h${i}`);
	assert.equal(w.wouldBlock(), false); // distinct hashes never accumulate a streak
});

test("LoopWindow: reset clears streak and window", () => {
	const w = new LoopWindow();
	w.push("h");
	w.push("h");
	w.push("h");
	assert.equal(w.wouldBlock(), true);
	w.reset();
	assert.equal(w.wouldBlock(), false);
	assert.equal(w.countInWindow, 0);
});

test("LoopWindow: tunables line up with the design doc (V-values)", () => {
	assert.equal(LOOP_WINDOW_SIZE, 8);
	assert.equal(LOOP_STRIKES, 3);
	assert.equal(LOOP_BLOCK_CAP_PER_RUN, 3);
	assert.equal(EMPTY_STEER_CAP_PER_RUN, 1);
	assert.equal(MAX_LOOP_BLOCKS_PER_SESSION, 20);
	assert.equal(MAX_STEERS_PER_SESSION, 5);
	assert.equal(THINKING_FALLBACK_STRIKES, 2);
});

test("buildLoopBlockReason: names tool + directive, never a bare refusal", () => {
	const reason = buildLoopBlockReason("read", 3);
	assert.match(reason, /read/);
	assert.match(reason, /3×/);
	assert.match(reason, /different/i);
});

// ---------- utf8Head / utf8Tail / excerptContent (Q3) ----------

test("utf8Head/Tail: ASCII passthrough under budget, sliced over", () => {
	const s = "x".repeat(100);
	assert.equal(utf8Head(s, 50), "x".repeat(50));
	assert.equal(utf8Head(s, 100), s);
	assert.equal(utf8Head(s, 150), s);
	assert.equal(utf8Tail(s, 20), "x".repeat(20));
});

test("utf8Head/Tail: multibyte boundary never emits replacement chars (é/emoji)", () => {
	const s = "é".repeat(100) + "🎉".repeat(50) + "log tail".repeat(10);
	for (const cut of [1, 2, 3, 4, 5, 7, 10, 33]) {
		const head = utf8Head(s, cut);
		const tail = utf8Tail(s, cut);
		assert.doesNotMatch(head, /\uFFFD/);
		assert.doesNotMatch(tail, /\uFFFD/);
		assert.ok(utf8len(head) <= cut + 3); // +3: one multibyte char boundary slack
	}
});

test("excerptContent: ≤16KB returns null (passthrough; handler returns undefined)", () => {
	assert.equal(excerptContent([{ type: "text", text: "x".repeat(EXCERPT_THRESHOLD_BYTES) }]), null);
	assert.equal(excerptContent([{ type: "text", text: "small" }]), null);
});

test("excerptContent: 90KB result → ≤2.2KB + directive + provenance (doc example)", () => {
	const big = "A".repeat(60 * 1024) + "M".repeat(29 * 1024) + "Z".repeat(1024);
	const patch = excerptContent([{ type: "text", text: big }]);
	assert.ok(patch);
	const text = patch!.content[0].text as string;
	assert.doesNotMatch(text, /MMMM/); // middle elided
	assert.match(text, /A{10,}/); // head kept
	assert.match(text, /Z{10,}/); // tail kept — errors live at the end
	assert.match(text, /re-run with narrower flags\/grep\/tail/);
	assert.equal(patch!.details.excerpted.original_bytes, 90 * 1024);
	assert.ok(patch!.details.excerpted.kept_bytes <= 2.2 * 1024, `kept=${patch!.details.excerpted.kept_bytes}`);
	assert.ok(utf8len(text) <= 2.2 * 1024);
});

test("excerptContent: head of FIRST + tail of LAST across multiple text parts; images preserved", () => {
	const content = [
		{ type: "text", text: "STDOUT-HEAD-" + "y".repeat(20 * 1024) },
		{ type: "image", source: { type: "base64", mediaType: "image/png", data: "AAAA" } },
		{ type: "text", text: "z".repeat(20 * 1024) + "-STDERR-TAIL exit=1" },
	];
	const patch = excerptContent(content, 16 * 1024);
	assert.ok(patch);
	assert.equal(patch!.content.filter((p) => p.type === "image").length, 1, "image untouched");
	const text = patch!.content.find((p) => p.type === "text")!.text as string;
	assert.match(text, /STDOUT-HEAD/);
	assert.match(text, /STDERR-TAIL exit=1/);
	assert.equal(patch!.details.excerpted.original_bytes, 40 * 1024 + 12 + 19);
});

test("excerptContent: string content treated as single part", () => {
	const patch = excerptContent("w".repeat(100 * 1024));
	assert.ok(patch);
	assert.match(patch!.content[0].text as string, /quality-monitor/);
});

function utf8len(s: string): number {
	return new TextEncoder().encode(s).length;
}

test("buildEmptySteerMessage: steer is instructive, capped shape (D2)", () => {
	const msg = buildEmptySteerMessage();
	assert.match(msg, /no visible content/);
	assert.match(msg, /plain text/);
});

// ---------- reconstructState (/resume parity, D6) ----------

test("reconstructState: counts from session entries; caps survive resume", () => {
	const entries = [
		{ type: "message", id: "a" },
		{
			type: "custom",
			id: "b",
			customType: "quality:empty_content",
			data: { model: "m", steered: true },
		},
		{
			type: "custom",
			id: "c",
			customType: "quality:empty_content",
			data: { model: "m", steered: false },
		},
		{ type: "custom", id: "d", customType: "quality:loop_block", data: { tool: "bash" } },
		{ type: "custom", id: "e", customType: "quality:excerpt", data: { tool: "bash" } },
		{ type: "custom", id: "f", customType: "quality:excerpt", data: { tool: "read" } },
		{
			type: "custom",
			id: "g",
			customType: "quality:thinking_fallback",
			data: { model: "deepseek-r1" },
		},
		{ type: "custom", id: "h", customType: "other-extension", data: { noise: 1 } },
	];
	const s = reconstructState(entries);
	assert.equal(s.emptyTotal, 2);
	assert.equal(s.steeredTotal, 1);
	assert.equal(s.loopBlocksTotal, 1);
	assert.equal(s.excerptsTotal, 2);
	assert.equal(s.thinkingFallbacksTotal, 1);
	assert.deepEqual(s.thinkingFallbackModels, ["deepseek-r1"]);
});

test("reconstructState: empty branch → zeroed state", () => {
	const s = reconstructState([]);
	assert.equal(s.emptyTotal, 0);
	assert.equal(s.excerptsTotal, 0);
});