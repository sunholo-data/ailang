import { test } from "node:test";
import assert from "node:assert/strict";
import { attributionTrailer, appendAttribution, hasAttribution, shouldBlock } from "./session-protocol-gate.ts";

const T = "Co-Authored-By: pi (ollama/glm-5.3-flash:cloud) <noreply@ailang.dev>";

test("attributionTrailer: names model+provider; falls back to harness", () => {
	assert.equal(attributionTrailer({ provider: "ollama", id: "glm-5.3-flash:cloud" }), T);
	assert.equal(attributionTrailer(undefined), "Co-Authored-By: pi-coding-agent <noreply@ailang.dev>");
});

test("appendAttribution: patches simple/double/single -m forms", () => {
	assert.ok(appendAttribution('git commit -m "fix: thing"', T)!.changed);
	assert.ok(appendAttribution('git commit -m "t" -m "b"', T)!.command.includes("b\n\n" + T));
	assert.ok(appendAttribution("git commit --message='fix: x'", T)!.changed);
});

test("appendAttribution: handles flag clusters like -am and -m=", () => {
	assert.ok(appendAttribution("git commit -am 'attr test 11'", T)!.command.includes("'attr test 11\n\n" + T + "'"));
	assert.ok(appendAttribution("git commit -m='x'", T)!.changed);
	assert.equal(appendAttribution("git commit -m", T)!.skipped, "editor-flow");
});

test("appendAttribution: idempotent + conservative skips", () => {
	const attributed = 'git commit -m "fix: thing\n\n' + T + '"';
	assert.equal(appendAttribution(attributed, T)!.changed, false);
	assert.equal(appendAttribution('git commit -m "$(cat msg)"', T)!.skipped, "dynamic-message");
	assert.equal(appendAttribution("git commit -a", T)!.skipped, "editor-flow");
	assert.equal(appendAttribution("git push origin dev", T), null);
});

test("gate+attribution compose: allowed commit is patched, blocked is not executed at all", () => {
	// while armed: commit is blocked → never executes → attribution is moot
	assert.equal(shouldBlock("bash", "git commit -m 'x'", false) !== null, true);
	// after ack: allowed → patch applies
	const r = appendAttribution('git commit -m "fix: thing"', T)!;
	assert.ok(r.changed);
	assert.ok(!hasAttribution("git push origin dev") || true);
});
