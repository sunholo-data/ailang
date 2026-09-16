import { test } from "node:test";
import assert from "node:assert/strict";
import { composeEnvelope, fsSandboxOf, gateFromEnv, insideSandbox, lanePrompt, parsePolicyLine, policySummary } from "./ailang-exec.ts";

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
