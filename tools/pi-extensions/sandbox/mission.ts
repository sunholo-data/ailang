import { readFileSync, writeFileSync } from "node:fs";
import { isAbsolute } from "node:path";

export interface MissionPolicy {
	enabled?: boolean;
	filesystem: { allowWrite: string[]; denyWrite: string[]; denyRead: string[] };
	network: { allowedDomains: string[]; deniedDomains: string[] };
	ignoreViolations?: Record<string, string[]>;
	enableWeakerNestedSandbox?: boolean;
}

const strings = (value: unknown): value is string[] =>
	Array.isArray(value) && value.every((item) => typeof item === "string");
const object = (value: unknown): value is Record<string, unknown> =>
	typeof value === "object" && value !== null && !Array.isArray(value);

export function loadMissionPolicy(path: string): MissionPolicy {
	if (!isAbsolute(path)) throw new Error("PI_SANDBOX_POLICY_FILE must be absolute");
	const policy: unknown = JSON.parse(readFileSync(path, "utf8"));
	if (!object(policy) || policy.enabled === false || !object(policy.filesystem) || !object(policy.network) ||
		!strings(policy.filesystem.allowWrite) || !strings(policy.filesystem.denyWrite) ||
		!strings(policy.filesystem.denyRead) || !strings(policy.network.allowedDomains) ||
		!strings(policy.network.deniedDomains)) {
		throw new Error("Invalid mission sandbox policy");
	}
	return policy as unknown as MissionPolicy;
}

export function missionBashAllowed(missionMode: boolean, initialized: boolean): boolean {
	return !missionMode || initialized;
}

export async function initializeMissionSandbox(
	initialize: () => Promise<void>, readyFile: string,
): Promise<void> {
	if (!isAbsolute(readyFile)) throw new Error("PI_SANDBOX_READY_FILE must be absolute");
	await initialize();
	writeFileSync(readyFile, "ready\n", { flag: "wx" });
}
