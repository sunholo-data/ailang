import assert from "node:assert/strict";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { register } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { bashCalls } from "./test-stubs/pi-coding-agent.mjs";
import { runtimeCalls } from "./test-stubs/sandbox-runtime.mjs";

register("./test-stubs/loader.mjs", import.meta.url);
const { default: sandboxExtension } = await import("./index.ts");

type Handler = (...args: any[]) => any;

function registerExtension() {
	const handlers = new Map<string, Handler>();
	let bash: { execute: Handler } | undefined;
	sandboxExtension({
		registerFlag() {},
		getFlag() { return false; },
		registerTool(tool: { execute: Handler }) { bash = tool; },
		on(name: string, handler: Handler) { handlers.set(name, handler); },
		registerCommand() {},
	} as any);
	assert.ok(bash, "extension must register bash");
	assert.ok(handlers.has("session_start"));
	assert.ok(handlers.has("user_bash"));
	return { bash, sessionStart: handlers.get("session_start")!, userBash: handlers.get("user_bash")! };
}

function context(cwd: string) {
	return { cwd, ui: { notify() {}, setStatus() {}, theme: { fg(_color: string, value: string) { return value; } } } };
}

function withEnvironment(run: (dir: string, ready: string) => Promise<void>) {
	return async () => {
		const oldPolicy = process.env.PI_SANDBOX_POLICY_FILE;
		const oldReady = process.env.PI_SANDBOX_READY_FILE;
		const dir = mkdtempSync(join(tmpdir(), "pi-sandbox-handler-"));
		const ready = join(dir, "ready");
		bashCalls.local.length = 0;
		bashCalls.sandboxed.length = 0;
		runtimeCalls.initialize.length = 0;
		runtimeCalls.reset = 0;
		runtimeCalls.initializeError = undefined;
		try { await run(dir, ready); }
		finally {
			if (oldPolicy === undefined) delete process.env.PI_SANDBOX_POLICY_FILE;
			else process.env.PI_SANDBOX_POLICY_FILE = oldPolicy;
			if (oldReady === undefined) delete process.env.PI_SANDBOX_READY_FILE;
			else process.env.PI_SANDBOX_READY_FILE = oldReady;
			rmSync(dir, { recursive: true, force: true });
		}
	};
}

function mission(dir: string, ready: string, policy: object) {
	const file = join(dir, "policy.json");
	writeFileSync(file, JSON.stringify(policy));
	process.env.PI_SANDBOX_POLICY_FILE = file;
	process.env.PI_SANDBOX_READY_FILE = ready;
}

const validPolicy = { network: { allowedDomains: [], deniedDomains: [] }, filesystem: { allowWrite: ["."], denyWrite: [], denyRead: [] } };
const unsupportedPlatform = process.platform !== "darwin" && process.platform !== "linux";

test("A: failed mission initialization refuses registered bash without local execution", { skip: unsupportedPlatform }, withEnvironment(async (dir, ready) => {
	mission(dir, ready, validPolicy);
	runtimeCalls.initializeError = new Error("init failed");
	const extension = registerExtension();
	await assert.rejects(extension.sessionStart({}, context(dir)), /init failed/);
	await assert.rejects(extension.bash.execute("id", { command: "echo forbidden" }), /not initialized/);
	assert.equal(bashCalls.local.length, 0, "unsandboxed bash must never run");
	assert.equal(bashCalls.sandboxed.length, 0);
	assert.equal(existsSync(ready), false);
}));

test("B: failed mission initialization refuses registered user_bash operations", { skip: unsupportedPlatform }, withEnvironment(async (dir, ready) => {
	mission(dir, ready, validPolicy);
	runtimeCalls.initializeError = new Error("init failed");
	const extension = registerExtension();
	await assert.rejects(extension.sessionStart({}, context(dir)), /init failed/);
	const result = extension.userBash();
	assert.ok(result?.operations?.exec, "user_bash must provide refusing operations");
	await assert.rejects(result.operations.exec("echo forbidden", dir, { onData() {} }), /not initialized/);
	assert.equal(bashCalls.local.length, 0);
	assert.equal(bashCalls.sandboxed.length, 0);
}));

test("C: mission policy without enabled initializes once and writes readiness", { skip: unsupportedPlatform }, withEnvironment(async (dir, ready) => {
	mission(dir, ready, validPolicy);
	const extension = registerExtension();
	await extension.sessionStart({}, context(dir));
	assert.equal(runtimeCalls.initialize.length, 1);
	assert.deepEqual(runtimeCalls.initialize[0].filesystem, validPolicy.filesystem);
	assert.equal(readFileSync(ready, "utf8"), "ready\n");
}));

test("D: disabled mission policy rejects session_start and bash", withEnvironment(async (dir, ready) => {
	mission(dir, ready, { ...validPolicy, enabled: false });
	const extension = registerExtension();
	await assert.rejects(extension.sessionStart({}, context(dir)), /Invalid mission sandbox policy/);
	await assert.rejects(extension.bash.execute("id", { command: "echo forbidden" }), /not initialized/);
	assert.equal(runtimeCalls.initialize.length, 0);
	assert.equal(bashCalls.local.length, 0);
	assert.equal(existsSync(ready), false);
}));

test("missing mission policy rejects registered session_start and bash", withEnvironment(async (dir, ready) => {
	process.env.PI_SANDBOX_POLICY_FILE = join(dir, "missing.json");
	process.env.PI_SANDBOX_READY_FILE = ready;
	const extension = registerExtension();
	await assert.rejects(extension.sessionStart({}, context(dir)), /ENOENT/);
	await assert.rejects(extension.bash.execute("id", { command: "echo forbidden" }), /not initialized/);
	assert.equal(runtimeCalls.initialize.length, 0);
	assert.equal(bashCalls.local.length, 0);
}));

test("malformed mission policy rejects registered session_start and bash", withEnvironment(async (dir, ready) => {
	const policy = join(dir, "policy.json");
	writeFileSync(policy, "{");
	process.env.PI_SANDBOX_POLICY_FILE = policy;
	process.env.PI_SANDBOX_READY_FILE = ready;
	const extension = registerExtension();
	await assert.rejects(extension.sessionStart({}, context(dir)), SyntaxError);
	await assert.rejects(extension.bash.execute("id", { command: "echo forbidden" }), /not initialized/);
	assert.equal(runtimeCalls.initialize.length, 0);
	assert.equal(bashCalls.local.length, 0);
}));

test("E: non-mission uninitialized bash retains local fallback", withEnvironment(async (dir) => {
	delete process.env.PI_SANDBOX_POLICY_FILE;
	delete process.env.PI_SANDBOX_READY_FILE;
	const extension = registerExtension();
	assert.equal(extension.userBash(), undefined);
	const result = await extension.bash.execute("id", { command: "echo local" });
	assert.deepEqual(result, { content: [{ type: "text", text: "local bash stub" }] });
	assert.equal(bashCalls.local.length, 1);
	assert.equal(bashCalls.local[0].params.command, "echo local");
	assert.equal(runtimeCalls.initialize.length, 0);
}));
