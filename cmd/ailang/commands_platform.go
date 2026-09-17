package main

import (
	"fmt"
	"io"
	"os"
)

// platformCommands are the commands that operate the fleet rather than the
// language: the message plane, the coordinator, the observatory, the registry
// and the local services. They keep the startup probes (observatory health +
// stale binary) that language commands must not run.
//
// S5 M2 moves most of these under the hidden `dev` and `ops` groups with the
// current spellings kept as aliases. M1 changes no spelling.
func platformCommands() []Command {
	return []Command{
		{
			Name:    "doctor",
			Summary: "Environment diagnostics (builtins, memory, managed_agents, ...)",
			Run:     noArgs(runDoctor),
		},
		{
			Name:    "messages",
			Aliases: []string{"msg"},
			Summary: "Agent message plane: list, send, read, ack",
			Run:     noArgs(messagesCommand),
		},
		{
			Name:    "cache",
			Aliases: []string{"brain"},
			Summary: "Project brain / semantic cache operations",
			Run:     noArgs(cacheCommand),
		},
		{
			Name:    "micro-rag",
			Aliases: []string{"microrag", "urag"},
			Summary: "Micro-RAG index and retrieval operations",
			Run:     noArgs(microragCommand),
		},
		{
			Name:    "agent-prompt",
			Summary: "Display the AILANG agent prompt",
			Run:     noArgs(runAgentPrompt),
		},
		{
			Name:    "mcp",
			Summary: "Model Context Protocol server operations",
			Run:     noArgs(runMCPCommand),
		},
		{
			Name:    "daemon",
			Summary: "Local ailang daemon: run, install, uninstall, status",
			Run:     runDaemonCommand,
		},
		{
			Name: "server",
			// "serve" kept as alias for backward compatibility
			Aliases: []string{"serve"},
			Summary: "Start the Observatory dashboard server (default port 1957)",
			Run:     serverCommand,
		},
		{
			Name:    "serve-api",
			Summary: "Serve AILANG exports as REST endpoints",
			Run:     serveAPICommand,
		},
		{
			Name:    "access-control",
			Summary: "Access control management",
			Run:     accessControlCommand,
		},
		{
			Name:    "editor",
			Summary: "Install editor syntax highlighting (vscode, vim, neovim)",
			Run:     noArgs(editorCommand),
		},
		{
			Name:    "pi",
			Summary: "Manage pi extensions (install, uninstall, status)",
			Run:     noArgs(piCommand),
		},
		{
			Name:    "axioms",
			Summary: "Design axiom compliance scorecard",
			Run:     noArgs(axiomsCommand),
		},
		{
			Name:    "trace",
			Summary: "Distributed trace management",
			Run:     noArgs(traceCommand),
		},
		{
			Name:    "observatory",
			Summary: "Observatory analytics",
			Run:     noArgs(observatoryCommand),
		},
		{
			Name:    "chains",
			Summary: "View execution chains (task -> session -> chat linkage)",
			Run:     noArgs(chainsCommand),
		},
		{
			Name:    "dashboard",
			Summary: "Dashboard operations for task visualization",
			Run:     noArgs(dashboardCommand),
		},
		{
			Name:    "budget",
			Summary: "Budget monitoring",
			Run:     noArgs(budgetCommand),
		},
		{
			Name:    "models",
			Summary: "Model registry: list, show, pricing",
			Run:     modelsCommand,
		},
		{
			Name:    "mission",
			Summary: "Mission registry, iteration, role dispatch and reports",
			// mission maps error classes onto exit codes, so it reports and
			// exits inside its own closure instead of returning the error.
			Run: func(args []string) error {
				if err := missionCommand(args); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
					os.Exit(missionErrorExitCode(err))
				}
				return nil
			},
		},
		{
			Name:    "coordinator",
			Summary: "Manage the autonomous agent daemon",
			Run:     coordinatorCommand,
		},
		{
			Name:    "storage",
			Summary: "Storage plane status and migration",
			Run:     storageCommand,
		},
		{
			Name:    "workspaces",
			Summary: "Workspace management",
			Run:     workspacesCommand,
		},
		{
			Name:    "exec",
			Summary: "Unified AI execution for programmatic use",
			Run:     noArgs(runExec),
		},
		{
			Name:    "design-review",
			Summary: "AI review of a design document",
			Run:     noArgs(runDesignReview),
		},
		{
			Name:    "design-quorum",
			Summary: "N-reviewer design quorum, reject-by-default",
			Run:     noArgs(runDesignQuorum),
		},
		{
			Name:    "add",
			Summary: "Add a dependency (--path, --git or --registry)",
			Run:     pkgAddCommand,
		},
		{
			Name:    "lock",
			Summary: "Resolve dependencies and write the lockfile",
			Run:     pkgLockCommand,
		},
		{
			Name:    "tree",
			Summary: "Show the dependency tree",
			Run:     pkgTreeCommand,
		},
		{
			Name:    "install",
			Summary: "Install a package (omit the version for latest)",
			Run:     pkgInstallCommand,
		},
		{
			Name:    "search",
			Summary: "Search the registry by keyword or tag",
			Run:     pkgSearchCommand,
		},
		{
			Name:    "publish",
			Summary: "Publish the current package to the registry",
			Run:     pkgPublishCommand,
		},
		{
			Name:    "unpublish",
			Summary: "Remove a package version from the registry",
			Run:     pkgUnpublishCommand,
		},
		{
			Name:    "generate-extension-registry",
			Summary: "Emit the static extension dispatch file from [extensions]",
			Run:     extRegistryGenCommand,
		},
		{
			Name:    "pkg-docs",
			Summary: "Display a package's AGENT.md (AI usage guide)",
			Run:     pkgDocsCommand,
		},
		{
			Name:    "pkg",
			Summary: "Package registry inspection and coordination",
			Run:     pkgGroupCommand,
		},
		{
			Name:    "policy-check",
			Summary: "Evaluate a policy program against its cases",
			Run:     noArgs(policyCheckCommand),
		},
		{
			Name:    "sandbox-check",
			Summary: "Diagnose FS sandbox path resolution (AILANG_FS_SANDBOX)",
			Run: func(args []string) error {
				sandboxCheckCommand(args)
				return nil
			},
		},
	}
}

// runDaemonCommand is `ailang daemon`. It owns its own help because the same
// block serves two paths: `--help` (exit 0) and NO arguments (exit 1).
//
// Before S5 M1 a bare `ailang daemon` defaulted its subcommand to "run" and
// STARTED the daemon — the most dangerous default in the CLI, and the reason
// `daemon` is called out in the Phase 3 plan. It now prints this help and
// exits 1 without starting anything; `ailang daemon run` is unchanged.
func runDaemonCommand(args []string) error {
	if wantsHelp(args) {
		printDaemonHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		printDaemonHelp(os.Stdout)
		os.Exit(1)
	}
	daemonCommand()
	return nil
}

func printDaemonHelp(w io.Writer) {
	fmt.Fprintf(w, "%s - Local ailang daemon\n\n", bold("ailang daemon"))
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ailang daemon <subcommand>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintf(w, "  %s        Run the daemon in the foreground\n", cyan("run"))
	fmt.Fprintf(w, "  %s    Install the daemon as a background service\n", cyan("install"))
	fmt.Fprintf(w, "  %s  Remove the installed service\n", cyan("uninstall"))
	fmt.Fprintf(w, "  %s     Show service status and recent log lines\n", cyan("status"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "A bare 'ailang daemon' no longer starts the daemon — name the subcommand.")
}
