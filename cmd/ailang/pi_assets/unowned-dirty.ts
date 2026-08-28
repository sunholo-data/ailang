/**
 * unowned-dirty — M-DX-PI-HARNESS A3 (round-2 redesign)
 * A visibility warning, NOT an authority check: on bash `git add|stash|checkout`,
 * consult git itself (`git status --porcelain`, 10s timeout) for what is dirty, and
 * warn when the operation may sweep files this session did not write. Never blocks;
 * output names itself a heuristic. Shared checkouts have multiple agents (observed).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

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

export default function (pi: ExtensionAPI) {
	const ownFiles = new Set<string>();

	// Track files this session wrote via edit/write (the only file-mutating
	// built-ins; bash writes are unknowable — the warning says so).
	pi.on("tool_call", async (event, ctx) => {
		if (event.toolName === "edit" || event.toolName === "write") {
			const p = (event.input as { path?: string } | undefined)?.path;
			if (p) ownFiles.add(p.startsWith("/") ? p : `${ctx.cwd}/${p}`);
			return; // never blocks anything
		}
		if (event.toolName !== "bash") return;
		const command = (event.input as { command?: string } | undefined)?.command ?? "";
		if (!isSweepingGitOp(command)) return;

		// Authority is git, not command parsing: what is ACTUALLY dirty right now?
		const res = await pi.exec("git", ["status", "--porcelain"], { timeout: 10_000 });
		const all = parsePorcelain(res.stdout ?? "");
		const unowned = unownedDirty(all, ownFiles).filter(
			(f) => !/^(internal\/ai\/)?ollama/.test(f), // placeholder-free filter; kept simple
		);
		if (unowned.length > 0) {
			await ctx.ui.notify(
				`[unowned-dirty heuristic] '${command.trim().slice(0, 60)}' may sweep ${unowned.length} dirty file(s) this session did not write: ` +
					unowned.slice(0, 5).join(", ") + (unowned.length > 5 ? `, +${unowned.length - 5} more` : "") +
					". Coordinate before stashing/committing others' in-flight work.",
				"warning",
			);
		}
	});

	// Reconstruct session-authored files on (re)load from the session branch.
	pi.on("session_start", async (_event, ctx) => {
		try {
			for (const entry of ctx.sessionManager.getBranch()) {
				const msg = (entry as { message?: { role?: string; content?: unknown } }).message;
				if (msg?.role !== "assistant" || !Array.isArray(msg.content)) continue;
				for (const part of msg.content as Array<{
					type?: string;
					name?: string;
					arguments?: { path?: string };
				}>) {
					if (part?.type === "toolCall" && (part.name === "edit" || part.name === "write")) {
						const p = part.arguments?.path;
						if (p) ownFiles.add(p.startsWith("/") ? p : `${ctx.cwd}/${p}`);
					}
				}
			}
		} catch {
			// fail-open for a WARN-only guard: fewer warnings is safe; blocking is not possible here
		}
	});
}