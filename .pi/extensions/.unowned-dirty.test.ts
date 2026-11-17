import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import unownedDirtyExtension, { parsePorcelain, unownedDirty, isSweepingGitOp, relativeTo, realOrSelf } from "./unowned-dirty.ts";

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

test("wiring: absolute own paths, relativized against the toplevel, exclude the session's own edits", () => {
	const all = parsePorcelain(" M mine.ts\n M sub/also-mine.go\n M theirs.go\n");
	const own = new Set(["/repo/mine.ts", "/repo/sub/also-mine.go", "/elsewhere/x.ts"]);
	assert.deepEqual(unownedDirty(all, own), ["mine.ts", "sub/also-mine.go", "theirs.go"]); // the HEAD defect
	assert.deepEqual(unownedDirty(all, relativeTo("/repo", own)), ["theirs.go"]);
	assert.deepEqual([...relativeTo("/repo/", own)].sort(), ["mine.ts", "sub/also-mine.go"]);
});

test("wiring: an own path spelled through a symlinked cwd still matches the physical toplevel", () => {
	const base = realOrSelf(mkdtempSync(join(tmpdir(), "ud-")));
	const top = join(base, "repo");
	mkdirSync(top);
	writeFileSync(join(top, "mine.ts"), "");
	symlinkSync(top, join(base, "link"));
	const own = [join(base, "link", "mine.ts")]; // as ctx.cwd under the symlink would spell it
	assert.deepEqual([...relativeTo(top, own)], []); // without realpath: dropped, reported unowned
	assert.deepEqual([...relativeTo(realOrSelf(top), own.map(realOrSelf))], ["mine.ts"]);
});

test("handler: a relative edit/write path is not reported unowned on a sweeping git op", async () => {
	const top = realOrSelf(mkdtempSync(join(tmpdir(), "ud-h-")));
	const handlers: Record<string, Function> = {};
	const notes: string[] = [];
	const fakePi = {
		on: (name: string, fn: Function) => { handlers[name] = fn; },
		exec: async (_cmd: string, args: string[]) =>
			args[0] === "rev-parse"
				? { code: 0, stdout: `${top}\n` } // git prints a trailing newline
				: { code: 0, stdout: " M mine.ts\n M theirs.go\n" },
	};
	const ctx = { cwd: top, ui: { notify: async (msg: string) => { notes.push(msg); } } };
	unownedDirtyExtension(fakePi as never);
	await handlers.tool_call({ toolName: "edit", input: { path: "mine.ts" } }, ctx);
	await handlers.tool_call({ toolName: "bash", input: { command: "git add -A" } }, ctx);
	assert.equal(notes.length, 1);
	assert.match(notes[0], /may sweep 1 dirty file\(s\)[^:]*: theirs\.go\./);
});
