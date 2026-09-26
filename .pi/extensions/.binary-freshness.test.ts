import { test } from "node:test";
import assert from "node:assert/strict";
import { parseVersionOutput, classify } from "./binary-freshness.ts";

test("parseVersionOutput: extracts commit, dirty flag, built time", () => {
	const v = parseVersionOutput("AILANG dev\nCommit: 46abe77\nFull: 46abe77a5-dirty\nBuilt:  2026-08-28T09:35:28Z");
	assert.equal(v?.commit, "46abe77");
	assert.equal(v?.dirty, true);
	assert.equal(v?.built, "2026-08-28T09:35:28Z");
	const v2 = parseVersionOutput("AILANG v0.35.0\nCommit: abcdef1\nFull: abcdef123456\nBuilt:  2026-08-28T09:35:28Z");
	assert.equal(v2?.dirty, false);
});

test("parseVersionOutput: no Commit line → null (fail-closed)", () => {
	assert.equal(parseVersionOutput("some other tool"), null);
});

test("classify: FRESH / STALE / DIRTY / UNKNOWN", () => {
	const v = { commit: "abc1234", dirty: false, built: "t" };
	assert.equal(classify(v, "abc1234567890", 0).status, "FRESH");
	assert.equal(classify(v, "def5678901234", 0).status, "STALE");
	assert.equal(classify(v, "abc1234567890", 3).status, "DIRTY");
	assert.equal(classify({ commit: "abc1234", dirty: true, built: "t" }, "abc1234567890", 0).status, "DIRTY");
	assert.equal(classify(null, "abc1234567890", 0).status, "UNKNOWN");
	assert.equal(classify(v, null, 0).status, "UNKNOWN");
	assert.match((classify(v, "def5678901234", 0) as { detail: string }).detail, /rebuild/);
});
