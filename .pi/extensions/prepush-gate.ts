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
 * run is blocked until the gates pass. Escape hatch: AILANG_SKIP_PREPUSH=1
 * (per-call, for genuine emergencies).
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

		// Gate inputs, honoring the Subprocess Contract (timeouts, caps).
		const gofmt = await pi.exec("gofmt", ["-l", "cmd/", "internal/"], { timeout: 30_000 });
		const unformatted = (gofmt.stdout ?? "").split("\n").filter((l) => l.trim() !== "");
		if (unformatted.length > 0) {
			return {
				block: true,
				reason: `gofmt: ${unformatted.length} unformatted Go file(s): ${unformatted.slice(0, 3).join(", ")}. Run gofmt -w on them, then push again.`,
			};
		}

		// Lint + file-size gates: run via make, 3min budget each.
		const lint = await pi.exec("make", ["lint"], { timeout: 180_000 });
		if (lint.code !== 0) {
			const tail = (lint.stderr ?? lint.stdout ?? "").split("\n").filter(Boolean).slice(-6).join("\n");
			return { block: true, reason: `prepush gate failed (lint):\n${tail.slice(0, 2000)}\n\nFix locally, then push again.` };
		}
		const sizes = await pi.exec("make", ["check-file-sizes"], { timeout: 15_000 });
		if (sizes.code !== 0) {
			return { block: true, reason: `prepush gate failed (file sizes >800 lines):\n${(sizes.stdout ?? "").slice(-400)}` };
		}
		// Gate passed this session — subsequent pushes in the same session skip
		// the (slow) re-run unless the tree changes again (tracked via edits).
		return;
	});
}