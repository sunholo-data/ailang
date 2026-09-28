export const bashCalls = { local: [], sandboxed: [] };

export function createBashTool(cwd, options) {
	return {
		name: "bash",
		async execute(id, params, signal, onUpdate) {
			if (!options?.operations) {
				bashCalls.local.push({ id, params, signal, onUpdate, cwd });
				return { content: [{ type: "text", text: "local bash stub" }] };
			}
			bashCalls.sandboxed.push({ id, params, cwd });
			return options.operations.exec(params.command, cwd, { onData() {}, signal });
		},
	};
}

export function getAgentDir() { return "/nonexistent/pi-agent-for-sandbox-test"; }
