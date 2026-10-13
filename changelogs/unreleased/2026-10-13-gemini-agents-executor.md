### Added — `gemini_agents` executor: Gemini Developer API managed agents that edit the local workspace

A sibling of `managed_agents` in the same package. It speaks the interactions contract to the
Gemini Developer API, which, unlike Vertex, accepts `repository`/`inline` sources, serves the
`antigravity-preview-09-2026` agent, and has a files API. The executor:
- mounts the workspace's GitHub origin at the merge-base with the remote;
- ships local changes (unpushed commits, uncommitted edits, untracked files) as one inline patch;
- has the agent export a diff, downloads it, `git apply`s it locally, and deletes the sandbox.

Live-verified with `ailang exec gemini_agents`: a scratch clone with an uncommitted edit got the
agent's edits applied on top. It took 131 s and cost $0.058. Auth is `GEMINI_AGENTS_API_KEY`
(new config var) from the locked `ailang-gemini-agents` project. New opt-in registry row:
`gemini-agents-antigravity`. The Cloud Run accept-set pin goes from 77 to 88 cells and from 41 to
52 accepted, because `gemini_agents` is binaryless like `managed_agents`.
