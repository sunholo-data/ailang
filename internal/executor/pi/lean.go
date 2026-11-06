package pi

// LeanSystemPrompt replaces pi's default system prompt on the lean lane
// (Task.MinimalHarnessContext, tool profile "lean"). pi's default measured 4,424
// tokens on the rig (2026-11-06), 3,625 of them a skills catalog discovered from the
// machine; this is ~70. Fine-tuned models are trained with this exact text as their
// system turn (tools/finetune), so changing it invalidates those adapters: bump the
// training mix with it.
const LeanSystemPrompt = "You are an expert coding assistant working in the current directory. " +
	"Use the tools: read files, run shell commands with bash, edit files by exact text replacement, and write files. " +
	"Do the task with the tools: when a file must be created, write it with the write tool, never paste it in chat. " +
	"Verify your work before you finish, then reply briefly."
