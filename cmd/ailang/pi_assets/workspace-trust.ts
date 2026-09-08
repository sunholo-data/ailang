/**
 * workspace-trust — M-DX-PI-HARNESS doctrine addition (2026-09-08)
 *
 * WHY: pi ≥0.84 gates project resources (`.agents/skills/`, `.pi/`) behind
 * project trust. Headless modes (`-p`, `--mode json`, `--mode rpc`) never
 * prompt, so any checkout without a saved `~/.pi/agent/trust.json` decision
 * (fresh cloud containers, /tmp worktrees, new mission worktree roots)
 * silently drops `.agents/skills/` — the same trap measured twice:
 * extensions 2026-08-31, the skills canary 2026-09-08 (mission-control.sh
 * evaluator note). Skills never got the Tier-2 global install treatment
 * extensions got, and fresh worktree paths make per-path trust.json entries
 * not scale.
 *
 * WHAT: handles pi's `project_trust` event. Trusts the project when the
 * cwd's git ORIGIN remote matches a configured pattern. The decision is
 * per-process (`remember: false`): trust.json stays owned by humans.
 * Never returns "no" — non-matching dirs fall through to pi's normal flow
 * (saved decisions, defaultProjectTrust, interactive prompt).
 *
 * PER-REPO CONFIGURATION (the executors run in many repos). A pattern is a
 * case-insensitive substring of the origin remote URL, so an `org/repo`
 * coordinate matches both `git@github.com:org/repo.git` and
 * `https://github.com/org/repo.git`. Sources, in precedence order:
 *
 * 1. PI_WORKSPACE_TRUST=0 — kill switch; the extension abstains entirely.
 * 2. `~/.pi/agent/workspace-trust.json` — `{"remotes": ["org/repo", ...]}`.
 *    Machine-owned, human-maintained; NEVER read from the project's own
 *    `.pi/` (a repo must not supply its own trust config — that would be
 *    self-approving trust, exactly what the gate exists to prevent). When
 *    present and valid it REPLACES the built-in defaults; `{"remotes": []}`
 *    means "no default patterns on this machine". Invalid or unreadable →
 *    loud stderr warning + ABSTAIN (fail-closed: a broken config never
 *    silently reverts to defaults).
 * 3. PI_WORKSPACE_TRUST_REMOTES="a/b,c/d" — always ADDITIVE. The coordinator
 *    also sets this per task (local: the agent's `repo:` coordinate;
 *    cloud execute-job: the job's clone URL), so every dispatched repo is
 *    trusted in its own checkout with zero per-repo setup. That injected
 *    pattern is dispatcher config — machine-owned, never repo content.
 * 4. Built-in defaults: ["sunholo", "arniwesth/motoko_agent"].
 *
 * SAFETY (stated honestly): this pre-approves project resources for any
 * repo whose origin remote matches. Trust gates project-local
 * settings/extension/skill injection, not code execution — repo files are
 * read regardless of trust — so a hostile repo naming itself to match a
 * pattern would be auto-trusted, which is an insider-repo threat, outside
 * pi's stated threat model for trust. Escape hatches: PI_WORKSPACE_TRUST=0
 * disables this extension; --approve/--no-approve still override
 * everything (pi resolves trustOverride before the event fires).
 *
 * Subprocess Contract: one `git remote get-url origin` per startup, 10s
 * timeout; any failure → "undecided" (fail toward pi's default flow, never
 * silently trust).
 */
import type {
	ExtensionAPI,
	ProjectTrustContext,
	ProjectTrustEvent,
	ProjectTrustEventResult,
} from "@earendil-works/pi-coding-agent";
import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

/** PI_WORKSPACE_TRUST=0 → this extension abstains entirely. */
export const KILL_SWITCH_ENV = "PI_WORKSPACE_TRUST";
/** PI_WORKSPACE_TRUST_REMOTES="corp/repo,org/repo2" → additive origin patterns (substring, case-insensitive). */
export const EXTRA_REMOTES_ENV = "PI_WORKSPACE_TRUST_REMOTES";
/** Built-in default family patterns: every sunholo org + the motoko mission repo (MOTOKO.md: origin = arniwesth/motoko_agent). */
export const DEFAULT_REMOTE_PATTERNS = ["sunholo", "arniwesth/motoko_agent"];
/** Machine config file path (user-owned; replaces defaults when valid). */
export function configPath(home: string): string {
	return join(home, ".pi", "agent", "workspace-trust.json");
}

export interface TrustConfig {
	/** null = file absent → defaults apply; array (possibly empty) = replaces defaults. */
	patterns: string[] | null;
	/** set when the file exists but is invalid/unreadable → caller must abstain. */
	warning: string | null;
}

/** Parse the machine config. ENOENT → { patterns: null, warning: null }; anything else broken → warning. */
export function loadConfig(readFile: (p: string) => string, home: string): TrustConfig {
	const path = configPath(home);
	let raw: string;
	try {
		raw = readFile(path);
	} catch (e) {
		const err = e as { code?: string; message?: string };
		if (err.code === "ENOENT") return { patterns: null, warning: null };
		return { patterns: null, warning: `cannot read ${path}: ${err.message ?? "unknown error"}` };
	}
	try {
		const parsed = JSON.parse(raw) as { remotes?: unknown };
		if (!Array.isArray(parsed.remotes)) {
			return { patterns: null, warning: `${path}: "remotes" must be an array of strings` };
		}
		const patterns = parsed.remotes.filter((p): p is string => typeof p === "string" && p.trim().length > 0);
		if (patterns.length !== parsed.remotes.length) {
			return { patterns: null, warning: `${path}: "remotes" contains non-string/empty entries` };
		}
		return { patterns, warning: null };
	} catch (e) {
		const err = e as { message?: string };
		return { patterns: null, warning: `${path}: invalid JSON — ${err.message ?? "unknown error"}` };
	}
}

/**
 * Pure decision. remote=null/"" (unresolvable) → undecided.
 * Invariants: never returns "no"; never sets remember.
 */
export function decide(input: {
	remote: string | null;
	killSwitch?: string;
	extraRemotes?: string;
	configPatterns?: string[] | null;
	configWarning?: string | null;
}): ProjectTrustEventResult {
	if ((input.killSwitch ?? "").trim() === "0") {
		return { trusted: "undecided" };
	}
	if (input.configWarning) {
		return { trusted: "undecided" }; // fail-closed: broken machine config never auto-trusts
	}
	const remote = (input.remote ?? "").trim().toLowerCase();
	if (!remote) {
		return { trusted: "undecided" };
	}
	const base = (input.configPatterns ?? DEFAULT_REMOTE_PATTERNS).map((p) => p.trim().toLowerCase());
	const extras = (input.extraRemotes ?? "")
		.split(",")
		.map((p) => p.trim().toLowerCase())
		.filter((p) => p.length > 0);
	if ([...base, ...extras].some((p) => remote.includes(p))) {
		return { trusted: "yes", remember: false };
	}
	return { trusted: "undecided" };
}

/** Resolve the cwd's git origin remote. Never throws — failures return null. */
export async function originRemote(
	cwd: string,
	exec: (cmd: string, args: string[], opts: { timeout: number }) => Promise<{ stdout?: string }>,
): Promise<string | null> {
	try {
		const res = await exec("git", ["-C", cwd, "remote", "get-url", "origin"], { timeout: 10_000 });
		return (res.stdout ?? "").trim() || null;
	} catch {
		return null;
	}
}

export default async function (pi: ExtensionAPI) {
	pi.on(
		"project_trust",
		async (event: ProjectTrustEvent, ctx: ProjectTrustContext): Promise<ProjectTrustEventResult> => {
			try {
				const config = loadConfig((p) => readFileSync(p, "utf8"), homedir());
				if (config.warning) {
					// Loud both where ops reads (lane logs / stderr) and where humans see (TUI).
					console.error(`[workspace-trust] ${config.warning} — abstaining; fix the config or set PI_WORKSPACE_TRUST=0`);
					if (ctx.hasUI) {
						await ctx.ui.notify(`workspace-trust: ${config.warning} (abstaining)`, "warning");
					}
				}
				const remote = await originRemote(event.cwd, pi.exec.bind(pi));
				return decide({
					remote,
					killSwitch: process.env[KILL_SWITCH_ENV],
					extraRemotes: process.env[EXTRA_REMOTES_ENV],
					configPatterns: config.patterns,
					configWarning: config.warning,
				});
			} catch {
				return { trusted: "undecided" }; // fail toward pi's default flow, never silently trust
			}
		},
	);
}