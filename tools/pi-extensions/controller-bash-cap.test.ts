import { test } from "node:test";
import assert from "node:assert/strict";
import cap, { capSeconds, clampTimeout } from "./controller-bash-cap.ts";

function load(env?: string) {
	const saved = process.env.MISSION_PI_BASH_CAP_S;
	if (env === undefined) delete process.env.MISSION_PI_BASH_CAP_S;
	else process.env.MISSION_PI_BASH_CAP_S = env;
	const handlers: Record<string, any> = {};
	try {
		cap({ on: (e: string, h: any) => { handlers[e] = h; } });
	} finally {
		if (saved === undefined) delete process.env.MISSION_PI_BASH_CAP_S;
		else process.env.MISSION_PI_BASH_CAP_S = saved;
	}
	return handlers;
}

test("an uncapped bash call gets the default 540s cap", async () => {
	const h = load();
	const input: any = { command: "for i in 1 2 3 4 5 6; do sleep 280; done" };
	await h.tool_call({ toolName: "bash", input });
	assert.equal(input.timeout, 540);
});

test("a timeout above the cap is clamped; one below it is kept", async () => {
	const h = load();
	const big: any = { command: "x", timeout: 1800 };
	const small: any = { command: "x", timeout: 30 };
	await h.tool_call({ toolName: "bash", input: big });
	await h.tool_call({ toolName: "bash", input: small });
	assert.equal(big.timeout, 540);
	assert.equal(small.timeout, 30);
});

test("the cap stays under the watchdog's 600s flat window", () => {
	assert.ok(capSeconds({}) < 5 * 120);
});

test("non-bash tools are untouched", async () => {
	const h = load();
	const input: any = { path: "a.go" };
	await h.tool_call({ toolName: "read", input });
	assert.equal(input.timeout, undefined);
});

test("the env override applies, and a malformed one fails loudly", async () => {
	const h = load("120");
	const input: any = { command: "x" };
	await h.tool_call({ toolName: "bash", input });
	assert.equal(input.timeout, 120);
	assert.throws(() => load("ten"), /MISSION_PI_BASH_CAP_S/);
	assert.throws(() => load("0"), /MISSION_PI_BASH_CAP_S/);
});

test("garbage timeouts are replaced by the cap", () => {
	for (const bad of [0, -5, NaN, Infinity, "600", null]) assert.equal(clampTimeout(bad, 540), 540);
});

test("the system prompt tells the model how to wait", async () => {
	const h = load();
	const r = await h.before_agent_start({ systemPrompt: "BASE" });
	assert.match(r.systemPrompt, /^BASE/);
	assert.match(r.systemPrompt, /540s/);
	assert.match(r.systemPrompt, /SEPARATE short commands/);
});
