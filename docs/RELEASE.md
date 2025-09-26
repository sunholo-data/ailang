# AILANG Release Process

This document describes the release process for AILANG, including CI/CD workflows, versioning strategy, and distribution methods.

## Table of Contents
- [Semantic Versioning](#semantic-versioning)
- [GitHub Actions Workflows](#github-actions-workflows)
- [Creating a Release](#creating-a-release)
- [Binary Distribution](#binary-distribution)
- [Local Development](#local-development)

## Semantic Versioning

AILANG follows [Semantic Versioning](https://semver.org/) (SemVer):
- **MAJOR.MINOR.PATCH** (e.g., `v1.2.3`)
- **MAJOR**: Breaking changes to the language or API
- **MINOR**: New features, backwards compatible
- **PATCH**: Bug fixes, backwards compatible

### Version Management

Version information is managed entirely through **git tags** - no manual file updates required:

1. **Tagged Releases**: Uses exact tag (e.g., `v0.1.0`)
2. **Development Builds**: Uses `git describe` format (e.g., `v0.1.0-5-g1234567`)
3. **Dirty Builds**: Appends `-dirty` for uncommitted changes

Version info embedded in binaries includes:
- Version string (from git tag)
- Git commit hash
- Build timestamp

## GitHub Actions Workflows

### Build Workflow (`.github/workflows/build.yml`)

**Triggers:**
- Push to `main` or `dev` branches
- Pull requests to `main` or `dev`
- Manual workflow dispatch

**Actions:**
1. Runs tests across all platforms
2. Builds binaries for multiple platforms:
   - `ailang-linux-amd64` - Linux x86_64
   - `ailang-darwin-amd64` - macOS Intel
   - `ailang-darwin-arm64` - macOS Apple Silicon
   - `ailang-windows-amd64` - Windows x86_64
3. Uploads artifacts (retained for 30 days)
4. Creates a combined bundle of all platforms

### Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Push of tags matching `v*` pattern (e.g., `v0.1.0`, `v2.0.0-beta.1`)

**Actions:**
1. Builds release binaries with version info
2. Creates GitHub Release
3. Attaches platform binaries to release
4. Generates changelog from commits
5. Adds installation instructions to release notes

## Creating a Release

### 1. Prepare Your Release

```bash
# Ensure you're on the main branch with latest changes
git checkout main
git pull origin main

# Run tests locally
make test

# Build and verify version locally
make build
./bin/ailang --version
```

### 2. Create and Push Version Tag

```bash
# For a new patch release (bug fixes)
git tag v0.1.1

# For a new minor release (new features)
git tag v0.2.0

# For a new major release (breaking changes)
git tag v1.0.0

# For pre-releases
git tag v0.2.0-beta.1

# Push the tag to trigger release workflow
git push origin v0.1.1
```

### 3. Monitor Release Creation

1. Go to the [Actions tab](../../actions) in GitHub
2. Watch the "Release" workflow run
3. Once complete, check the [Releases page](../../releases)

### 4. Release Notes

The release workflow automatically:
- Generates a changelog from commits since last tag
- Includes installation instructions for each platform
- Marks pre-releases appropriately (if version contains `-`)

You can edit the release notes after creation if needed.

## Binary Distribution

### Download Methods

#### From GitHub Releases

Users can download binaries directly from the [Releases page](../../releases).

#### Using curl/wget

The release notes include platform-specific installation commands:

```bash
# macOS Intel
curl -L https://github.com/sunholo/ailang/releases/download/v0.1.0/ailang-darwin-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# macOS Apple Silicon
curl -L https://github.com/sunholo/ailang/releases/download/v0.1.0/ailang-darwin-arm64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Linux
curl -L https://github.com/sunholo/ailang/releases/download/v0.1.0/ailang-linux-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/sunholo/ailang/releases/download/v0.1.0/ailang-windows-amd64.zip -OutFile ailang.zip
Expand-Archive ailang.zip -DestinationPath .
```

### Build Artifacts

Every push to `main` or `dev` creates build artifacts:
- Available in GitHub Actions for 30 days
- Useful for testing pre-release versions
- Accessible from the Actions tab

## Local Development

### Building with Version Info

```bash
# Build with automatic version detection
make build

# Version will be:
# - Git tag if on a tagged commit
# - Git describe output if after a tag
# - "0.1.0-dev" if no tags exist
```

### Installing Locally

```bash
# Install to $GOPATH/bin with version info
make install

# Quick install (no version info)
make quick-install
```

### Testing Version Output

```bash
# Check version after building
./bin/ailang --version

# Output example for tagged release:
# AILANG v0.1.0
# Commit: abc1234
# Built:  2025-01-26_10:30:00

# Output example for development:
# AILANG v0.1.0-5-gabc1234-dirty
# Commit: abc1234
# Built:  2025-01-26_10:30:00
```

## Troubleshooting

### Tag Not Triggering Release

Ensure your tag follows the `v*` pattern:
```bash
# Good
git tag v0.1.0
git tag v2.0.0-beta.1

# Bad (won't trigger release)
git tag 0.1.0      # Missing 'v' prefix
git tag version1   # Wrong format
```

### Version Shows "dev" or "unknown"

This happens when:
1. No git repository (building outside repo)
2. No tags exist yet
3. Building in CI without full git history

Solution: Ensure git history is available and tags are fetched:
```bash
git fetch --tags
make build
```

### Release Workflow Fails

Common issues:
1. **Permissions**: Ensure the workflow has write permissions for releases
2. **Tag format**: Must match `v*` pattern
3. **Build failures**: Check test results in workflow logs

## Future Improvements

Planned enhancements:
- [ ] Homebrew formula for macOS
- [ ] APT/YUM repositories for Linux
- [ ] Chocolatey package for Windows
- [ ] Docker images for each release
- [ ] Automatic changelog categorization
- [ ] Release signing with GPG