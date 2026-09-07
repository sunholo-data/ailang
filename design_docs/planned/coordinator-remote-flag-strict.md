# Design: Strict Flag Parsing for Hand-Rolled coordinator Subcommands

**Status:** Planned
**Target:** next minor release
**Scope:** cmd/ailang only — no language/semantics change

## Problem

Five hand-rolled flag parsers in `cmd/ailang` (in `coordinator_list.go`,
`coordinator_lifecycle.go`, `coordinator_inspect.go`) have `switch` statements
with no `default` case. An unrecognised flag is silently swallowed. For
subcommands that nominally accept `--remote` but ignore it, this makes the
silence worse: a user typo or an unsupported flag looks like success.

## Proposal

Add a `default` case to each of the five parsers that returns:

```
unknown flag: %s
```

and exits non-zero with usage text. This is the only change. It also converts
"`--remote` is accepted but ignored" into either an explicit rejection or a
documented no-op — surfaced instead of swallowed.

## Risk (one)

A flag that is **currently silently accepted** in some existing script, CI
job, or coordinator task invocation will become a hard error. **How to check:**
grep the repo (`rg 'ailang coordinator (list|inspect|start|stop|status|reopen)'`),
coordinator task templates, and launchd/CI configs for invocations of the
affected subcommands, and confirm every flag passed is one the parser already
handles. Anything unknown found there must be fixed in the same PR.

## Acceptance Criteria

1. `ailang coordinator <subcmd> --bogus` (for each of the five parsers)
   exits non-zero and prints `unknown flag: --bogus`.
2. All existing repo-internal invocations of these subcommands (scripts,
   coordinator tasks, launchd plists, CI) still parse cleanly — verified by
   the grep audit above plus `make test`.
