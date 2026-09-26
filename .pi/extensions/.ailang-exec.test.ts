import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
	cliRequest, composeEnvelope, gateFromEnv, gateWithSummary, insideSandbox, lanePrompt, parsePolicyLine,
	parseResultLine, register, teachingPrompt,
	type PolicySummary, type ToolRequest, type ToolResponse, type ToolRunner,
} from "./ailang-exec.ts";

// M-EXECUTOR-POLICY-HARDENING M4: the extension composes typed requests and
// relays the Go endpoint's answers; it parses no TOML and composes no argv.

const summary = (over: Partial<PolicySummary> = {}): PolicySummary => ({
	security_mode: "restricted", policy_digest: "d1", fs_sandbox: "/w", caps: ["IO", "FS"], net_allow: [], process_allow: [],
	cli: ["check", "iface", "docs_search"], ops: ["read", "write", "edit", "check", "iface", "docs_search"], timeout_ms: 5000, ...over,
});

test("gateFromEnv: unset env is default-deny with a readable reason", () => {
	const g = gateFromEnv({});
	assert.equal(g.policyPath, null);
	assert.match(g.refusal ?? "", /not granted/);
	assert.match(g.refusal ?? "", /AILANG_AGENT_POLICY/);
});

test("gateFromEnv: unreadable policy refuses by path", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/etc/resident/policy.toml" }, () => {
		throw new Error("ENOENT");
	});
	assert.match(g.refusal ?? "", /\/etc\/resident\/policy\.toml/);
});

test("gateWithSummary: D4 — policy inside its own sandbox is refused, from the Go summary", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/workspace/policy.toml" }, () => "x");
	const v = gateWithSummary(g, { ok: true, summary: summary({ fs_sandbox: "/workspace" }) });
	assert.match(v.gate.refusal ?? "", /rewrite the policy/);
	assert.equal(v.summary, null);
});

test("gateWithSummary: a policy the endpoint refuses to resolve is a refusal naming the reason", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/etc/p.toml" }, () => "x");
	const v = gateWithSummary(g, { ok: false, refused: "allowed_caps admits Process, which restricted mode…" });
	assert.match(v.gate.refusal ?? "", /does not resolve/);
	assert.match(v.gate.refusal ?? "", /Process/);
});

test("gateWithSummary: policy outside the sandbox is granted with its summary", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/etc/resident/policy.toml" }, () => "x");
	const v = gateWithSummary(g, { ok: true, summary: summary() });
	assert.equal(v.gate.refusal, null);
	assert.equal(v.summary?.fs_sandbox, "/w");
});

test("insideSandbox", () => {
	assert.equal(insideSandbox("/w/s", "/w/s/policy"), true);
	assert.equal(insideSandbox("/w/s", "/w/s"), true);
	assert.equal(insideSandbox("/w/s", "/w/sandbox2"), false); // prefix, not path, must not match
});

test("parsePolicyLine / parseResultLine pick their lines out of stderr", () => {
	const l = parsePolicyLine('warning: x\npolicy: {"ok":true,"policy_digest":"abc","decision":{"ok":true}}\n');
	assert.equal(l?.ok, true);
	assert.equal(l?.policy_digest, "abc");
	assert.equal(parsePolicyLine("nothing here"), null);
	assert.equal(parseResultLine('policy-result: {"version":1,"reason":"timeout"}\n')?.reason, "timeout");
});

test("composeEnvelope: admission keeps program stdout, strips the policy line", () => {
	const e = composeEnvelope(0, "hello\n", 'policy: {"ok":true,"policy_digest":"d1","decision":{"ok":true}}\n');
	assert.equal(e.admitted, true);
	assert.equal(e.stdout, "hello\n");
	assert.equal(e.policy_digest, "d1");
	assert.equal(e.stderr, "");
	assert.equal(e.limit, null);
});

test("composeEnvelope: a supervisor limit is carried beside the admission", () => {
	const e = composeEnvelope(3, "partial\n", 'policy: {"ok":true,"policy_digest":"d1","decision":{"ok":true}}\npolicy-result: {"version":1,"stage":"execute","reason":"timeout"}\n');
	assert.equal(e.admitted, true);
	assert.equal(e.exit_code, 3);
	assert.equal(e.limit?.reason, "timeout");
	assert.equal(e.stderr, "");
});

test("composeEnvelope: denial surfaces the decision, admitted=false", () => {
	const e = composeEnvelope(2, '{"file":"p.ail","decision":{"ok":false,"error_kind":"policy_violation","missing_from_policy":["Net"]}}\n', "");
	assert.equal(e.admitted, false);
	assert.equal((e.decision as { error_kind: string }).error_kind, "policy_violation");
});

test("composeEnvelope: a crash is not a refusal", () => {
	const e = composeEnvelope(1, "", "Error: --policy: --caps is not allowed");
	assert.equal(e.admitted, false);
	assert.equal(e.decision, null);
	assert.match(e.stderr, /--caps/);
});

test("lanePrompt: granted policy names the sandboxed tools, caps, sandbox, ops and the no-network rule", () => {
	const text = lanePrompt({ policyPath: "/etc/p.toml", refusal: null }, summary());
	assert.match(text, /NO shell/);
	assert.match(text, /ailang_read, ailang_edit, ailang_write/);
	assert.match(text, /ailang_run/);
	assert.match(text, /allowed effects = \{IO, FS\}/);
	assert.match(text, /confined to \/w/);
	assert.match(text, /no network access/);
	assert.match(text, /ops available under this policy: check, iface, docs_search/);
	assert.match(text, /effect ceiling violation in package/);
	assert.match(text, /timeout 5000 ms/);
});

test("lanePrompt: Process narrowed to its allowlist; Net names its hosts", () => {
	const text = lanePrompt({ policyPath: "/p", refusal: null }, summary({ caps: ["IO", "Process", "Net"], process_allow: ["git:pull"], net_allow: ["api.example"] }));
	assert.match(text, /Process is allowed only for: git:pull/);
	assert.match(text, /Net is allowed only to: api\.example/);
});

test("lanePrompt: no policy says execution is not granted, still allows writing", () => {
	const text = lanePrompt(gateFromEnv({}), null);
	assert.match(text, /NOT granted/);
	assert.match(text, /write and type-check/);
});

test("teachingPrompt: reads the active prompt via the binary, empty (not a throw) when it fails", () => {
	assert.equal(teachingPrompt(() => "# AILANG\nstuff\n"), "# AILANG\nstuff");
	assert.equal(teachingPrompt(() => { throw new Error("no ailang"); }), "");
	process.env.AILANG_LANE_TEACHING = "0";
	assert.equal(teachingPrompt(() => "x"), "");
	delete process.env.AILANG_LANE_TEACHING;
});

test("cliRequest: the tool's fields ARE the request; nothing is parsed out of a string", () => {
	assert.deepEqual(cliRequest({ op: "check", path: "a.ail" }), { op: "check", path: "a.ail" });
	assert.deepEqual(cliRequest({ op: "docs_search", query: "--limit 1 walk", flags: { limit: "3" } }), { op: "docs_search", query: "--limit 1 walk", flags: { limit: "3" } });
	assert.deepEqual(cliRequest({ op: "test", package: "." , flags: {} }), { op: "test", package: "." });
});

// ---- register(): the exact tool surface, and every call goes through the runner ----

const fakeType = { Object: (p: unknown) => p, String: () => "s", Array: () => "a", Record: () => "r", Optional: (x: unknown) => x };

function fakePi() {
	const hooks: Record<string, unknown[]> = {};
	const tools: Record<string, { execute: (...a: unknown[]) => Promise<{ details: Record<string, unknown> }> }> = {};
	return {
		api: {
			on: (name: string, fn: unknown) => { (hooks[name] ??= []).push(fn); },
			registerTool: (t: { name: string; execute: (...a: unknown[]) => Promise<{ details: Record<string, unknown> }> }) => { tools[t.name] = t; },
			exec: async () => ({ code: 0, stdout: "", stderr: "" }),
		},
		hooks, tools,
	};
}

function recordingRunner(reply: (req: ToolRequest) => ToolResponse): { runner: ToolRunner; calls: { policyPath: string; req: ToolRequest }[] } {
	const calls: { policyPath: string; req: ToolRequest }[] = [];
	return {
		calls,
		runner: async (policyPath, req) => { calls.push({ policyPath, req }); return reply(req); },
	};
}

test("no policy: tools register, refuse by reason, no prompt hook, runner never called", async () => {
	const f = fakePi();
	const r = recordingRunner(() => ({ ok: true }));
	await register(f.api as never, fakeType, {}, r.runner);
	assert.deepEqual(Object.keys(f.tools).sort(), ["ailang_cli", "ailang_edit", "ailang_read", "ailang_run", "ailang_write"]);
	assert.equal(f.hooks["before_agent_start"], undefined);
	const out = await f.tools["ailang_read"].execute("id", { path: "x" });
	assert.equal(out.details.ok, false);
	assert.match(String(out.details.refused), /not granted/);
	assert.equal(r.calls.length, 0);
});

test("with a policy: the summary comes from the endpoint, the hook names the lane, and every tool call is a typed request carrying the LAUNCHER's policy path", async () => {
	const dir = mkdtempSync(join(tmpdir(), "lane-"));
	const pol = join(dir, "policy.toml");
	writeFileSync(pol, "allowed_caps = [\"IO\"]\n");
	process.env.AILANG_LANE_TEACHING = "0";
	try {
		const f = fakePi();
		const r = recordingRunner((req) => (req.op === "summary" ? { ok: true, summary: summary({ fs_sandbox: join(dir, "ws") }) } : { ok: true, content: "data" }));
		await register(f.api as never, fakeType, { AILANG_AGENT_POLICY: pol }, r.runner);
		assert.equal(r.calls[0].req.op, "summary");
		assert.equal(r.calls[0].policyPath, pol);
		const hook = f.hooks["before_agent_start"]?.[0] as (ev: { systemPrompt: string }) => Promise<{ systemPrompt: string }>;
		assert.ok(hook, "hook must be registered when a policy is attached");
		assert.match((await hook({ systemPrompt: "base" })).systemPrompt, /Execution lane: ailang_only/);

		await f.tools["ailang_read"].execute("id", { path: "a.txt" });
		await f.tools["ailang_write"].execute("id", { path: "b.txt", content: "hi" });
		await f.tools["ailang_edit"].execute("id", { path: "b.txt", old_text: "hi", new_text: "ho" });
		await f.tools["ailang_cli"].execute("id", { op: "iface", module: "std/fs" });
		const reqs = r.calls.slice(1).map((c) => c.req);
		assert.deepEqual(reqs, [
			{ op: "read", path: "a.txt" },
			{ op: "write", path: "b.txt", content: "hi" },
			{ op: "edit", path: "b.txt", old_text: "hi", new_text: "ho" },
			{ op: "iface", module: "std/fs" },
		]);
		for (const c of r.calls) assert.equal(c.policyPath, pol, "the policy path is the launcher's, on every call");
	} finally {
		delete process.env.AILANG_LANE_TEACHING;
	}
});

test("with a policy the endpoint refuses: tools register but refuse with the endpoint's reason", async () => {
	const dir = mkdtempSync(join(tmpdir(), "lane-"));
	const pol = join(dir, "policy.toml");
	writeFileSync(pol, "allowed_caps = [\"IO\", \"Process\"]\n");
	const f = fakePi();
	const r = recordingRunner(() => ({ ok: false, refused: "allowed_caps admits Process, which restricted mode has no confined adapter for" }));
	await register(f.api as never, fakeType, { AILANG_AGENT_POLICY: pol }, r.runner);
	assert.equal(f.hooks["before_agent_start"], undefined);
	const out = await f.tools["ailang_cli"].execute("id", { op: "check", path: "x.ail" });
	assert.match(String(out.details.refused), /Process/);
	assert.equal(r.calls.length, 1, "only the summary was asked; a refused gate never forwards a call");
});

test("ailang_run: a program outside the sandbox is refused before anything runs", async () => {
	const dir = mkdtempSync(join(tmpdir(), "lane-"));
	const pol = join(dir, "policy.toml");
	writeFileSync(pol, "allowed_caps = [\"IO\", \"FS\"]\n");
	process.env.AILANG_LANE_TEACHING = "0";
	try {
		const f = fakePi();
		let execs = 0;
		f.api.exec = async () => { execs++; return { code: 0, stdout: "", stderr: "" }; };
		const r = recordingRunner((req) => (req.op === "summary" ? { ok: true, summary: summary({ fs_sandbox: join(dir, "ws") }) } : { ok: true }));
		await register(f.api as never, fakeType, { AILANG_AGENT_POLICY: pol }, r.runner);
		const out = await f.tools["ailang_run"].execute("id", { path: join(dir, "elsewhere", "x.ail") });
		assert.equal(out.details.admitted, false);
		assert.match(String(out.details.refused), /outside the FS sandbox/);
		assert.equal(execs, 0, "nothing was executed");
	} finally {
		delete process.env.AILANG_LANE_TEACHING;
	}
});
