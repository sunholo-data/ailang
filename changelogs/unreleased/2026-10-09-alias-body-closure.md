### Fixed — imported aliases retain their defining module's types (#1614)

- Close nonrecursive nullary alias bodies, exported function schemes and constructor
  fields before publishing module interfaces. An unrelated imported `Item` or `Planet`
  no longer changes `Box.items` or `System.planets`; incompatible record shapes still
  fail type checking. Private local alias dependencies are included.
- Compile cache version v6 invalidates open v5 interfaces once. Applied parameterized
  heads, recursive references, nominal module identity and import ambiguity diagnostics
  remain deferred; residual local-name capture keeps `TC_TYPE_SHADOW_001`.
