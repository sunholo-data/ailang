# AILANG package authoring — start here

Use the installed toolchain first: `ailang version`, `ailang prompt` (read the
whole prompt, including contracts, testing and packages), `ailang docs --help`,
`ailang docs --all-functions <term>`, and `ailang pkg-docs --help` for dependency
documentation. Reproduce suspected limitations with a minimal `ailang check`
before adopting old workarounds. Use web sources for external protocols when needed.

## Structure and style

- Scaffold with `ailang init package --name vendor/name`. Names use underscores.
- Keep reusable logic in packages; keep application configuration in the demo.
- Separate pure domain logic from small effectful adapters. Prefer ADTs for
  closed states/errors, explicit Result/Option, and exported boundary types.
- Use `import ./module` within a package (module namespace, not filesystem);
  `import pkg/vendor/name/module` between packages. Import Ok/Err explicitly.
- Declare public modules in [exports].modules and the smallest [effects].max.
  Document behavior, examples, effects, limitations and validation in AGENT.md.

## Contracts, tests and effects

Add meaningful requires/ensures to pure functions: express actual invariants,
not `true` to satisfy a count. Test valid, invalid and boundary cases with native
inline `tests [...]`, `test "name" { ... }` blocks and property tests. Properties
are useful for codecs, round trips and normalization. Keep external integration
checks as additional evidence. Contract presence is not proof or test coverage.

Use explicit effect rows and `@limit` budgets where operation counts can be
bounded; document intentionally unbounded work. A package effect ceiling grants
no runtime authority. Keep secrets out of sources, fixtures and logs.

## Validate and report evidence

    ailang lock
    ailang check --package .
    ailang test --package .
    ailang test path/to/module.ail
    ailang pkg quality --strict .
    ailang publish --dry-run

`test --package` discovers *_test.ail; run source modules containing inline tests
explicitly too. A function named test_* alone is not a native test declaration.
Check actual totals and skipped tests: exit zero with no tests is not evidence.
Use `ailang verify --help` for bounded SMT verification and report proved,
counterexample, unknown and skipped outcomes separately.

`pkg quality` is an offline AST inventory, not a test runner, typechecker or
proof engine. It reports missing contracts on functions without declared effects,
unbudgeted explicit effects, missing native tests and missing AGENT.md. Strict
mode fails on gaps. It cannot infer effects, assess assertion quality or establish
coverage. Use `--json` for automation; record compiler/test/proof results separately.

## Dependencies and publication

Registry packages can be published. `ailang install vendor/name@latest` resolves
once to an exact version. Use path dependencies for local co-development and
regenerate their lock after moving/cloning; do not call local path locks portable.
Publish dependencies first, then dependents. `ailang publish` rewrites path deps
to registry versions in the tarball and restores the local manifest.

`ailang publish --dry-run` previews packaging (and configured smoke checks); it
returns before remote registry validation. It does not establish test coverage
or proof. After validation, use `ailang publish` when publication is authorized.
Never describe committed/pushed packages as published without a registry result.

## More local detail

The core checkout's .agents/skills/ailang-packages/SKILL.md includes manifest and
error references and scripts/validate_package.sh. Detailed local sources live in
docs/docs/guides/{packages.md,package-publishing.md,testing.md,contracts.mdx} and
docs/docs/reference/capability-budgets.mdx. This compact guide is embedded in the
binary and does not require a checkout or network.
