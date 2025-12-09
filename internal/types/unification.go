package types

// unification.go - Core unification entry point and utilities
// This file contains the main Unifier type and primary Unify method.
// The actual unification logic is split across specialized files:
//   - unification_core.go: Core Unifier type, initialization, main Unify dispatcher
//   - unification_types.go: Type-specific unification (functions, lists, arrays, tuples, type apps)
//   - unification_records.go: Record type unification (TRecord, TRecord2, TRecordOpen, rows)
//   - unification_occurs.go: Occurs check implementation with cycle detection
//   - unification_equality.go: Safe equality checking with cycle detection
//   - unification_substitution.go: Substitution application with cycle detection
