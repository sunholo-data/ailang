import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve, sep } from "node:path";
import {
	DEFAULT_LIMIT,
	MAX_LIMIT,
	SNIPPET_MAX,
	clampLimit,
	effectsRow,
	findExamplesDir,
	isExcluded,
	listExampleFiles,
	rankKey,
	relPosix,
	runSearch,
	searchExamples,
} from "./examples-search.ts";

const MAIN_FS = 'module examples/runnable/dir\nimport std/fs (listDir)\nimport std/io (println)\n\nexport func main() -> () ! {IO, FS} {\n  let xs = listDir(".")\n  println("ok")\n}\n';
const PURE = "module examples/pure\n\nexport pure func double(x: int) -> int {\n  x * 2\n}\n";
const MATCH_RESULT = 'module examples/runnable/res\nimport std/io (println)\n\nexport func main() -> () ! {IO} {\n  match Ok(1) {\n    Ok(v) => println("v"),\n    Err(e) => println(e)\n  }\n}\n';

test("findExamplesDir: walks up to the dir holding both examples/ and std/", () => {
	// resolve() so the expected strings carry the same drive prefix the walker sees on Windows
	const repo = resolve(sep, "repo");
	const have = new Set([join(repo, "examples"), join(repo, "std")]);
	const exists = (p: string) => have.has(p);
	assert.equal(findExamplesDir(join(repo, "examples", "runnable", "deep"), exists), join(repo, "examples"));
	// examples/ without std/ is not a checkout
	const onlyExamples = (p: string) => p.endsWith(`${sep}examples`);
	assert.equal(findExamplesDir(resolve(sep, "elsewhere", "x"), onlyExamples), null);
	assert.equal(findExamplesDir(resolve(sep, "nowhere"), () => false), null);
});

test("isExcluded / relPosix: archive, bugs and expected_fail are skipped; separators normalise", () => {
	assert.equal(isExcluded("archive/broken/x.ail"), true);
	assert.equal(isExcluded("bugs/x.ail"), true);
	assert.equal(isExcluded("expected_fail/x.ail"), true);
	assert.equal(isExcluded("runnable/x.ail"), false);
	assert.equal(isExcluded("archived_not_really/x.ail"), false);
	const root = resolve(sep, "r", "examples");
	assert.equal(relPosix(root, join(root, "runnable", "a.ail")), "runnable/a.ail");
});

test("effectsRow: main's row wins, first export otherwise, empty for pure", () => {
	assert.equal(effectsRow(MAIN_FS), "! {IO, FS}");
	assert.equal(effectsRow(PURE), "");
	assert.equal(effectsRow("export func helper() -> () ! { IO } {\n}\nexport func main() -> () ! {FS} {\n}\n"), "! {FS}");
	assert.equal(effectsRow("export func helper() -> () ! { IO } {\n}\n"), "! {IO}");
	assert.equal(effectsRow("-- no exports at all\n"), "");
});

test("clampLimit: default, floor, bounds", () => {
	assert.equal(clampLimit(undefined), DEFAULT_LIMIT);
	assert.equal(clampLimit("8"), DEFAULT_LIMIT);
	assert.equal(clampLimit(NaN), DEFAULT_LIMIT);
	assert.equal(clampLimit(0), 1);
	assert.equal(clampLimit(-3), 1);
	assert.equal(clampLimit(2.9), 2);
	assert.equal(clampLimit(10_000), MAX_LIMIT);
});

test("searchExamples: case-insensitive substring, first line, snippet, effects, runnable first", () => {
	const files = [
		{ path: "/r/examples/zzz_toplevel.ail", text: "-- uses LISTDIR here\nexport func main() -> () ! {FS} {\n}\n" },
		{ path: "/r/examples/runnable/dir.ail", text: MAIN_FS },
		{ path: "/r/examples/pure.ail", text: PURE },
	];
	const m = searchExamples(files, "listdir", 8);
	assert.equal(m.length, 2);
	assert.equal(m[0].path, "/r/examples/runnable/dir.ail"); // runnable/ ranks first
	assert.equal(m[0].first_match_line, 2);
	assert.equal(m[0].snippet, "import std/fs (listDir)");
	assert.equal(m[0].effects, "! {IO, FS}");
	assert.equal(m[1].path, "/r/examples/zzz_toplevel.ail");
	assert.equal(m[1].first_match_line, 1);
	assert.equal(m[1].effects, "! {FS}");
	// no match, empty query, whitespace query → nothing (never the whole corpus)
	assert.deepEqual(searchExamples(files, "nonexistent-xyz"), []);
	assert.deepEqual(searchExamples(files, ""), []);
	assert.deepEqual(searchExamples(files, "   "), []);
});

test("searchExamples: limit caps and the snippet is trimmed to SNIPPET_MAX", () => {
	const long = `${"x".repeat(SNIPPET_MAX + 50)} needle`;
	const files = Array.from({ length: 12 }, (_, i) => ({ path: `/r/examples/e${String(i).padStart(2, "0")}.ail`, text: `   ${long}\n` }));
	const capped = searchExamples(files, "needle", 5);
	assert.equal(capped.length, 5);
	assert.equal(capped[0].snippet.length, SNIPPET_MAX);
	assert.ok(capped[0].snippet.endsWith("…"));
	assert.equal(searchExamples(files, "needle").length, DEFAULT_LIMIT);
});

test("rankKey: runnable/ sorts before top-level regardless of name", () => {
	assert.ok(rankKey("/r/examples/runnable/zeta.ail") < rankKey("/r/examples/alpha.ail"));
	assert.ok(rankKey("examples/runnable/b.ail") < rankKey("examples/runnable/c.ail"));
});

test("listExampleFiles + runSearch over a real fixture tree; missing checkout is an error, not a throw", () => {
	const root = mkdtempSync(join(tmpdir(), "examples-search-"));
	try {
		const ex = join(root, "examples");
		mkdirSync(join(ex, "runnable"), { recursive: true });
		mkdirSync(join(ex, "archive", "broken"), { recursive: true });
		mkdirSync(join(ex, "expected_fail"), { recursive: true });
		mkdirSync(join(root, "std"), { recursive: true });
		writeFileSync(join(ex, "runnable", "res.ail"), MATCH_RESULT);
		writeFileSync(join(ex, "runnable", "dir.ail"), MAIN_FS);
		writeFileSync(join(ex, "pure.ail"), PURE);
		writeFileSync(join(ex, "notes.md"), "match Ok in markdown must not count");
		writeFileSync(join(ex, "archive", "broken", "old.ail"), "match Ok(1) { _ => 1 }");
		writeFileSync(join(ex, "expected_fail", "bad.ail"), "match Ok(1) { _ => 1 }");

		const listed = listExampleFiles(ex).map((p) => relPosix(ex, p));
		assert.deepEqual(listed, ["pure.ail", "runnable/dir.ail", "runnable/res.ail"]);

		// from a nested cwd inside the checkout
		const r = runSearch(join(ex, "runnable"), "match Ok", 8);
		assert.equal(r.error, undefined);
		assert.equal(r.examples_dir, ex);
		assert.equal(r.count, 1);
		assert.equal(r.matches[0].path, join(ex, "runnable", "res.ail")); // absolute: `read` works from any cwd
		assert.equal(r.matches[0].first_match_line, 5);
		assert.equal(r.matches[0].effects, "! {IO}");

		// from a workspace with no checkout above it
		const nowhere = mkdtempSync(join(tmpdir(), "examples-search-nowhere-"));
		try {
			const miss = runSearch(nowhere, "match", 8);
			assert.equal(miss.count, 0);
			assert.deepEqual(miss.matches, []);
			assert.match(miss.error ?? "", /^no examples dir found from /);
		} finally {
			rmSync(nowhere, { recursive: true, force: true });
		}
	} finally {
		rmSync(root, { recursive: true, force: true });
	}
});
