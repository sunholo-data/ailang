# Sprint Retrospective: M-WITH-RECV-RESERVED-WORDS

Sequential execution on 2026-10-08; both implementation milestones complete.
Planned capacity: two working days / 400 authored lines. No reliable historical
velocity was available; no speedup or measured LOC/day is claimed.

M1 delivers shared reservation diagnostics and depth-aware local recovery, with
AST boundary, positioning, nested-type, independent-error and valid-traffic tests.
M2 delivers the canonical reference and synchronized discovery bundles. The
corrected runnable example has three passing inline tests.

The executor's full-suite scripts were deliberately replaced by bounded targeted
checks under the crash re-dispatch constraint. No compiler/sysroot was installed,
no full make test ran, and no push/PR was attempted. Go and jq were exposed from
existing/standalone utilities; build temporaries stayed on workspace disk.

Friction: the keyword map has four contextual identifier entries and IsKeyword
omits four other entries. The implementation preserves contextual traffic and
uses the existing reservation map without changing the lexer. Legacy docs
examples fail baseline checking on string ++. Standard npm build fails on a
baseline std/web index omission. Registry pages require the existing deployment
sync. Full site compilation was killed, so an isolated bounded Docusaurus build
verified the page and HTTP 200 route. Whole CLI tests fail with CGO-disabled SQLite
stubs and unrelated module paths; focused CLI checking tests pass.

Independent evaluation found no implementation defect (94/100). Future executor
scripts should accept targeted package and memory budgets; documentation build
prerequisites should be checked before expensive bundling. CI must validate the
full suite and deployment must verify the production URL.
