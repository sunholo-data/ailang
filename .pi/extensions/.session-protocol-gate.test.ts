/**
 * Table-driven unit tests for the session-protocol-gate predicate.
 * Run: node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import {
	shouldBlock,
	bashAllowed,
	headlessPrerequisitesMet,
	BLOCK_REASON,
} from "./session-protocol-gate.ts";

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

	// Nothing → missing both
	let r = headlessPrerequisitesMet([]);
	assert.equal(r.met, false);
	assert.equal(r.missing.length, 2);

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