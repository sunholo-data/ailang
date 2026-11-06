### Added — `lean` agent tool profile (minimal harness context)

`--tool-policy lean` (or `tool_policy: lean`) runs an agent with the four core tools (read, bash,
edit, write) and **without the harness's ambient context**. On pi:
- a short fixed system prompt (`pi.LeanSystemPrompt`) replaces pi's default;
- skills, extension, context-file and prompt-template discovery are off;
- explicitly loaded extensions, such as the eval shell guard, still run.

The new `executor.Task.MinimalHarnessContext` carries the setting; executors without these
controls ignore it.

Measured on the rig (2026-11-06), the prefix before every turn drops from **7,338 tokens to
~910**. pi's default system prompt was 4,424 tokens, 3,625 of them a skills catalog discovered
from the machine; the 16 tool schemas were ~2.9k. This is the lane fine-tuned AILANG models are
trained and served on (design doc m-eval-finetuning-data-pipeline, D-FT-9).
