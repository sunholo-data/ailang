# AILANG Release Guide

Complete guide to releasing new versions of AILANG.

## Automated Release (Recommended)

### Using release-manager Skill

**When ready to release, invoke the `release-manager` skill with the version number.**

The release-manager skill handles:
- Pre-release verification (tests, linting, file sizes)
- Version updates in documentation
- Git tagging and pushing
- CI/CD monitoring
- Release verification

### Using post-release Skill

**After release completes, use the `post-release` skill:**

The post-release skill handles:
- Run baseline evaluations
- **Update website dashboard** (critical!)
- Update design docs and public documentation
- Extract metrics for CHANGELOG

## Manual Release Process

If you need to perform a release manually (not recommended):

### 1. Pre-Release Checks

```bash
# Run all tests
make test

# Run linting
make lint

# Check file sizes
make check-file-sizes

# Verify examples
make verify-examples
```

### 2. Update Version Numbers

Update version in:
- `CHANGELOG.md` - Add new version section
- `README.md` - Update version badges
- Any version constants in code

### 3. Create Git Tag

```bash
# Create annotated tag
git tag -a v0.x.x -m "Release v0.x.x"

# Push tag to remote
git push origin v0.x.x
```

### 4. Monitor CI/CD

Watch GitHub Actions to ensure:
- Tests pass
- Builds complete
- Artifacts are generated

### 5. Verify Release

```bash
# Check GitHub release was created
gh release view v0.x.x

# Verify artifacts are attached
gh release view v0.x.x --json assets
```

## Manual Dashboard Update

**Update website dashboard for specific version:**

```bash
# Generate dashboard files (markdown + JSON with history)
# Note: 2>/dev/null suppresses progress messages that would appear in the markdown
ailang eval-report eval_results/baselines/v0.3.12 v0.3.12 --format=docusaurus 2>/dev/null > docs/docs/benchmarks/performance.md

# DO NOT redirect JSON to file - it writes to docs/static/benchmarks/latest.json automatically with history preservation
ailang eval-report eval_results/baselines/v0.3.12 v0.3.12 --format=json

# Verify JSON is valid
jq -r '.version, .aggregates.finalSuccess' docs/static/benchmarks/latest.json

# Clear Docusaurus cache (prevents webpack errors)
cd docs && npm run clear

# Test locally (optional)
cd docs && npm start
# Visit: http://localhost:3000/ailang/docs/benchmarks/performance

# Commit and push
git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
git commit -m "Update benchmark dashboard for v0.3.12"
git push
```

## Common Issues

### Problem: Dashboard shows old version

**Symptom**: Dashboard shows v0.3.9 instead of v0.3.12

**Solution**: Use `ailang eval-report` with specific baseline directory
```bash
ailang eval-report eval_results/baselines/v0.3.12 v0.3.12 --format=json
```

### Problem: "Uncaught runtime errors" / webpack chunk errors

**Cause**: Docusaurus build cache stale

**Solution**: Clear cache and rebuild
```bash
cd docs && npm run clear && rm -rf docs/.docusaurus docs/build && npm start
```

### Problem: Dashboard JSON shows "null" for aggregates

**Cause**: Used wrong JSON file (performance matrix vs dashboard JSON)

**Solution**: Use `ailang eval-report` output, not files from `eval_results/performance_tables/`

### Problem: CI/CD failures after tag push

**Symptom**: GitHub Actions fail after pushing tag

**Common causes:**
1. Tests failing that passed locally
2. Linting errors
3. File size violations
4. Missing dependencies

**Solution**: Check GitHub Actions logs, fix issues, delete tag, and re-tag:
```bash
# Delete local tag
git tag -d v0.x.x

# Delete remote tag
git push origin :refs/tags/v0.x.x

# Fix issues, then re-tag
git tag -a v0.x.x -m "Release v0.x.x"
git push origin v0.x.x
```

## Release Checklist

Use this checklist for manual releases:

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] File sizes OK (`make check-file-sizes`)
- [ ] Examples verified (`make verify-examples`)
- [ ] CHANGELOG.md updated
- [ ] README.md updated (if needed)
- [ ] Version numbers updated
- [ ] Git tag created
- [ ] Tag pushed to remote
- [ ] CI/CD passed
- [ ] GitHub release created
- [ ] Artifacts attached
- [ ] Eval baseline run
- [ ] Dashboard updated
- [ ] Design docs moved to implemented/

## Semantic Versioning

AILANG follows semantic versioning (SemVer):

- **MAJOR** (v1.0.0 → v2.0.0): Breaking changes
- **MINOR** (v0.3.0 → v0.4.0): New features, backward compatible
- **PATCH** (v0.3.12 → v0.3.13): Bug fixes, backward compatible

**Current status**: Pre-1.0 (v0.x.x), so minor versions may include breaking changes.

## Post-Release Tasks

After releasing:

1. **Announce release**
   - Update docs site
   - Social media (if applicable)
   - Community channels

2. **Monitor for issues**
   - Watch GitHub issues
   - Monitor CI/CD
   - Check for user reports

3. **Plan next release**
   - Review remaining TODOs
   - Prioritize next features
   - Update roadmap

## See Also

- **release-manager skill** - Automated release workflow
- **post-release skill** - Post-release automation
- **CHANGELOG.md** - Version history
- **README.md** - Project status
