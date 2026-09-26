/**
 * M-COORDINATOR-EXECUTION-TRUST M1a — tiering + the built-in prerequisite floor.
 * Run: node --experimental-strip-types --test .pi/extensions/.session-gate-tiers.test.ts
 * (dot-prefixed so pi's project-extension discovery skips this file)
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import {
	shouldBlock,
	prerequisitesMet,
	prereqSetForWorkspace,
	BLOCK_REASON,
	type GateContext,
} from "./session-protocol-gate.ts";

const tier1 = (met: boolean): GateContext => ({ tier: "tier1", prereqsMet: met });
const tier2 = (met: boolean): GateContext => ({ tier: "tier2", prereqsMet: met });

/** A session branch containing the given tool calls, in pi's shape. */
function branch(...calls: Array<{ name: string; args: Record<string, string> }>) {
	return calls.map((c) => ({
		message: {
			role: "assistant",
			content: [{ type: "toolCall", name: c.name, arguments: c.args }],
		},
	}));
}

// ---- MU-1: tier 1 disarms on verified prerequisites, with no ack ----

test("tier1 + prerequisites met unlocks mutation without an ack", () => {
	assert.equal(shouldBlock("edit", undefined, false, tier1(true)), null);
	assert.equal(shouldBlock("write", undefined, false, tier1(true)), null);
	assert.equal(shouldBlock("bash", "rm -rf build/", false, tier1(true)), null);
});

test("tier1 with prerequisites NOT met stays blocked", () => {
	assert.equal(shouldBlock("edit", undefined, false, tier1(false)), BLOCK_REASON);
});

// ---- MU-3: tier 2 has no auto-path. THE load-bearing arm for D1. ----

test("tier2 never auto-disarms, however satisfied the prerequisites are", () => {
	assert.equal(shouldBlock("edit", undefined, false, tier2(true)), BLOCK_REASON);
	assert.equal(shouldBlock("write", undefined, false, tier2(true)), BLOCK_REASON);
});

test("tier2 unlocks only on an explicit ack", () => {
	assert.equal(shouldBlock("edit", undefined, true, tier2(true)), null);
});

// ---- Fail-closed defaults ----

test("an absent gate context is tier2 semantics, not permissive", () => {
	assert.equal(shouldBlock("edit", undefined, false), BLOCK_REASON);
	assert.equal(shouldBlock("edit", undefined, false, undefined), BLOCK_REASON);
});

test("an unrecognised tier fails closed to tier2", () => {
	const bogus = { tier: "tier0", prereqsMet: true } as unknown as GateContext;
	assert.equal(shouldBlock("edit", undefined, false, bogus), BLOCK_REASON);
});

test("read-only tools are never blocked in any tier", () => {
	for (const ctx of [tier1(false), tier2(false), undefined]) {
		assert.equal(shouldBlock("read", undefined, false, ctx), null);
		assert.equal(shouldBlock("grep", undefined, false, ctx), null);
	}
});

// ---- MU-5c: the generic floor must be satisfiable with no CLAUDE.md ----

test("generic floor is satisfiable in a repo with no CLAUDE.md", () => {
	const b = branch(
		{ name: "read", args: { path: "/workspace/task-1/README.md" } },
		{ name: "bash", args: { command: "ailang messages list --unread" } },
	);
	const { met, missing } = prerequisitesMet(b, "generic");
	assert.equal(met, true, `expected generic floor met, missing: ${missing.join("; ")}`);
});

test("generic floor still requires orientation AND an inbox check", () => {
	const noInbox = branch({ name: "read", args: { path: "/workspace/task-1/README.md" } });
	assert.equal(prerequisitesMet(noInbox, "generic").met, false);

	const noOrientation = branch({ name: "bash", args: { command: "ailang messages list" } });
	assert.equal(prerequisitesMet(noOrientation, "generic").met, false);
});

// ---- The AILANG set is the floor PLUS its own requirement ----

test("AILANG set additionally requires a CLAUDE.md read", () => {
	const genericOnly = branch(
		{ name: "read", args: { path: "/workspace/task-1/README.md" } },
		{ name: "bash", args: { command: "ailang messages list --unread" } },
	);
	assert.equal(prerequisitesMet(genericOnly, "generic").met, true);
	assert.equal(prerequisitesMet(genericOnly, "ailang").met, false,
		"the AILANG set must be strictly stronger than the floor");

	const full = branch(
		{ name: "read", args: { path: "/workspace/task-1/CLAUDE.md" } },
		{ name: "bash", args: { command: "ailang messages list --unread" } },
	);
	assert.equal(prerequisitesMet(full, "ailang").met, true);
});

test("anything the AILANG set accepts, the floor accepts too", () => {
	const full = branch(
		{ name: "read", args: { path: "/workspace/task-1/CLAUDE.md" } },
		{ name: "bash", args: { command: "ailang messages list --unread" } },
	);
	assert.equal(prerequisitesMet(full, "generic").met, true);
});

// ---- Selection is by TRUSTED metadata, never by a file in the clone ----

test("prereq set is chosen by the workspace identity the coordinator supplies", () => {
	assert.equal(prereqSetForWorkspace("sunholo-data/ailang"), "ailang");
	assert.equal(prereqSetForWorkspace("sunholo-data/ailang-parse"), "generic");
	assert.equal(prereqSetForWorkspace("some-org/anything"), "generic");
	assert.equal(prereqSetForWorkspace(""), "generic");
	assert.equal(prereqSetForWorkspace(undefined), "generic");
});
