### Added — Claude Fable 5.2 in the eval registry

`claude-fable-5-2` (released 2026-11-09) is now a model row: $8/$40 per 1M, cache reads $0.20/M
(0.025x input), 1M context, 64000 output cap (family-equal), adaptive thinking always on. Access
verified on the Messages API and the claude CLI subscription lane; registered as
`{Adaptive:true, CanDisable:false}` in `internal/ai/reasoning_anthropic.go`.

### Changed — Fable 5.2 takes the Fable seat in `extended_suite`

`claude-fable-5-2` replaces `claude-fable-5-1` in `extended_suite`, at 20% lower cost than 5.1.
`claude-fable-5-1` stays opt-in as the longitudinal anchor.
