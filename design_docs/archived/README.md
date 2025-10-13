# Archived Design Documents

This directory contains design documents that are no longer relevant or have been superseded by implementation reality.

## Documents in this directory:

### 20251008_script_mode_OBSOLETE.md
**Original Title**: Script Mode and Lenient Entry Discovery
**Archived Date**: 2025-10-13
**Reason**: Problem no longer exists. AI models successfully learned to generate proper AILANG syntax with `module` declarations. The eval harness shows 38.9% success rate with AIs correctly using:
- `module benchmark/solution` declarations
- `export func main()` entry points
- Proper AILANG syntax

The failures in benchmarks like `cli_args` are due to AIs generating completely wrong syntax (Python, pseudocode, BASIC), not due to missing script mode support.

**Evidence**:
- Current eval results: 35/90 passing (38.9%)
- All passing benchmarks use proper `module` + `export func main()` structure
- Entry point discovery works fine with `--entry main` (default)
- No evidence of users or AIs expecting "script mode" behavior

**Implementation Decision**: WONTFIX - Module declarations and explicit entry points are a design feature, not a bug. The prompt engineering approach (teaching AIs correct syntax) is working as intended.
