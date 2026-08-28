import { test } from "node:test";
import assert from "node:assert/strict";
import { parseCheckOutput, filterBuiltins } from "./ailang-lsp-lite.ts";

test("parseCheckOutput: TC error with span + builtinHint passthrough", () => {
	const out = "→ Effect checking...\nError: type error in typo (decl 0): undefined variable: _fs_renam at typo.ail:6:3 — '_fs_renam' follows the builtin naming convention; run `ailang builtins list` for the real inventory\n";
	const d = parseCheckOutput(out);
	assert.equal(d.length, 1);
	assert.equal(d[0].severity, "error");
	assert.equal(d[0].file, "typo.ail");
	assert.equal(d[0].line, 6);
	assert.equal(d[0].col, 3);
	assert.equal(d[0].message, "type error in typo (decl 0): undefined variable: _fs_renam");
	assert.match(d[0].hint, /ailang builtins list/);
});

test("parseCheckOutput: IMP010 with code and warning lines", () => {
	const out = "Error: IMP010: symbol 'x' not exported by 'std/fs' — did you mean 'y'?\nWarning: stdlib version mismatch: expected dev, found v0.34.0 at /repo/std\n";
	const d = parseCheckOutput(out);
	assert.equal(d.length, 2);
	assert.equal(d[0].code, "IMP010");
	assert.equal(d[0].severity, "error");
	assert.match(d[0].hint, /did you mean/);
	assert.equal(d[1].severity, "warning");
});

test("parseCheckOutput: clean output → no diagnostics", () => {
	assert.deepEqual(parseCheckOutput("✓ No errors found!"), []);
});

test("filterBuiltins: query, module, cap; empty query returns all", () => {
	const inv = [
		{ name: "_list_nth", module: "std/list", signature: "_list_nth: (list[a], int) -> a" },
		{ name: "_fs_rename", module: "std/fs", signature: "_fs_rename: (string, string) -> () ! {FS}" },
		{ name: "_str_len", module: "std/string", signature: "_str_len: string -> int" },
	];
	assert.equal(filterBuiltins(inv, "nth", undefined).length, 1);
	assert.equal(filterBuiltins(inv, undefined, "std/fs").length, 1);
	assert.equal(filterBuiltins(inv, "nonexistent-xyz", undefined).length, 0);
	assert.equal(filterBuiltins(inv, undefined, undefined).length, 3);
	// cap applies only to queried searches
	assert.equal(filterBuiltins(inv, "s", undefined).length <= 10, true);
});
