### Fixed — Mission helpers handle empty glob matches in zsh (2026-10-22)

Quorum artifact lookup no longer aborts when an export or local artifact directory
is empty, and pi extension arguments no longer abort when `.pi/extensions` has no
TypeScript files. Skill snippets source these helpers into the controller's zsh
tool shell, where NOMATCH previously stopped both loops before returning.

Both functions now enable NULL_GLOB with LOCAL_OPTIONS for the duration of the
call, restoring the caller's options on return. Bash 3.2 behavior is unchanged.
Behavioral tests cover both Gate-2 skill copies, exact results, option restoration,
and missing completion markers. See the [sprint plan](../../design_docs/planned/sprint-plan-zsh-safe-mission-libs.md).
