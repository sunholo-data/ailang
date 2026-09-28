### Fixed — coordinator PR titles name the work, not a hash (2026-09-28)

Three stages of one request read `Daneel design b5f7599f…`, `Handoff: Daneel design b5f7599f…`,
`Handoff: Daneel design b5f7599f…` (#1347, #1356, #1367): the PR title is the task's own title, and
Daneel titles its requests with a hash while every handoff wraps the parent's title. Titles now drop
the `Handoff:` / `(approved)` wrapping, and a title that is still an id yields to the subject of the
original request — read from its JSON `request` even when a handoff truncated it mid-field, first
line, `Subject:`/`Re:`/`Fwd:` and "Create a design document for" lead-ins removed. Replayed over the
last 30 coordinator PRs: every hash title became readable (#1347/#1356/#1367 → "AILANG package
registry"); human titles unchanged apart from the dropped `Handoff:`.
