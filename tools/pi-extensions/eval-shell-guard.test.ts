import { test } from "node:test";
import assert from "node:assert/strict";
import guard, { capSeconds, clampTimeout, isUnboundedFind, REFUSAL } from "./eval-shell-guard.ts";

function load(env?: string) {
	const saved = process.env.AILANG_EVAL_BASH_CAP_S;
	if (env === undefined) delete process.env.AILANG_EVAL_BASH_CAP_S;
	else process.env.AILANG_EVAL_BASH_CAP_S = env;
	const handlers: Record<string, any> = {};
	try {
		guard({ on: (e: string, h: any) => { handlers[e] = h; } });
	} finally {
		if (saved === undefined) delete process.env.AILANG_EVAL_BASH_CAP_S;
		else process.env.AILANG_EVAL_BASH_CAP_S = saved;
	}
	return handlers;
}

// The two commands measured on the rig 2026-10-02, verbatim shapes.
test("the measured find / commands are refused with the stdlib lookup", async () => {
	const h = load();
	for (const command of [
		`find / -path '*stdlib*xml*' 2>/dev/null | head -20`,
		`ls ~/.ailang 2>&1; find / -name "ailing" -type f 2>/dev/null | head`,
		`which ailang; find / -path '*/std/xml.ail' 2>/dev/null | head`,
	]) {
		const input: any = { command };
		const out = await h.tool_call({ toolName: "bash", input });
		assert.deepEqual(out, { block: true, reason: REFUSAL }, command);
	}
	assert.match(REFUSAL, /ailang docs std\/<module>/);
});

test("home-rooted finds are refused; bounded and workspace finds are allowed", () => {
	const home = "/Users/someone";
	for (const c of ["find ~ -name x", "find $HOME -name x", "find ${HOME}/ -name x", `find ${home} -name x`, "sudo find / -name x", "x=$(find / -name y)"]) {
		assert.equal(isUnboundedFind(c, home), true, c);
	}
	for (const c of ["find . -name '*.ail'", "find / -maxdepth 2 -name x", "find ./src -type f", `find ${home}/work -name x`, "grep -r find /dev/null", "echo find /"]) {
		assert.equal(isUnboundedFind(c, home), false, c);
	}
});

test("every allowed bash call gets the 120s default cap", async () => {
	const h = load();
	const input: any = { command: "ailang run main.ail" };
	assert.equal(await h.tool_call({ toolName: "bash", input }), undefined);
	assert.equal(input.timeout, 120);
});

test("a timeout above the cap is clamped; one below it is kept", () => {
	assert.equal(clampTimeout(1800, 120), 120);
	assert.equal(clampTimeout(30, 120), 30);
	assert.equal(clampTimeout(undefined, 120), 120);
	assert.equal(clampTimeout(-5, 120), 120);
});

test("the cap is configurable and fails loud on a malformed value", () => {
	assert.equal(capSeconds({ AILANG_EVAL_BASH_CAP_S: "300" }), 300);
	assert.equal(capSeconds({}), 120);
	assert.throws(() => capSeconds({ AILANG_EVAL_BASH_CAP_S: "0" }));
	assert.throws(() => capSeconds({ AILANG_EVAL_BASH_CAP_S: "soon" }));
});

test("other tools are untouched, and the prompt states the limits", async () => {
	const h = load();
	const input: any = { path: "/" };
	assert.equal(await h.tool_call({ toolName: "read", input }), undefined);
	assert.equal(input.timeout, undefined);
	const out = await h.before_agent_start({ systemPrompt: "base" });
	assert.match(out.systemPrompt, /^base/);
	assert.match(out.systemPrompt, /stopped after 120s/);
	assert.match(out.systemPrompt, /ailang docs std\/<module>/);
});
