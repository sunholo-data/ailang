### Fixed — IFC: a library's sinks and labels now bind its importers (#1134, BREAKING)

- A `{not ℓ}` sink refinement declared in one module is enforced at call sites in every module
  that imports it. Before, `gate(body: string{not email})` rejected an `<email>` argument only
  when the caller lived in `gate`'s own module; moved one file away, the same leak compiled
  clean. The error names the edge: `value labelled <email> reaches parameter "body" of imported
  function lib.gate (via gate) which forbids it (string{not email})`, at the caller's position.
- The same holds for positive parameter labels (`param_label_cover`, Check C), for imported
  return labels (`-> string<email>` taints the importer's value) and for undeclared intrinsic
  taint (a library wrapper around `secret()` taints its result, including through a second
  library). Imported `! {Declassify}` / `! {Declassify[label=ℓ]}` functions relabel exactly as
  they do locally.
- Every way of naming the import resolves to the same contract: `import lib (gate)`,
  `gate as g`, `import lib as L` then `L.gate`, aliases, higher-order arguments and wrapper
  lambdas. Lexical shadowing is respected. The CLI and the browser REPL (`LoadModule`) give the
  same verdicts; a rejected REPL module publishes nothing.
- An interface whose IFC metadata is missing or malformed is refused, never read as clean.
- Checks A and C are structural over the parameter type, in every module (a pre-existing hole,
  found migrating email-parse for this change). `takeList(xs: [string<sqlsafe>])` accepted `[s]`
  for an `<arg>` value `s` while `takeOne(x: string<sqlsafe>)` rejected `s`. A `<ℓ>` or
  `{not ℓ}` written in a covariant data position — list/array element, tuple element, record
  field (inline or a declared record/alias), type-constructor argument (`Option`, `Result`, a
  local generic), ADT constructor field — is now a demand on the argument, and the exported
  summary carries it to importers. Values carry one flat label, so the bound is per parameter:
  each root-to-leaf path admits the label written along it (a path naming two different labels
  admits only ⊥), and the argument's label must be admitted by some labelled path. Labels inside
  a function-typed parameter stay contravariant: the callee owes them when it applies the
  function. Not yet bounded: a function-typed parameter's return label (follow-up).
- **Breaking (structural Check A/C):** a list, record or tuple argument carrying a label its
  parameter type does not declare now fails with `param_label_cover`; so does an argument whose
  label comes from an imported higher-order function's conservative result (`mapE(f, rows)` is
  labelled with `rows` too).
- **Breaking:** programs that passed labelled values into imported sinks now fail `ailang check`.
  Route the value through a declassifier, or stop passing it.
- Compile cache epoch `v9` → `v10`: every cached module recompiles once, because a cache hit skips
  the IFC gate and a v9 caller verdict was reached without imported contracts.
- Not changed: requirements the checker *infers* for unannotated parameters (#1719) stay within
  their module. An exported `relay(x: string) = gate(x)` does not carry `{not ℓ}` to importers;
  annotate the exported parameter to make it part of the contract.
- New examples: `examples/runnable/secrets/ifc_xmod_{lib,app,leak}.ail`; guide section
  "Imported sinks and sources" in `docs/docs/guides/ifc-labels.mdx`.
