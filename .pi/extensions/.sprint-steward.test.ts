import { test } from "node:test";
import assert from "node:assert/strict";
import { applyMilestoneUpdate, validate, type SprintJson } from "./sprint-steward.ts";

const base: SprintJson = {
	sprint_id: "T", status: "not_started",
	features: [
		{ id: "M1_X", description: "d", estimated_loc: 10, dependencies: [], acceptance_criteria: ["a works"], passes: null, started: null, completed: null, notes: null },
		{ id: "M2_Y", description: "d", estimated_loc: 20, dependencies: ["M1_X"], acceptance_criteria: ["b works"], passes: null, started: null, completed: null, notes: null },
	],
	velocity: { estimated_total_loc: 30 },
};

test("applyMilestoneUpdate: passes + timestamps only; description untouched", () => {
	const next = applyMilestoneUpdate(base, "M1_X", { passes: true, notes: "done" });
	assert.equal(next.features[0].passes, true);
	assert.ok(next.features[0].completed);
	assert.equal(next.features[0].description, "d");
	assert.equal(next.features[1].passes, null);
	assert.equal(next.features[0].estimated_loc, 10);
});

test("applyMilestoneUpdate: refuses unknown milestone + placeholder state", () => {
	assert.throws(() => applyMilestoneUpdate(base, "NOPE", { passes: true }));
	const corrupt = JSON.parse(JSON.stringify(base)) as SprintJson;
	corrupt.features[0].id = "MILESTONE_ID";
	assert.throws(() => validate(corrupt));
	const badLoc = JSON.parse(JSON.stringify(base)) as SprintJson;
	badLoc.velocity.estimated_total_loc = 999;
	assert.throws(() => validate(badLoc));
});
