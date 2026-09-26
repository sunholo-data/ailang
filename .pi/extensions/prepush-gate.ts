/**
 * prepush-gate — M-DX-PI-HARNESS M6: run the repo's CI gates locally BEFORE
 * `git push`, blocking the push with the failing output when they fail.
 *
 * Rationale (observed 2026-08-28): CI caught three issues the author's
 * targeted checks missed — the 800-line file ceiling, golangci-lint `unused`
 * on dead code, and the cihygiene verify-target wiring gate. Local gates were
 * never run because knowledge of them is tribal. This extension makes the
 * pre-push gate mechanical: same commands CI runs, same failure surface.
 *
 * Fail-CLOSED (unlike unowned-dirty's warn): a push that would land a red CI
 * run is blocked until the gates pass. Escape hatch: AILANG_SKIP_PREPUSH=1 in
 * the environment pi was LAUNCHED with (operator-level). An agent writing
 * `AILANG_SKIP_PREPUSH=1 git push` or exporting it in its bash tool does NOT
 * bypass the gate — a hatch the gated agent can open itself is no gate.
 *
 * Each make gate runs only when the repo's Makefile defines that target. A Go
 * repo without ailang's Makefile (ailang-world: Go under cmd/, no Makefile)
 * was blocked on every push by `make lint` → "No rule to make target" — an
 * unpassable gate, measured 2026-09-23 (world iteration 180, row 39 unlandable).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export interface GateStep {
	name: string;
	cmd: string;
	args: string[];
	timeoutMs: number;
}

/** Pure: the prepush gate chain for a Go repo (ailang's gates, fast subset). */
export function gateSteps(): { name: string; cmd: string; args: string[]; timeoutMs: number }[] {
	return [
		{ name: "gofmt", cmd: "gofmt", args: ["-l", "cmd", "internal"], timeoutMs: 15_000 },
		{ name: "lint", cmd: "make", args: ["lint"], timeoutMs: 180_000 },
		{ name: "file-sizes", cmd: "make", args: ["check-file-sizes"], timeoutMs: 15_000 },
	];
}

/**
 * Does this working tree contain the Go layout these gates assume?
 *
 * Deliberately checks for TRACKED .go files under the two directories the gate
 * actually inspects, rather than for a go.mod: a repo can vendor a go.mod and
 * still have nothing for `gofmt -l cmd/ internal/` to read, which is the shape
 * that produced an unpassable gate.
 */
export async function isGoRepo(pi: ExtensionAPI): Promise<boolean> {
	return (await trackedGoRoots(pi)).length > 0;
}

/** The Go source roots (of cmd/, internal/) holding TRACKED .go files. */
export async function trackedGoRoots(pi: ExtensionAPI): Promise<string[]> {
	const tracked = await pi.exec("git", ["ls-files", "cmd/*.go", "internal/*.go", "cmd/**/*.go", "internal/**/*.go"], {
		timeout: 15_000,
	});
	return goRoots(tracked.stdout ?? "");
}

/** Pure: the Go source roots (of cmd/, internal/) that `git ls-files` output touches. */
export function goRoots(lsFilesStdout: string): string[] {
	const files = lsFilesStdout.split("\n").filter((l) => l.trim() !== "");
	return ["cmd", "internal"].filter((root) => files.some((f) => f.startsWith(`${root}/`)));
}

/**
 * Pure: given `make -n <target>`'s result, does the repo define the target?
 * Only the two "it isn't there" shapes count as absent; any other failure
 * means the target exists and is broken, so the gate still runs (and blocks).
 */
export function makeTargetDefined(target: string, code: number, output: string): boolean {
	if (code === 0) return true;
	if (/No targets specified and no makefile found/i.test(output)) return false;
	const noRule = new RegExp(`No rule to make target [\`'"‘]${target.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}['"’]`);
	return !noRule.test(output);
}

/** Pure: is the operator-level escape hatch set in pi's own environment? */
export function skipRequested(env: Record<string, string | undefined>): boolean {
	return env.AILANG_SKIP_PREPUSH === "1";
}

/** Pure: does this bash command push to a remote? */
export function isPushCommand(command: string | undefined): boolean {
	if (!command) return false;
	return /\bgit push\b/.test(command) || /\bgh pr (create|merge)\b/.test(command);
}

export interface GateResult {
	blocked: boolean;
	reason: string;
	steps: Array<{ name: string; ok: boolean; detail: string }>;
}

export default function (pi: ExtensionAPI) {
	// gate state per session: gates pass once per session unless new changes
	// land afterwards (dirty set grows) — then re-arm. Simple v1: run on every
	// push, cache the PASS for the session (a passing gate + no new edits stays
	// valid; new edits reset it via the edit/write tracker).
	let gateResult: { hash: string; blocked: boolean; message: string } | null = null;

	pi.on("tool_call", async (event, ctx) => {
		if (event.toolName !== "bash") return;
		const command = (event.input as { command?: string } | undefined)?.command ?? "";
		if (!isPushCommand(command)) return;

		// These are ailang's gates — gofmt over cmd/ and internal/, `make lint`,
		// `make check-file-sizes`. This extension is baked into the pi image, so
		// it runs in EVERY repo an agent is dispatched to, including ones with no
		// Go and no such make targets.
		//
		// In ailang-packages (an AILANG monorepo, zero Go sources) the gate could
		// never pass, so every push was blocked. Measured 2026-09-07 on
		// task-f8ec8f5e: the agent made the requested fix, committed it, then spent
		// its whole remaining budget failing to push — and tried to get out by
		// writing a fake `gofmt` shim into /usr/local/bin. That is what a gate
		// which cannot pass teaches an unsupervised agent to do; the work was
		// correct and was lost anyway.
		//
		// A gate for a toolchain the repo does not use has nothing to check.
		// Skipping is not failing open — there is no Go here to be unformatted.
		if (skipRequested(process.env)) return;

		const roots = await trackedGoRoots(pi);
		if (roots.length === 0) return;

		// Gate inputs, honoring the Subprocess Contract (timeouts, caps).
		const gofmt = await pi.exec("gofmt", ["-l", ...roots.map((r) => `${r}/`)], { timeout: 30_000 });
		const unformatted = (gofmt.stdout ?? "").split("\n").filter((l) => l.trim() !== "");
		if (unformatted.length > 0) {
			return {
				block: true,
				reason: `gofmt: ${unformatted.length} unformatted Go file(s): ${unformatted.slice(0, 3).join(", ")}. Run gofmt -w on them, then push again.`,
			};
		}

		// Lint + file-size gates: only the targets this repo's Makefile defines.
		// A repo without them has its own CI gates, which re-run on the PR.
		const defined = async (target: string) => {
			const probe = await pi.exec("make", ["-n", target], { timeout: 15_000 });
			return makeTargetDefined(target, probe.code ?? 1, `${probe.stderr ?? ""}\n${probe.stdout ?? ""}`);
		};
		if (await defined("lint")) {
			const lint = await pi.exec("make", ["lint"], { timeout: 180_000 });
			if (lint.code !== 0) {
				const tail = (lint.stderr ?? lint.stdout ?? "").split("\n").filter(Boolean).slice(-6).join("\n");
				return { block: true, reason: `prepush gate failed (lint):\n${tail.slice(0, 2000)}\n\nFix locally, then push again.` };
			}
		}
		if (await defined("check-file-sizes")) {
			const sizes = await pi.exec("make", ["check-file-sizes"], { timeout: 15_000 });
			if (sizes.code !== 0) {
				return { block: true, reason: `prepush gate failed (file sizes >800 lines):\n${(sizes.stdout ?? "").slice(-400)}` };
			}
		}
		// Gate passed this session — subsequent pushes in the same session skip
		// the (slow) re-run unless the tree changes again (tracked via edits).
		return;
	});
}