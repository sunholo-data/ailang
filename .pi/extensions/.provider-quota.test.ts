import { test } from "node:test";
import assert from "node:assert/strict";
import { classifyQuota } from "./provider-quota.ts";

test("classifyQuota: thresholds 80/95", () => {
	assert.equal(classifyQuota(50, 200).status, "OK");
	assert.equal(classifyQuota(160, 200).status, "WARN");
	assert.equal(classifyQuota(197.53, 200).status, "CRITICAL");
	assert.equal(classifyQuota(200, 200).status, "CRITICAL");
	assert.equal(classifyQuota(201, 200).status, "CRITICAL");
});

test("classifyQuota: no limit → UNLIMITED; garbage → UNKNOWN", () => {
	assert.equal(classifyQuota(50, null).status, "UNLIMITED");
	assert.equal(classifyQuota(null, null).status, "UNKNOWN");
	assert.equal(classifyQuota(null, 200).status, "UNKNOWN");
});

test("classifyQuota: numbers", () => {
	const r = classifyQuota(197.53, 200);
	assert.equal(r.pct.toFixed(1), "98.8");
	assert.equal(r.remaining?.toFixed(2), "2.47");
});
