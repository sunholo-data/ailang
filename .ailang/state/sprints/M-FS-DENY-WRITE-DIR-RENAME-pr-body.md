Refs #1569

Filesystem deny patterns now protect matching ancestors during rename and removal. Both rename operands and every removal entry point use the shared structural matcher, closing literal file-pattern, nested `/**`, and mid-glob bypasses. Refusals identify the original pattern and ancestor relationship with `E_FS_PROTECTED`.

Ordinary writes and mkdirs retain their direct-path checks; basename policies still protect files after allowed directory moves. Regression tests verify denial before mutation, unchanged disk state, destination planting, empty ancestor removal, and allowed atomic publishing.

Validation results are recorded in the sprint progress JSON and independent evaluation report.
