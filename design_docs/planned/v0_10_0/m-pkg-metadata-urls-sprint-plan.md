# Sprint Plan: M-PKG-METADATA-URLS

## Summary
Add repository, homepage, and license_url fields to package metadata. Propagate through validator → registry index → API → website + CLI.

**Duration:** 3 milestones, ~50 LOC, half a day
**Risk Level:** Low (additive, all fields optional, backward compatible)

## Milestones

### M1: Struct + Validator Propagation
**Estimated:** ~20 LOC
- Add fields to `MetadataManifest` and `IndexEntry` structs
- Extract URL fields in `handlePublish`
- Copy URL fields in `updateIndex`, `tryUpdateIndex`, `handleRebuildIndex`

**Acceptance Criteria:**
- [ ] MetadataManifest has Repository, Homepage, LicenseURL fields (omitempty)
- [ ] IndexEntry has same fields
- [ ] Validator extracts from manifest metadata on publish
- [ ] Index update copies fields to index entry
- [ ] Existing packages without URLs still work (backward compat)

### M2: CLI + Website Display
**Estimated:** ~25 LOC
- Show URLs in `ailang pkg info` output
- Show source/docs/license links in PackageDetail component
- Show source link in VendorIndex cards

**Acceptance Criteria:**
- [ ] `ailang pkg info` shows Repository/Homepage/License when present
- [ ] `ailang pkg info` omits URL lines when absent
- [ ] Website PackageDetail shows clickable links
- [ ] Build passes

### M3: Update ailang-packages Manifests
**Estimated:** ~5 LOC per package (15 packages)
- Add repository, homepage, license_url to all 15 ailang.toml files
- Republish triggers auto-deploy via Cloud Build

**Acceptance Criteria:**
- [ ] All 15 ailang.toml files have URL fields
- [ ] After republish, `ailang pkg info sunholo/auth` shows URLs
- [ ] Website package pages show source links
