/**
 * Table-driven unit tests for workspace-trust (incl. per-repo machine config).
 * Run: node --experimental-strip-types --test .pi/extensions/.workspace-trust.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import { execSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
	DEFAULT_REMOTE_PATTERNS,
	KILL_SWITCH_ENV,
	configPath,
	decide,
	loadConfig,
	originRemote,
} from "./workspace-trust.ts";

/** ExecResult-shaped stub that shells out synchronously, like pi.exec does async. */
const realExec = async (cmd: string, args: string[], opts: { timeout: number }) => ({
	stdout: execSync([cmd, ...args].join(" "), { timeout: opts.timeout }).toString(),
});

const FAMILY = "https://github.com/sunholo-data/ailang.git";

test("decide: trusts sunholo-family remotes (https + ssh forms), never remembers", () => {
	for (const remote of [
		"https://github.com/sunholo-data/ailang.git",
		"git@github.com:sunholo-data/ailang.git",
		"https://github.com/sunholo/email.git",
		"https://github.com/sunholo-websites/site.git",
		"git@github.com:sunholo-voight-kampff/ailang-fork.git",
		"https://github.com/arniwesth/motoko_agent.git",
	]) {
		const r = decide({ remote });
		assert.equal(r.trusted, "yes", `expected yes for ${remote}`);
		assert.notEqual(r.remember, true, "must never write trust.json entries");
	}
});

test("decide: non-family remotes fall through undecided (never 'no')", () => {
	for (const remote of [
		"https://github.com/torvalds/linux.git",
		"https://github.com/acme-corp/website.git",
	]) {
		assert.equal(decide({ remote }).trusted, "undecided", `expected undecided for ${remote}`);
	}
	assert.equal(decide({ remote: null }).trusted, "undecided");
	assert.equal(decide({ remote: "" }).trusted, "undecided");
});

test("decide: kill switch abstains even for family remotes", () => {
	const r = decide({ remote: FAMILY, killSwitch: "0" });
	assert.equal(r.trusted, "undecided");
	// anything other than "0" (unset, "1", whitespace) leaves the gate active
	assert.equal(decide({ remote: FAMILY, killSwitch: " 1 " }).trusted, "yes");
});

test("decide: env patterns are additive and case-insensitive (org/repo coordinates match URL forms)", () => {
	const remote = "https://github.com/Example-Corp/Example-Repo.git";
	assert.equal(decide({ remote, extraRemotes: "example-corp" }).trusted, "yes");
	assert.equal(decide({ remote, extraRemotes: "Example-Corp/Example-Repo" }).trusted, "yes");
	assert.equal(decide({ remote, extraRemotes: "other-org,," }).trusted, "undecided");
	// env is additive on top of defaults
	assert.equal(decide({ remote: FAMILY, extraRemotes: "example-corp" }).trusted, "yes");
});

test("decide: config patterns REPLACE defaults; empty array disables defaults; env stays additive", () => {
	// config replaces defaults → family remote no longer matches by default
	assert.equal(decide({ remote: FAMILY, configPatterns: ["acme-corp/one"] }).trusted, "undecided");
	// the configured repo matches
	assert.equal(
		decide({ remote: "https://github.com/acme-corp/one.git", configPatterns: ["acme-corp/one"] }).trusted,
		"yes",
	);
	// {"remotes": []} → no default patterns; env still adds
	assert.equal(decide({ remote: FAMILY, configPatterns: [] }).trusted, "undecided");
	assert.equal(decide({ remote: FAMILY, configPatterns: [], extraRemotes: "sunholo-data/ailang" }).trusted, "yes");
	// null config = file absent → defaults apply
	assert.equal(decide({ remote: FAMILY, configPatterns: null }).trusted, "yes");
});

test("decide: config warning abstains even for matching remotes (fail-closed)", () => {
	assert.equal(
		decide({ remote: FAMILY, configPatterns: ["sunholo"], configWarning: "invalid JSON — boom" }).trusted,
		"undecided",
	);
	// kill switch outranks everything
	assert.equal(decide({ remote: FAMILY, killSwitch: "0", configWarning: null }).trusted, "undecided");
});

test("decide: never returns 'no' across the whole input space", () => {
	const results = [
		decide({ remote: FAMILY }),
		decide({ remote: "https://github.com/torvalds/linux.git" }),
		decide({ remote: null }),
		decide({ remote: FAMILY, killSwitch: "0" }),
		decide({ remote: "x", extraRemotes: "x" }),
		decide({ remote: "x", configPatterns: ["x"] }),
		decide({ remote: "x", configWarning: "broken" }),
	];
	for (const r of results) {
		assert.notEqual(r.trusted, "no", "workspace-trust must never suppress pi's own flow with 'no'");
	}
});

test("decide: default patterns cover the fleet's built-ins", () => {
	assert.deepEqual(DEFAULT_REMOTE_PATTERNS, ["sunholo", "arniwesth/motoko_agent"]);
});

test("loadConfig: absent file (ENOENT) → defaults marker, no warning", () => {
	const missing = (p: string): string => {
		throw Object.assign(new Error("no such file"), { code: "ENOENT" });
	};
	const cfg = loadConfig(missing, "/home/tester");
	assert.equal(cfg.patterns, null);
	assert.equal(cfg.warning, null);
	assert.equal(configPath("/home/tester"), "/home/tester/.pi/agent/workspace-trust.json");
});

test("loadConfig: valid file parses cleanly", () => {
	const raw = JSON.stringify({ remotes: ["acme-corp/one", "other/two"] });
	const cfg = loadConfig(
		(p) => {
			assert.equal(p, configPath("/home/tester"));
			return raw;
		},
		"/home/tester",
	);
	assert.deepEqual(cfg.patterns, ["acme-corp/one", "other/two"]);
	assert.equal(cfg.warning, null);
});

test("loadConfig: invalid JSON, wrong shape, junk entries, unreadable → warning (abstain)", () => {
	const cases: Array<{ name: string; raw: string | Error }> = [
		{ name: "invalid JSON", raw: "{not json" },
		{ name: "remotes not an array", raw: JSON.stringify({ remotes: "sunholo" }) },
		{ name: "non-string entries", raw: JSON.stringify({ remotes: ["ok", 42, null] }) },
		{ name: "empty-string entries", raw: JSON.stringify({ remotes: ["ok", ""] }) },
		{ name: "unreadable", raw: Object.assign(new Error("EACCES: permission denied"), { code: "EACCES" }) },
	];
	for (const c of cases) {
		const cfg = loadConfig(() => {
			if (c.raw instanceof Error) throw c.raw;
			return c.raw;
		}, "/home/tester");
		assert.ok(cfg.warning, `${c.name}: expected a warning`);
		assert.equal(cfg.patterns, null, `${c.name}: patterns must be null`);
	}
});

test("originRemote: resolves origin in a real git repo; null outside one; null on exec failure", async () => {
	const dir = mkdtempSync(join(tmpdir(), "wt-trust-test-"));
	try {
		execSync(`git -C ${dir} init -q`);
		execSync(`git -C ${dir} remote add origin https://github.com/sunholo-data/ailang.git`);
		const url = await originRemote(dir, realExec);
		assert.match(url ?? "", /sunholo-data\/ailang\.git$/);

		const outside = mkdtempSync(join(tmpdir(), "wt-trust-norepo-"));
		try {
			assert.equal(await originRemote(outside, realExec), null);
		} finally {
			rmSync(outside, { recursive: true, force: true });
		}

		const throwingExec = async () => {
			throw new Error("simulated subprocess failure");
		};
		assert.equal(await originRemote(dir, throwingExec), null);
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
});

test("KILL_SWITCH_ENV names the documented switch", () => {
	assert.equal(KILL_SWITCH_ENV, "PI_WORKSPACE_TRUST");
});