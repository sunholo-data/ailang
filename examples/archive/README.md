# Archived Examples

This folder contains examples that have been archived because they:
- Were duplicates of better examples elsewhere
- Used outdated syntax from earlier AILANG versions
- Were debug/test files not suitable for documentation
- Had type errors due to stdlib changes

## Contents

### broken/
Examples with type errors or parse errors that need investigation:
- `testing_advanced.ail` - Validation failures
- `testing_basic.ail` - Validation failures
- `nested_records.ail` - Type errors
- `sem_frame_test.ail` - stdlib type mismatch

### debug/
Development debugging files:
- `debug_let.ail`, `debug_let2.ail`, `debug_types_demo.ail`

### nested_match_variants/
Redundant pattern matching examples (canonical versions kept in runnable/):
- `nested_match_debug.ail`
- `nested_match_no_comment.ail`
- `nested_match_ai_generated.ail`
- `nested_match_comprehensive.ail`
- `nested_match_three_levels.ail`
- `nested_match_with_io.ail`

## Recovery

If you need to restore an archived example:
1. Check the error with `ailang check --relax-modules examples/archive/broken/file.ail`
2. Fix the issue (effect declarations, type annotations, etc.)
3. Move back to appropriate folder

## Canonical Examples

For working pattern matching examples, see:
- `runnable/nested_match_simple.ail`
- `runnable/nested_match_minimal.ail`
- `runnable/patterns.ail`
