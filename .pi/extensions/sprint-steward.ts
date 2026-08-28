/**
 * sprint-steward — M-DX-PI-HARNESS A2
 * Mechanical enforcement of the sprint JSON constrained-modification contract:
 * only `passes`/`started`/`completed`/`notes` (and sprint `status`) ever change;
 * everything else is validated and refused. M2/M3 depend on this state file.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

export interface SprintFeature {
	id: string;
	description: string;
	estimated_loc: number;
	dependencies: string[];
	acceptance_criteria: string[];
	passes: boolean | null;
	started: string | null;
	completed: string | null;
	notes: string | null;
}
export interface SprintJson {
	sprint_id: string;
	status: string;
	features: SprintFeature[];
	velocity: { estimated_total_loc: number; [k: string]: unknown };
	[k: string]: unknown;
}

export function readSprint(cwd: string, sprintId: string): SprintJson {
	const p = join(cwd, ".ailang", "state", "sprints", `sprint_${sprintId}.json`);
	return JSON.parse(readFileSync(p, "utf8")) as SprintJson;
}

/** Pure: apply a constrained update to one milestone. Throws on non-conforming result. */
export function applyMilestoneUpdate(
	d: SprintJson,
	milestoneId: string,
	update: { passes?: boolean; notes?: string },
): SprintJson {
	const f = d.features.find((x) => x.id === milestoneId);
	if (!f) throw new Error(`milestone '${milestoneId}' not in sprint ${d.sprint_id}`);
	const next: SprintJson = JSON.parse(JSON.stringify(d)); // deep copy
	const tgt = next.features.find((x) => x.id === milestoneId)!;
	if (update.passes !== undefined) tgt.passes = update.passes;
	if (update.passes === true && !tgt.completed) tgt.completed = new Date().toISOString();
	if (update.passes === true && !tgt.started) tgt.started = tgt.completed;
	if (update.notes !== undefined) tgt.notes = update.notes;
	validate(next);
	return next;
}

/** Pure: validate the constrained-modification contract on a candidate state. */
export function validate(d: SprintJson): void {
	if (!Array.isArray(d.features) || d.features.length < 2) {
		throw new Error("sprint must keep >=2 features");
	}
	for (const f of d.features) {
		if (!/^[A-Z0-9][A-Z0-9_-]*$/i.test(f.id) || f.id === "MILESTONE_ID") {
			throw new Error(`placeholder milestone id: '${f.id}'`);
		}
		if (!Array.isArray(f.acceptance_criteria) || f.acceptance_criteria.some((c) => c === "Criterion 1")) {
			throw new Error(`placeholder acceptance criteria in '${f.id}'`);
		}
		if (typeof f.estimated_loc !== "number" || f.passes === undefined) {
			throw new Error(`feature '${f.id}' missing required fields`);
		}
	}
	const loc = d.features.reduce((s, x) => s + x.estimated_loc, 0);
	if (d.velocity && d.velocity.estimated_total_loc !== loc) {
		throw new Error(`estimated_total_loc ${d.velocity.estimated_total_loc} != sum ${loc}`);
	}
}

export default function (pi: ExtensionAPI) {
	pi.registerCommand("sprint-start", {
		description: "Mark a sprint in_progress: /sprint-start <sprint-id>",
		handler: async (args, ctx) => {
			const id = (args ?? "").trim();
			if (!id) return ctx.ui.notify("Usage: /sprint-start <sprint-id>", "warning");
			const d = readSprint(ctx.cwd, id);
			d.status = "in_progress";
			const first = d.features.find((x) => x.passes !== true);
			if (first && !first.started) first.started = new Date().toISOString();
			validate(d);
			writeFileSync(join(ctx.cwd, ".ailang", "state", "sprints", `sprint_${id}.json`), JSON.stringify(d, null, 2));
			await ctx.ui.notify(`sprint ${id}: in_progress`, "info");
		},
	});

	pi.registerCommand("sprint-complete", {
		description: "Mark one milestone passed: /sprint-complete <sprint-id> <milestone-id>",
		handler: async (args, ctx) => {
			const [id, mid] = (args ?? "").trim().split(/\s+/);
			if (!id || !mid) return ctx.ui.notify("Usage: /sprint-complete <sprint-id> <milestone-id>", "warning");
			const d = readSprint(ctx.cwd, id);
			const next = applyMilestoneUpdate(d, mid, { passes: true });
			writeFileSync(join(ctx.cwd, ".ailang", "state", "sprints", `sprint_${id}.json`), JSON.stringify(next, null, 2));
			const done = next.features.filter((x) => x.passes === true).length;
			await ctx.ui.notify(`sprint ${id}: ${done}/${next.features.length} milestones passed`, "info");
		},
	});
}