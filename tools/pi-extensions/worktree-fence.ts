/**
 * Worktree Fence — mission-executor containment for pi.
 *
 * WHY THIS EXISTS. The mission runs pi as its sprint executor with full user
 * permissions from a git worktree ($WT). Containment today is the directive's
 * scope fence plus the controller's post-hoc `git -C <main-checkout> status
 * --short` review — i.e. prose plus an audit, with nothing enforcing it. Iteration
 * 168 showed the cost: a killed executor kept running and overwrote a verified tree
 * mid-evaluation.
 *
 * The `sandbox/` example extension fences the BASH tool at the OS level
 * (@anthropic-ai/sandbox-runtime → Seatbelt on macOS), but it does NOT touch
 * `write` or `edit` — those are Node fs calls inside the pi process, which is not
 * sandboxed. This extension closes exactly that gap. Use BOTH: sandbox for bash,
 * this for write/edit.
 *
 * DESIGN: allow-list, not deny-list. The `protected-paths` example blocks a few
 * known-bad substrings; that is the wrong shape here — we do not know every path
 * worth protecting, but we do know the single path that is legitimate. Anything
 * that does not resolve inside the fence root is refused.
 *
 * Root = $PI_FENCE_ROOT, else cwd (the mission recipe already does `cd "$WT"`).
 *
 * Headless-safe by construction: it only ever BLOCKS or allows, and never prompts.
 * `ctx.hasUI` guards the notification, because the mission runs pi non-interactively
 * (`--mode json -p ... < /dev/null`) and a prompt there would wedge the loop.
 */

import { existsSync, realpathSync } from "node:fs";
import { dirname, isAbsolute, resolve, sep } from "node:path";

/**
 * Resolve `p` to a real absolute path, following symlinks.
 *
 * The target usually does NOT exist yet (that is the point of `write`), so
 * realpath the nearest existing ancestor and re-append the remainder. Without
 * this, a symlinked parent directory escapes the fence: on macOS `/tmp` is itself
 * a symlink to `/private/tmp`, so a naive string compare against a `/tmp/...` root
 * rejects every legitimate write.
 */
function realResolve(p: string, base: string): string {
	const abs = isAbsolute(p) ? p : resolve(base, p);
	let existing = abs;
	const trailing: string[] = [];

	while (!existsSync(existing)) {
		const parent = dirname(existing);
		if (parent === existing) return abs; // hit the filesystem root; nothing to resolve
		trailing.unshift(existing.slice(parent.length + 1));
		existing = parent;
	}

	try {
		return resolve(realpathSync(existing), ...trailing);
	} catch {
		return abs;
	}
}

/** True when `child` is `root` itself or lives underneath it. */
function isWithin(root: string, child: string): boolean {
	// The `+ sep` matters: without it "/a/b" would admit "/a/bc".
	return child === root || child.startsWith(root.endsWith(sep) ? root : root + sep);
}

export default function (pi: any) {
	const rawRoot = process.env.PI_FENCE_ROOT || process.cwd();
	const root = realResolve(rawRoot, process.cwd());

	// Tools that can create or mutate files. `bash` is deliberately absent: the
	// sandbox extension fences it at the OS level, and duplicating that here would
	// mean parsing shell, which is a losing game.
	const WRITE_TOOLS = new Set(["write", "edit"]);

	// Where a tool call carries its target path. Checked in order; the first key
	// present wins. Kept explicit so a pi version that renames the field fails
	// LOUDLY below rather than silently letting writes through.
	const PATH_KEYS = ["path", "file_path", "filePath"];

	pi.on("tool_call", async (event: any, ctx: any) => {
		if (!WRITE_TOOLS.has(event.toolName)) return undefined;

		const input = event.input ?? {};
		const key = PATH_KEYS.find((k) => typeof input[k] === "string");

		// FAIL CLOSED. A write tool whose path we cannot find is a write we cannot
		// judge — refuse it and say so, rather than assume it is fine.
		if (!key) {
			const reason =
				`worktree-fence: could not find a path argument on "${event.toolName}" ` +
				`(looked for ${PATH_KEYS.join(", ")}; got ${Object.keys(input).join(", ") || "no keys"}). ` +
				`Refusing: an unrecognised write shape must not bypass the fence.`;
			if (ctx?.hasUI) ctx.ui.notify(reason, "error");
			return { block: true, reason };
		}

		// Resolve relative paths against the PROCESS CWD, which is what pi itself
		// does — not against the fence root. Getting this wrong is silently fatal:
		// with base=root, "dbg.txt" resolves to <root>/dbg.txt and is allowed, while
		// pi writes it to <cwd>/dbg.txt, outside the fence. Measured 2026-08-10 — a
		// live run escaped while the unit tests were green, because those tests had
		// root == cwd and could not distinguish the two bases.
		const target = realResolve(input[key] as string, process.cwd());

		if (!isWithin(root, target)) {
			const reason =
				`worktree-fence: "${input[key]}" resolves to ${target}, which is outside ` +
				`the fence root ${root}. Writes are confined to the worktree. If this file ` +
				`genuinely needs changing, report it in FINDINGS.md — the controller decides.`;
			if (ctx?.hasUI) ctx.ui.notify(reason, "warning");
			return { block: true, reason };
        }

		return undefined;
	});
}
