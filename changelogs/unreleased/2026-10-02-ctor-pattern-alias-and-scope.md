### Fixed — aliased constructor imports never matched; unknown constructor patterns compiled clean (2026-10-02)

`import std/option (None as Nada)` was accepted but the alias was bound nowhere: `Nada` in a pattern
silently never matched (on the evaluator; the strict VM rejected the entry), and `Nada` in an
expression failed `undefined variable`. Constructor aliases now bind in patterns and expressions and
match the original constructor on every route, including next to a same-named local constructor
(the reported `Arrived as TripArrived` clash). A constructor pattern naming a constructor that no
local declaration, import, or loaded module defines (`Bogus`, or a typo) is now a compile error,
`TC_MATCH_001`, listing the scrutinee's constructors, instead of an arm that can never match.
Constructors known only through a loaded module keep matching by name (#323). Fixes #1478; design:
`design_docs/implemented/v0_51_1/m-ctor-pattern-alias-and-scope.md`.
