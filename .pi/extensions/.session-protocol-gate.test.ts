/**
 * Table-driven unit tests for the session-protocol-gate predicate.
 * Run: node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
	chmodSync,
	mkdirSync,
	mkdtempSync,
	readFileSync,
	rmSync,
	writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import {
	shouldBlock,
	bashAllowed,
	headlessPrerequisitesMet,
	BLOCK_REASON,
} from "./session-protocol-gate.ts";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");

test(
	"mission_pi_run: child receives canonical messaging environment and preserves storage",
	{ skip: process.platform === "win32" },
	() => {
		const fixture = mkdtempSync(join(tmpdir(), "ailang-mission-pi-env-"));
		try {
			const binDir = join(fixture, "bin");
			mkdirSync(binDir);
			const fakePi = join(binDir, "pi");
			writeFileSync(
				fakePi,
				`#!/usr/bin/env bash
set -u
printf '%s\\n' "\${AILANG_MESSAGES_STORE-__UNSET__}" "\${AILANG_MESSAGES_PROJECT-__UNSET__}" "\${AILANG_STORAGE-__UNSET__}" > child-env.txt
printf '%s\\n' '{"type":"agent_end"}'
`,
			);
			chmodSync(fakePi, 0o755);

			const directive = join(fixture, "directive.txt");
			writeFileSync(directive, "MISSION-ROLE: evaluator\n");
			const runner = join(repoRoot, "scripts", "mission_pi_run.sh");

			const arms = [
				{
					name: "unset caller",
					caller: {},
					expectedStorage: "__UNSET__",
				},
				{
					name: "hostile caller",
					caller: {
						AILANG_MESSAGES_STORE: "local",
						AILANG_MESSAGES_PROJECT: "wrong-project",
						AILANG_STORAGE: "local",
					},
					expectedStorage: "local",
				},
			];

			for (const arm of arms) {
				const worktree = join(fixture, `worktree-${arm.name.replaceAll(" ", "-")}`);
				mkdirSync(worktree);
				const init = spawnSync("git", ["init", "-q", worktree], { encoding: "utf8" });
				assert.equal(init.status, 0, `${arm.name}: git init: ${init.stderr}`);

				const out = join(fixture, `${arm.name.replaceAll(" ", "-")}.ndjson`);
				const env = {
					...process.env,
					PATH: `${binDir}:${process.env.PATH ?? ""}`,
					MISSION_PI_POLL_SECONDS: "1",
				};
				delete env.AILANG_MESSAGES_STORE;
				delete env.AILANG_MESSAGES_PROJECT;
				delete env.AILANG_STORAGE;
				Object.assign(env, arm.caller);

				const run = spawnSync(
					"bash",
					[
						runner,
						"--model",
						"fake/provider",
						"--directive",
						directive,
						"--workdir",
						worktree,
						"--out",
						out,
						"--max-seconds",
						"5",
						"--stall-seconds",
						"5",
					],
					{ encoding: "utf8", env, timeout: 15_000 },
				);
				assert.equal(
					run.status,
					0,
					`${arm.name}: runner rc=${run.status}, signal=${run.signal}, stderr=${run.stderr}`,
				);

				const receipt = readFileSync(join(worktree, "child-env.txt"), "utf8")
					.trimEnd()
					.split("\n");
				assert.deepEqual(
					receipt,
					["gcp", "ailang-multivac", arm.expectedStorage],
					`${arm.name}: exact child environment receipt`,
				);
				assert.match(readFileSync(out, "utf8"), /"type":"agent_end"/);
			}
		} finally {
			rmSync(fixture, { recursive: true, force: true });
		}
	},
);

test("shouldBlock: edit/write blocked absolutely while armed", () => {
	assert.equal(shouldBlock("edit", undefined, false), BLOCK_REASON);
	assert.equal(shouldBlock("write", undefined, false), BLOCK_REASON);
});

test("shouldBlock: everything unlocked once acked", () => {
	for (const tool of ["edit", "write", "bash", "read", "grep"]) {
		assert.equal(shouldBlock(tool, "anything", true), null);
	}
});

test("shouldBlock: read-only tools never blocked while armed", () => {
	for (const tool of ["read", "grep", "glob", "ls", "custom_other_extension_tool"]) {
		assert.equal(shouldBlock(tool, undefined, false), null);
	}
});

test("bash: allowlisted read-only commands pass while armed", () => {
	for (const cmd of [
		"ls -la",
		"cat CLAUDE.md",
		"grep -rn gate internal/",
		"git status",
		"git log --oneline -5",
		"ailang messages list --unread",
		"ailang check foo.ail",
		"cat a.md && ls", // compound, all segments allowlisted
	]) {
		assert.equal(bashAllowed(cmd), true, `expected allowed: ${cmd}`);
		assert.equal(shouldBlock("bash", cmd, false), null, cmd);
	}
});

test("shouldBlock: fail-closed on write vectors while armed", () => {
	for (const cmd of [
		"python -c 'open(\"x\",\"w\").write(1)'",
		"dd if=/dev/zero of=x",
		"cp a b",
		"mv a b",
		"git push origin dev",
		"rm -rf x",
		"echo hi > file",
		"cat a && rm a", // compound with a non-allowlisted tail
		"make test",
		undefined, // missing command → fail-closed
		"", // empty → fail-closed
	]) {
		assert.equal(shouldBlock("bash", cmd as string | undefined, false), BLOCK_REASON, String(cmd));
	}
});

test("shouldBlock: unknown tools pass (F4 — other extensions' tools out of scope v1)", () => {
	assert.equal(shouldBlock("some_extension_tool", undefined, false), null);
});

test("headlessPrerequisitesMet: requires both CLAUDE.md and ailang messages evidence", () => {
	const readCall = {
		message: {
			role: "assistant",
			content: [
				{
					type: "toolCall",
					name: "read",
					arguments: { path: "/repo/CLAUDE.md" },
				},
			],
		},
	};
	const messagesCall = {
		message: {
			role: "assistant",
			content: [
				{
					type: "toolCall",
					name: "bash",
					arguments: { command: "ailang messages list --unread" },
				},
			],
		},
	};
	const bashCatCall = {
		message: {
			role: "assistant",
			content: [
				{
					type: "toolCall",
					name: "bash",
					arguments: { command: "cat CLAUDE.md | head -5" },
				},
			],
		},
	};
	const blockNoise = {
		message: {
			role: "toolResult",
			toolName: "bash",
			content: [{ type: "text", text: BLOCK_REASON }], // block text mentions CLAUDE.md — must NOT count
		},
	};

	// Nothing → every step missing. Assert on CONTENT, not on a count: the count
	// broke when M1a made the AILANG set the generic floor PLUS a CLAUDE.md read
	// (2 → 3), which is a correct change that a brittle length assertion flagged
	// as a regression. What matters is WHICH steps are reported.
	let r = headlessPrerequisitesMet([]);
	assert.equal(r.met, false);
	assert.ok(r.missing.some((m) => m.includes("CLAUDE.md")), "must name the CLAUDE.md step");
	assert.ok(r.missing.some((m) => m.includes("ailang messages")), "must name the inbox step");
	assert.ok(r.missing.some((m) => m.includes("inspect the workspace")), "must name the orientation step");

	// Block-reason noise must not satisfy the CLAUDE.md step
	r = headlessPrerequisitesMet([blockNoise, messagesCall]);
	assert.equal(r.met, false);

	// Both steps (read variant) → met
	r = headlessPrerequisitesMet([readCall, messagesCall]);
	assert.equal(r.met, true);

	// Both steps (bash cat variant) → met
	r = headlessPrerequisitesMet([bashCatCall, messagesCall]);
	assert.equal(r.met, true);
});
