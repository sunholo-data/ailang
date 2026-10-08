### Fixed — reserved-word parameter diagnostics
- Reserved parameter names now report `PAR_RESERVED_KEYWORD` at the offending word,
  with local recovery that preserves later parameters and declarations. Record-type
  fields explain keyword reservations while retaining `PAR_FIELD_NAME_EXPECTED`.
- Reservation-specific rename suggestions link to the restored reserved-keywords
  reference, including the handler and CSP syntax-freeze decisions. All existing
  keyword reservations and contextual testing identifiers remain unchanged.
