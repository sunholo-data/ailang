### Fixed — `import std/map (Map)` and `import std/array (Array)` no longer fail with IMP010

`Map` and `Array` are builtin types, available in every module without an import. Neither
`std/map` nor `std/array` exports a type of that name, so importing one failed with
`IMP010: symbol 'Map' not exported by 'std/map'`.

Since 2026-10-28 (prompt v0.16.8) the prompt has taught `import std/set (Set)`, which is needed to
write `Set[a]`, on the lines next to `import std/map as M`. `std/option` exports `Option` the same
way. Models now import `Map` and `Array` by analogy. In the Gemini 4 Hydrogen placement this caused
4 of its 5 failures; 3.8 Flash hit it twice.

Importing a builtin structural type from the module that owns it is now accepted and has no
effect. The ownership table is `types.IsStructuralTypeOf`, read by `loader.getExport`. Importing it
from any other module, such as `import std/list (Map)`, still fails with IMP010.
