### Fixed — nested runtime stop test waits for both helpers

The deadline regression test now waits for the outer and nested readiness
markers before requesting stop, so both cleanup handlers are registered when
the test exercises the shared deadline. See the
[fixture diagnosis](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_57_4/m-triage29-nested-stop-nil-join.md)
and issue #2069.
