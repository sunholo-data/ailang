import { test } from "node:test";
import assert from "node:assert/strict";
import { parsePorcelain, unownedDirty, isSweepingGitOp } from "./unowned-dirty.ts";

test("parsePorcelain: statuses and paths", () => {
	const files = parsePorcelain(" M a.go\nM  b.txt\n?? new.txt\n");
	assert.deepEqual(files.map(f => [f.status, f.path]), [[" M", "a.go"], ["M ", "b.txt"], ["??", "new.txt"]]);
	assert.equal(files.length, 3);
});

test("unownedDirty: excludes session-authored files", () => {
	const all = parsePorcelain(" M mine.ts\n M theirs.go\n?? theirs2.go\n");
	const unowned = unownedDirty(all, new Set(["mine.ts"]));
	assert.deepEqual(unowned, ["theirs.go", "theirs2.go"]);
});

test("isSweepingGitOp: add/stash/checkout/restore/reset; not status/log", () => {
	assert.equal(isSweepingGitOp("git add ."), true);
	assert.equal(isSweepingGitOp("git stash"), true);
	assert.equal(isSweepingGitOp("git checkout dev"), true);
	assert.equal(isSweepingGitOp("git status"), false);
	assert.equal(isSweepingGitOp("git log"), false);
	assert.equal(isSweepingGitOp(undefined), false);
});
