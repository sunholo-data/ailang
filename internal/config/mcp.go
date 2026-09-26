package config

// The teaching prompt's fresh-fetch path and the MCP endpoint behind it.
const (
	EnvMCPURL         = "AILANG_MCP_URL"
	EnvMCPQuiet       = "AILANG_MCP_QUIET"
	EnvMCPVerbose     = "AILANG_MCP_VERBOSE"
	EnvNoGitHubSearch = "AILANG_NO_GITHUB_SEARCH"
)

var mcpVars = []Var{
	{EnvMCPURL, "", AreaMCP, "MCP endpoint `ailang prompt` and `mcp status` fetch the canonical prompt from; unset means the production endpoint."},
	{EnvMCPQuiet, "", AreaMCP, "Set to anything to silence the prompt-source note `ailang prompt` prints on a terminal."},
	{EnvMCPVerbose, "", AreaMCP, "Set to anything to print the prompt-source note even when stdout is not a terminal."},
	{EnvNoGitHubSearch, "", AreaMCP, "Set to anything to stop `ailang docs search` falling back to the GitHub backend when local docs are missing."},
}

// MCPURL returns AILANG_MCP_URL, "" when unset.
func MCPURL() string { return get(EnvMCPURL) }

// MCPQuiet reports whether AILANG_MCP_QUIET is set.
func MCPQuiet() bool { return get(EnvMCPQuiet) != "" }

// MCPVerbose reports whether AILANG_MCP_VERBOSE is set.
func MCPVerbose() bool { return get(EnvMCPVerbose) != "" }

// NoGitHubSearch reports whether AILANG_NO_GITHUB_SEARCH is set.
func NoGitHubSearch() bool { return get(EnvNoGitHubSearch) != "" }
