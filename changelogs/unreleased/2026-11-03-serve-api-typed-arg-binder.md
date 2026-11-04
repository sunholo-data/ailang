### Fixed

- serve-api now binds JSON arguments by declared parameter types on MCP, REST, A2A and UI resource calls. Integer parameters accept integral numbers, float parameters preserve their declared type, and nested records and lists use the same conversion. Fractional or out-of-range integers and scalar type mismatches produce argument errors before evaluation. Omitted optional integer parameters bind zero.
