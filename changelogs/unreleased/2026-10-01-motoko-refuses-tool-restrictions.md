### Fixed — motoko refuses tool restrictions it cannot enforce

The motoko executor ignored `Task.AllowedTools` and the program policy. A task
that restricted tools (the `ailang_only` lane, or the coordinator's read-only
question list) ran with motoko's full native surface, including bash and file
writes, and its result recorded the tool policy as nil ("unmeasured"). motoko
now refuses, before starting, any task that forbids a tool it always exposes
(Bash, Read, Write, Edit, Grep) or that carries a program-policy file. The error
names what was forbidden and points at the pi executor. Runs it accepts record
the native surface as `ToolPolicy`. The eval harness's standard tool list is
unaffected.
