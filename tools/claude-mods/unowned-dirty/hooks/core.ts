// Copied verbatim from .pi/extensions/unowned-dirty.ts — the pure functions
// only. scripts/check_claude_mods_drift.sh fails when this drifts from pi.
// Edit pi first, then copy.

export interface DirtyFile {
	status: string; // porcelain XY codes, e.g. " M", "M ", "??"
	path: string;
}

/** Pure: parse `git status --porcelain` lines into (status, path). */
export function parsePorcelain(out: string): DirtyFile[] {
	return out
		.split("\n")
		.filter((l) => l.trim() !== "")
		.map((l) => ({ status: l.slice(0, 2), path: l.slice(3).trim().replace(/^"|"$/g, "") }));
}

/** Pure: files dirty in the tree that this session did not write. */
export function unownedDirty(all: DirtyFile[], ownFiles: Set<string>): string[] {
	return all.map((f) => f.path).filter((p) => !ownFiles.has(p));
}

/** Pure: does this bash command perform a sweeping git operation? */
export function isSweepingGitOp(command: string | undefined): boolean {
	if (!command) return false;
	return /\bgit (add|stash|checkout|restore|reset)\b/.test(command);
}
