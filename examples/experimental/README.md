# Experimental Examples

Examples demonstrating planned AILANG features that are not yet implemented. These serve as design documentation and future test cases.

## Warning

**These examples DO NOT currently work!** They require features that are:
- On the [roadmap](/docs/roadmap/)
- Partially implemented
- Planned for future releases

## Current Contents

### Property-Based Testing (M-TESTING)
- `factorial.ail` - Inline tests and properties syntax
- `quicksort.ail` - Property-based testing for sorting

**Status**: Core algorithms work, but `tests [...]` and `properties [...]` syntax not implemented.
**Target**: v0.7.0+ (M-TESTING milestone)

### Concurrency (M-CHANNELS)
- `concurrent_pipeline.ail` - CSP-based data processing with channels
- `web_api.ail` - HTTP server with quasiquotes

**Status**: Requires CSP channels, session types, and quasiquotes.
**Target**: v0.8.0+ (long-term roadmap)

### REPL Introspection
- `ai_agent_integration.ail` - Effects introspection (`:effects` command)

**Status**: Requires REPL meta-commands not exposed to programs.
**Target**: v0.7.0 (M-REPL1)

## Graduated to runnable/

The following examples now work and have been moved to `runnable/`:
- `ai_call.ail` - OpenAI API integration (Net effect)
- `claude_haiku_call.ail` - Claude API integration (Net effect)
- `demo_ai_api.ail` - HTTP API demonstration (Net effect)

## For Working Examples

See:
- `examples/runnable/` - Full working programs (CI-verified)
- `examples/docs/` - Documentation examples
- `examples/tests/` - Test cases
