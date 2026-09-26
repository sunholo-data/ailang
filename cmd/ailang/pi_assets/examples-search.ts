/**
 * examples-search — M-AGENT-AILANG-ONLY-EXECUTION follow-up (2026-09-16)
 *
 * `examples_search({query, limit?})`: find a WORKING example under examples/
 * whose text contains the query. The `ailang_only` lane has no shell, so the
 * model could only `read` an example whose path it already knew; the first
 * Jobs batch (six shell-less tasks) burned turns guessing at constructs that
 * examples/runnable/*.ail already demonstrate. Case-insensitive substring
 * over the file text, best-first: runnable/ before the rest, then by path.
 *
 * The examples dir is resolved by walking UP from cwd to the first directory
 * holding both `examples/` and `std/` (a repo root, wherever the workspace
 * is); a workspace outside any checkout gets `{count: 0, error}` — never a
 * throw. `examples/archive/**` (retired), `examples/bugs/**` and
 * `examples/expected_fail/**` (deliberately broken) are excluded: a search
 * tool that hands the model a known-broken program as "an example" is worse
 * than none. Matches carry an ABSOLUTE path so `read` works from any cwd.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";

export const DEFAULT_LIMIT = 8;
export const MAX_LIMIT = 50;
export const SNIPPET_MAX = 200;

/** Top-level examples/ subdirectories that are NOT teaching material. */
export const EXCLUDED_DIRS: readonly string[] = ["archive", "bugs", "expected_fail"];

export interface ExampleFile {
	/** Path as the caller wants it reported (absolute from the tool). */
	path: string;
	text: string;
}

export interface ExampleMatch {
	path: string;
	first_match_line: number; // 1-based
	snippet: string; // the matching line, trimmed, ≤ SNIPPET_MAX chars
	effects: string; // "! {IO, FS}" from the main/export line, "" when absent
}

export interface SearchResult {
	count: number;
	matches: ExampleMatch[];
	examples_dir?: string;
	error?: string;
}

/**
 * Pure given `exists`: walk up from `start` to the first dir that holds both
 * `examples/` and `std/`, or null. Bounded by the filesystem root.
 */
export function findExamplesDir(start: string, exists: (p: string) => boolean = existsSync): string | null {
	let dir = resolve(start);
	for (;;) {
		if (exists(join(dir, "examples")) && exists(join(dir, "std"))) return join(dir, "examples");
		const parent = dirname(dir);
		if (parent === dir) return null;
		dir = parent;
	}
}

/** Pure: is this path (relative to examples/, "/"-separated) inside an excluded top-level dir? */
export function isExcluded(relPath: string): boolean {
	const top = relPath.split("/")[0];
	return EXCLUDED_DIRS.includes(top);
}

/** "/"-separated path relative to `root` — the same string on every OS. */
export function relPosix(root: string, abs: string): string {
	return relative(root, abs).split(sep).join("/");
}

/**
 * Every *.ail under examplesDir (absolute paths, sorted), skipping EXCLUDED_DIRS.
 * Directories that vanish mid-walk are skipped, not fatal.
 */
export function listExampleFiles(examplesDir: string): string[] {
	const out: string[] = [];
	const walk = (dir: string) => {
		let names: string[];
		try {
			names = readdirSync(dir);
		} catch {
			return;
		}
		for (const name of names.sort()) {
			const abs = join(dir, name);
			if (isExcluded(relPosix(examplesDir, abs))) continue;
			let st;
			try {
				st = statSync(abs);
			} catch {
				continue;
			}
			if (st.isDirectory()) walk(abs);
			else if (st.isFile() && name.endsWith(".ail")) out.push(abs);
		}
	};
	walk(examplesDir);
	return out;
}

/** Pure: the `! {…}` effect row of the main/first-export line, "" when absent. */
export function effectsRow(text: string): string {
	const lines = text.split("\n");
	const pick = lines.find((l) => /^\s*export\s+(pure\s+)?func\s+main\b/.test(l)) ?? lines.find((l) => /^\s*export\s+(pure\s+)?func\b/.test(l));
	if (!pick) return "";
	const m = /!\s*\{([^}]*)\}/.exec(pick);
	return m ? `! {${m[1].trim()}}` : "";
}

/** Pure: rank runnable/ examples first (they are gated and pinned), then by path. */
export function rankKey(path: string): string {
	const p = path.split(sep).join("/");
	return (p.includes("/examples/runnable/") || p.startsWith("examples/runnable/") ? "0:" : "1:") + p;
}

/** Pure: clamp a requested limit into [1, MAX_LIMIT], defaulting when absent/invalid. */
export function clampLimit(limit: unknown): number {
	const n = typeof limit === "number" && Number.isFinite(limit) ? Math.floor(limit) : DEFAULT_LIMIT;
	return Math.min(MAX_LIMIT, Math.max(1, n));
}

/**
 * Pure: files whose text contains `query` (case-insensitive), capped at
 * `limit`. `count` is the number RETURNED (what the model can act on), and
 * the caller's `limit` is the only cap — an empty query matches nothing
 * rather than everything, so a model cannot page the whole corpus by accident.
 */
export function searchExamples(files: ExampleFile[], query: string, limit: number = DEFAULT_LIMIT): ExampleMatch[] {
	const q = (query ?? "").trim().toLowerCase();
	if (!q) return [];
	const cap = clampLimit(limit);
	const matches: ExampleMatch[] = [];
	const ordered = [...files].sort((a, b) => (rankKey(a.path) < rankKey(b.path) ? -1 : rankKey(a.path) > rankKey(b.path) ? 1 : 0));
	for (const f of ordered) {
		if (matches.length >= cap) break;
		if (!f.text.toLowerCase().includes(q)) continue;
		const lines = f.text.split("\n");
		const idx = lines.findIndex((l) => l.toLowerCase().includes(q));
		const line = idx >= 0 ? lines[idx].trim() : "";
		matches.push({
			path: f.path,
			first_match_line: idx + 1,
			snippet: line.length > SNIPPET_MAX ? `${line.slice(0, SNIPPET_MAX - 1)}…` : line,
			effects: effectsRow(f.text),
		});
	}
	return matches;
}

/** The whole tool, minus pi: resolve, list, read, search. Never throws. */
export function runSearch(cwd: string, query: string, limit: unknown): SearchResult {
	const examplesDir = findExamplesDir(cwd);
	if (!examplesDir) return { count: 0, matches: [], error: `no examples dir found from ${cwd}` };
	const files: ExampleFile[] = [];
	for (const abs of listExampleFiles(examplesDir)) {
		try {
			files.push({ path: abs, text: readFileSync(abs, "utf8") });
		} catch {
			/* a file that vanished between list and read is not a match */
		}
	}
	const matches = searchExamples(files, query, clampLimit(limit));
	return { count: matches.length, matches, examples_dir: examplesDir };
}

export default async function (pi: ExtensionAPI) {
	const { Type } = await import("typebox");

	pi.registerTool({
		name: "examples_search",
		label: "Search AILANG examples",
		description:
			"Find a WORKING example program under examples/ whose text contains `query` (case-insensitive substring, " +
			"e.g. 'match' + 'Result', 'listDir', 'foldlE'). Returns {count, matches:[{path, first_match_line, snippet, effects}]}; " +
			"then `read` the path. Use this before writing a construct you are unsure of — examples/runnable/*.ail are gated and pinned.",
		parameters: Type.Object({
			query: Type.String({ description: "Substring to look for in the example text" }),
			limit: Type.Optional(Type.Number({ description: `Max matches (default ${DEFAULT_LIMIT}, max ${MAX_LIMIT})` })),
		}),
		async execute(_id, params, _signal, _onUpdate, ctx) {
			void ctx;
			let result: SearchResult;
			try {
				result = runSearch(process.cwd(), params.query, params.limit);
			} catch (e) {
				result = { count: 0, matches: [], error: `examples_search failed: ${String(e)}` };
			}
			return {
				content: [{ type: "text", text: JSON.stringify(result) }],
				details: result,
			};
		},
	});
}
