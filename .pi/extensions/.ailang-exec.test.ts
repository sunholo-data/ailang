import { test } from "node:test";
import assert from "node:assert/strict";
import { CLI_DEFAULT_ALLOW, CLI_GATE_ONLY, cliDecision, composeEnvelope, fsSandboxOf, gateFromEnv, insideSandbox, lanePrompt, parsePolicyLine, policySummary, teachingPrompt } from "./ailang-exec.ts";

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

test("gateFromEnv: D4 — policy inside its own sandbox is refused", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/workspace/policy.toml" }, () => 'fs_sandbox = "/workspace"\nallowed_caps = ["IO"]\n');
	assert.match(g.refusal ?? "", /rewrite the policy/);
});

test("gateFromEnv: policy outside the sandbox is granted", () => {
	const g = gateFromEnv({ AILANG_AGENT_POLICY: "/etc/resident/policy.toml" }, () => 'fs_sandbox = "/workspace"\n');
	assert.equal(g.refusal, null);
	assert.equal(g.policyPath, "/etc/resident/policy.toml");
});

test("fsSandboxOf / insideSandbox", () => {
	assert.equal(fsSandboxOf('entry = "main"\nfs_sandbox = "/w/s"\n'), "/w/s");
	assert.equal(fsSandboxOf("entry = \"main\"\n"), null);
	assert.equal(insideSandbox("/w/s", "/w/s/policy"), true);
	assert.equal(insideSandbox("/w/s", "/w/s"), true);
	assert.equal(insideSandbox("/w/s", "/w/sandbox2"), false); // prefix, not path, must not match
});

test("parsePolicyLine picks the admission line out of stderr", () => {
	const l = parsePolicyLine('warning: x\npolicy: {"ok":true,"policy_digest":"abc","decision":{"ok":true}}\n');
	assert.equal(l?.ok, true);
	assert.equal(l?.policy_digest, "abc");
	assert.equal(parsePolicyLine("nothing here"), null);
});

test("composeEnvelope: admission keeps program stdout, strips the policy line", () => {
	const e = composeEnvelope(0, "hello\n", 'policy: {"ok":true,"policy_digest":"d1","decision":{"ok":true}}\n');
	assert.equal(e.admitted, true);
	assert.equal(e.stdout, "hello\n");
	assert.equal(e.policy_digest, "d1");
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

test("policySummary reads caps, sandbox and net_allow", () => {
	const p = policySummary('allowed_caps = ["IO", "FS"]\nfs_sandbox = "/w"\nnet_allow = ["api.example.com"]\nentry = "main"\n');
	assert.deepEqual(p.caps, ["IO", "FS"]);
	assert.equal(p.sandbox, "/w");
	assert.deepEqual(p.net, ["api.example.com"]);
	assert.deepEqual(policySummary('process_allow = ["git:pull", "git:status"]\n').process, ["git:pull", "git:status"]);
});

test("lanePrompt: Process narrowed to its allowlist", () => {
	const text = lanePrompt({ policyPath: "/p", refusal: null }, () => 'allowed_caps = ["IO", "Process"]\nprocess_allow = ["git:pull"]\n');
	assert.match(text, /Process is allowed only for: git:pull/);
});

test("lanePrompt: granted policy names tools, caps, sandbox and the no-network rule", () => {
	const text = lanePrompt({ policyPath: "/etc/p.toml", refusal: null }, () => 'allowed_caps = ["IO", "FS"]\nfs_sandbox = "/w"\n');
	assert.match(text, /NO shell/);
	assert.match(text, /ailang_run/);
	assert.match(text, /allowed effects = \{IO, FS\}/);
	assert.match(text, /confined to \/w/);
	assert.match(text, /no network access/);
});

test("lanePrompt: no policy says execution is not granted, still allows writing", () => {
	const text = lanePrompt(gateFromEnv({}));
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

// ---- ailang_cli: the allowlisted rest of the CLI ---------------------------

test("cliDecision: default set admits the read-only surface and refuses the rest", () => {
	assert.equal(cliDecision(["iface", "std/fs"], null, null).ok, true);
	assert.equal(cliDecision(["docs", "search", "walk"], null, null).ok, true);
	assert.equal(cliDecision(["ai-check", "report.ail"], null, "/w").ok, true);
	// docs is admitted only as `docs search` — another docs subcommand is not
	assert.match(cliDecision(["docs", "serve"], null, null).reason ?? "", /not in the policy/);
	for (const cmd of ["messages", "coordinator", "install", "publish", "eval-suite", "brain", "pi", "mission"]) {
		const d = cliDecision([cmd, "x"], null, null);
		assert.equal(d.ok, false, cmd);
		assert.match(d.reason ?? "", /not in the policy's cli_allow/);
	}
});

test("cliDecision: test is allowed — the runner evaluates pure code only", () => {
	assert.equal(cliDecision(["test", "--package", "."], null, "/w").ok, true);
	assert.equal(cliDecision(["test", "hello_test.ail"], null, "/w").ok, true);
});

test("cliDecision: run/exec/repl are refused even when the operator lists them", () => {
	for (const cmd of CLI_GATE_ONLY) {
		const d = cliDecision([cmd, "x.ail"], [cmd, "iface"], "/w");
		assert.equal(d.ok, false, cmd);
		assert.match(d.reason ?? "", /ailang_run/);
	}
	assert.ok(!CLI_DEFAULT_ALLOW.some((e) => CLI_GATE_ONLY.includes(e.split(":")[0])), "default set never names a gate-only command");
});

test("cliDecision: an explicit cli_allow replaces the default — empty list refuses everything", () => {
	assert.equal(cliDecision(["iface", "std/fs"], [], null).ok, false);
	assert.equal(cliDecision(["fmt", "a.ail"], ["fmt"], null).ok, true);
	assert.equal(cliDecision(["iface", "std/fs"], ["fmt"], null).ok, false);
	// cmd:sub narrows: `docs:search` admits `docs search`, not `docs`
	assert.equal(cliDecision(["docs", "search", "q"], ["docs:search"], null).ok, true);
	assert.equal(cliDecision(["docs"], ["docs:search"], null).ok, false);
});

test("cliDecision: path arguments must stay inside the sandbox", () => {
	assert.equal(cliDecision(["fmt", "src/a.ail"], null, "/w").ok, true);
	assert.equal(cliDecision(["fmt", "/w/src/a.ail"], null, "/w").ok, true);
	assert.match(cliDecision(["fmt", "../../etc/passwd"], null, "/w").reason ?? "", /outside the FS sandbox/);
	assert.match(cliDecision(["fmt", "/etc/x.ail"], null, "/w").reason ?? "", /outside the FS sandbox/);
	// flags and bare words (module names, queries) are not paths
	assert.equal(cliDecision(["iface", "--json", "std/fs"], null, "/w").ok, true);
	assert.equal(cliDecision(["docs", "search", "how to walk"], null, "/w").ok, true);
	assert.equal(cliDecision([], null, "/w").ok, false);
});

test("policySummary: cli_allow absent is null (default set), present is the list, empty is []", () => {
	assert.equal(policySummary('allowed_caps = ["IO"]\n').cli, null);
	assert.deepEqual(policySummary('cli_allow = ["iface", "docs:search"]\n').cli, ["iface", "docs:search"]);
	assert.deepEqual(policySummary('cli_allow = []\n').cli, []);
});

test("lanePrompt: names ailang_cli, the package-ceiling rule, and the allowed subcommands", () => {
	const toml = 'allowed_caps = ["IO", "FS"]\nfs_sandbox = "/w"\ncli_allow = ["iface", "fmt"]\n';
	const p = lanePrompt({ policyPath: "/p/policy.toml", refusal: null }, () => toml);
	assert.match(p, /ailang_cli/);
	assert.match(p, /effect ceiling violation in package/);
	assert.match(p, /`\.ailang-scratch\/` at the sandbox root/);
	assert.match(p, /NEVER edit a package's `\[effects\] max`/);
	assert.match(p, /ailang_cli may run only these subcommands: iface, fmt/);
	const q = lanePrompt({ policyPath: "/p/policy.toml", refusal: null }, () => 'allowed_caps = ["IO"]\n');
	assert.match(q, /ailang_cli may run only these subcommands: check, ai-check, iface/);
});

// ---- the lane prompt is injected ONLY when a policy is attached ----------
// A plain `pi` on the rig loads this same suite and HAS a shell; telling it
// otherwise made a local session refuse work it could do (2026-09-19).

import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { register } from "./ailang-exec.ts";
const fakeType = { Object: (p: unknown) => p, String: () => "s", Array: () => "a", Optional: (x: unknown) => x };

function fakePi() {
	const hooks: Record<string, unknown[]> = {};
	const tools: string[] = [];
	return {
		api: {
			on: (name: string, fn: unknown) => { (hooks[name] ??= []).push(fn); },
			registerTool: (t: { name: string }) => { tools.push(t.name); },
			exec: async () => ({ code: 0, stdout: "", stderr: "" }),
		},
		hooks, tools,
	};
}

test("no policy: tools register, no prompt hook, so a shell-bearing session is not told it has none", async () => {
	const saved = process.env.AILANG_AGENT_POLICY;
	delete process.env.AILANG_AGENT_POLICY;
	try {
		const f = fakePi();
		await register(f.api as never, fakeType, process.env);
		assert.deepEqual(f.tools.sort(), ["ailang_cli", "ailang_run"]);
		assert.equal(f.hooks["before_agent_start"], undefined);
	} finally {
		if (saved !== undefined) process.env.AILANG_AGENT_POLICY = saved;
	}
});

test("with a policy: the prompt hook is registered and names the lane", async () => {
	const dir = mkdtempSync(join(tmpdir(), "lane-"));
	const pol = join(dir, "policy.toml");
	writeFileSync(pol, `allowed_caps = ["IO"]\nfs_sandbox = "${join(dir, "ws")}"\nentry = "main"\n`);
	const saved = process.env.AILANG_AGENT_POLICY;
	process.env.AILANG_AGENT_POLICY = pol;
	process.env.AILANG_LANE_TEACHING = "0";
	try {
		const f = fakePi();
		await register(f.api as never, fakeType, process.env);
		const hook = f.hooks["before_agent_start"]?.[0] as (ev: { systemPrompt: string }) => Promise<{ systemPrompt: string }>;
		assert.ok(hook, "hook must be registered when a policy is attached");
		const out = await hook({ systemPrompt: "base" });
		assert.match(out.systemPrompt, /Execution lane: ailang_only/);
	} finally {
		if (saved !== undefined) process.env.AILANG_AGENT_POLICY = saved; else delete process.env.AILANG_AGENT_POLICY;
		delete process.env.AILANG_LANE_TEACHING;
	}
});
