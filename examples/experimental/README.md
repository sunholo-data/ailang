# Experimental Examples

Examples using future or unimplemented AILANG features. These files demonstrate planned capabilities that are not yet fully implemented.

## ⚠️ Warning

**These examples DO NOT currently work!** They use features that are:
- Planned for future releases
- Partially implemented
- Require external dependencies not yet integrated

## Why Keep Them?

These files serve as:
- **Design documentation**: Show intended API and syntax
- **Feature tracking**: What capabilities users want
- **Test cases**: Will become working examples when features ship

## Contents

### AI Agent Integration
- `ai_call.ail` - Generic AI agent calls
- `claude_haiku_call.ail` - Claude API integration
- `demo_ai_api.ail` - AI API demonstration
- `demo_openai_api.ail` - OpenAI API integration

**Status**: Requires HTTP + JSON decoding (both implemented in v0.3.14), but needs higher-level HTTP client library and API key management.

### Concurrency (Future: v0.4.0+)
- `concurrent_pipeline.ail` - Concurrent data processing
- `web_api.ail` - HTTP server implementation

**Status**: Requires CSP channels and session types (planned for v0.4.0+)

### Advanced Features
- `factorial.ail` - May use advanced optimization features
- `quicksort.ail` - May use advanced list operations

**Status**: Core algorithms work, but may demonstrate future optimizations

## Timeline

| Feature | Current Status | Target Version | Estimated |
|---------|---------------|----------------|-----------|
| HTTP Client Library | Builtin exists | v0.3.15 | Q4 2025 |
| AI Agent Integration | Design only | v0.3.16 | Q1 2026 |
| CSP Channels | Planned | v0.4.0 | Q2 2026 |
| Session Types | Planned | v0.4.0 | Q2 2026 |
| HTTP Server | Design only | v0.4.1 | Q3 2026 |

## Testing

**Do not run these examples** - they will fail with type errors, missing builtins, or unimplemented features.

For current working examples, see:
- `examples/runnable/` - Full working programs
- `examples/snippets/` - Documentation code snippets
- `examples/tests/` - Test cases

## Contributing

To propose new experimental examples:
1. Use realistic syntax based on design docs
2. Add comments explaining what features are needed
3. Include expected behavior in comments
4. Link to relevant design doc or issue

Example:
```ailang
-- experimental/my_feature.ail
-- Requires: Feature X (see design_docs/planned/feature_x.md)
-- Status: Not implemented
-- Target: v0.X.Y

module experimental/my_feature

-- Expected: This should work when Feature X ships
export func demo() -> string {
  -- Your example code here
  "result"
}
```
