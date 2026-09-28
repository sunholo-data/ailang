export const runtimeCalls = { initialize: [], reset: 0, initializeError: undefined };

export const SandboxManager = {
	async initialize(config) {
		runtimeCalls.initialize.push(config);
		if (runtimeCalls.initializeError) throw runtimeCalls.initializeError;
	},
	async reset() { runtimeCalls.reset++; },
	async wrapWithSandbox(command) { return command; },
};
