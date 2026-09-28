const stubs = new Map([
	["@anthropic-ai/sandbox-runtime", new URL("./sandbox-runtime.mjs", import.meta.url).href],
	["@earendil-works/pi-coding-agent", new URL("./pi-coding-agent.mjs", import.meta.url).href],
]);

export function resolve(specifier, context, nextResolve) {
	const url = stubs.get(specifier);
	return url ? { url, shortCircuit: true } : nextResolve(specifier, context);
}
