import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { initializeMissionSandbox, loadMissionPolicy, missionBashAllowed } from "./mission.ts";

test("mission policy requires a complete explicit source", () => {
	const dir = mkdtempSync(join(tmpdir(), "pi-policy-"));
	const file = join(dir, "policy.json");
	const valid = { filesystem: { allowWrite: ["."], denyWrite: [], denyRead: [] },
		network: { allowedDomains: [], deniedDomains: [] } };
	try {
		assert.throws(() => loadMissionPolicy(file));
		assert.throws(() => loadMissionPolicy("relative.json"));
		writeFileSync(file, "{");
		assert.throws(() => loadMissionPolicy(file));
		writeFileSync(file, JSON.stringify({ ...valid, enabled: false }));
		assert.throws(() => loadMissionPolicy(file));
		writeFileSync(file, JSON.stringify({ network: valid.network }));
		assert.throws(() => loadMissionPolicy(file));
		writeFileSync(file, JSON.stringify(valid));
		assert.deepEqual(loadMissionPolicy(file).filesystem.allowWrite, ["."]);
	} finally { rmSync(dir, { recursive: true, force: true }); }
});

test("ready marker follows successful initialization only", async () => {
	const dir = mkdtempSync(join(tmpdir(), "pi-ready-"));
	const ready = join(dir, "ready");
	try {
		await assert.rejects(initializeMissionSandbox(async () => { throw Error("init failed"); }, ready));
		assert.throws(() => readFileSync(ready));
		await initializeMissionSandbox(async () => {}, ready);
		assert.equal(readFileSync(ready, "utf8"), "ready\n");
	} finally { rmSync(dir, { recursive: true, force: true }); }
});

test("failed-init mission bash is refused", () => {
	let localCalls = 0;
	const bash = () => {
		if (!missionBashAllowed(true, false)) throw Error("Mission sandbox is not initialized");
		localCalls++;
	};
	assert.throws(bash, /not initialized/);
	assert.equal(localCalls, 0);
	assert.equal(missionBashAllowed(false, false), true);
});
