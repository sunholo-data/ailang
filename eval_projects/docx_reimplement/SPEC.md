# Task: reimplement docparse/services/docx_parser.ail (large-context AILANG)

**Tier:** hard / large-context (P0 frontier instrument)
**Repo:** a copy of sunholo-data/ailang-parse. The file `docparse/services/docx_parser.ail` (~530 lines)
has been stubbed. Reimplement it so the document parser reproduces the correct output on all DOCX fixtures.

## What docx_parser does
Converts DOCX XML into the `Block` ADT: paragraphs (style/text/runs), tables (merged cells via gridSpan/
vMerge), track changes (insert/delete/move), headers/footers/footnotes/endnotes, comments, images, nested
SDTs. Exports `parseDocx(filepath) -> [Block] ! {FS}` plus `parseDocxMetadata/Headers/Footers/Footnotes/
Endnotes/Images/Comments`. It depends on (READ THESE — large context):
- `docparse/types/document` (the Block ADT + constructors),
- `docparse/services/zip_extract` (readDocxContent, media/header/footnote entry helpers),
- `std/xml` (parse, findAll, findFirst, getText, getAttr, getChildren, getTag).

## Pass criteria
`ailang run --entry main --caps IO,FS,Env docparse/main.ail data/test_files/<f>.docx` reproduces the
expected content blocks + document summary for every fixture in `data/test_files/*.docx`.
Grade: `eval_projects/docx_reimplement/verify.sh <repo-dir>` → 17/17 fixtures pass (diff vs golden).

## Why this is the P0 instrument
The standard single-file benchmark set is saturated (motoko = pi at 96.9% / 100% best-of-N). This task is
LARGE-CONTEXT (must read the Block ADT + zip_extract + xml usage across the package) and LONG (530 lines
of XML tree-walking with edge cases: merged cells, track changes, footnotes) — the regime where harness
context-management actually diverges. Deterministic golden over 17 real DOCX fixtures.
