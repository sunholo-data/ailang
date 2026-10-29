package codex

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

var (
	mcpNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// promptFromStdin is codex exec's PROMPT argument meaning "read the prompt from stdin".
const promptFromStdin = "-"

// buildCodexArgs ends with "-": codex exec then reads the prompt from stdin, which the
// caller supplies. The prompt never goes on argv — see the claude executor's
// prompt_delivery.go for the measured cost (a sibling agent's `pkill -f "ailang run"`
// SIGTERMed every eval agent whose argv carried the teaching prompt).
func buildCodexArgs(task *executor.Task, model string) ([]string, error) {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--dangerously-bypass-approvals-and-sandbox",
		"--model", model,
	}
	if task != nil {
		for _, server := range task.MCPServers {
			if !mcpNamePattern.MatchString(server.Name) {
				return nil, fmt.Errorf("invalid MCP server name %q", server.Name)
			}
			if server.Command == "" {
				return nil, fmt.Errorf("MCP server %q command is required", server.Name)
			}
			prefix := "mcp_servers." + server.Name + "."
			args = append(args, "-c", prefix+"command="+strconv.Quote(server.Command))
			if len(server.Args) > 0 {
				args = append(args, "-c", prefix+"args="+tomlStringArray(server.Args))
			}
			if len(server.EnvVars) > 0 {
				for _, name := range server.EnvVars {
					if !envNamePattern.MatchString(name) {
						return nil, fmt.Errorf("MCP server %q has invalid environment variable name %q", server.Name, name)
					}
				}
				args = append(args, "-c", prefix+"env_vars="+tomlStringArray(server.EnvVars))
			}
			if server.Required {
				args = append(args, "-c", prefix+"required=true")
			}
		}
	}
	return append(args, promptFromStdin), nil
}

func tomlStringArray(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
