# Release Process

This document describes how to create a new release of packnplay.

## Prerequisites

- Write access to the `obra/packnplay` repository
- Write access to the `obra/homebrew-tap` repository (created automatically)
- GitHub CLI (`gh`) authenticated

## Release Steps

### 1. Prepare the Release

Ensure all changes for the release are merged to `main`:

```bash
git checkout main
git pull origin main
```

### 1.5. Verify Version System (Before Release)

Before creating the release tag, verify the version system works correctly:

```bash
# Build locally to test version injection
make build
./packnplay version

# Should show development version like: v1.1.0-89-g8a39345
# Commit hash should match: git rev-parse HEAD
# Build date should be recent
```

**Expected Output:**
```
packnplay v1.1.0-89-g8a39345
  commit: 8a393453a45aa38bc5cc0ca60c7560fabdabc13d
  built:  2025-11-15T17:19:46Z
```

If version shows default values ("dev", "none", "unknown"), check:
1. Git repository has commits and tags
2. Makefile LDFLAGS are correctly configured
3. Build is using `make build` not `go build`

### 2. Update Version and Changelog

Update `CHANGELOG.md` with release notes following the existing format:

```bash
# Edit CHANGELOG.md to add new version section
vim CHANGELOG.md

# Commit the changes
git add CHANGELOG.md
git commit -m "chore: prepare release vX.Y.Z"
git push origin main
```

### 3. Create and Push a Tag

Create a version tag following semantic versioning (vMAJOR.MINOR.PATCH):

```bash
# Create annotated tag
git tag -a v1.0.1 -m "Release v1.0.1"

# Push the tag to trigger release workflow
git push origin v1.0.1
```

### 4. Monitor the Release

The GitHub Actions workflow will automatically:

1. Build binaries for all platforms:
   - `linux/amd64`
   - `linux/arm64`
   - `darwin/amd64` (Intel Macs)
   - `darwin/arm64` (Apple Silicon)

2. Create archives and checksums

3. Create a GitHub Release with all artifacts

4. Update the Homebrew formula in `obra/homebrew-tap`

Monitor progress at: https://github.com/obra/packnplay/actions

### 5. Verify the Release

Once the workflow completes:

#### Check GitHub Release

Visit https://github.com/obra/packnplay/releases and verify:
- All 4 platform binaries are attached
- Checksums file is present
- Release notes are generated from commits

#### Test Homebrew Installation

```bash
# Update Homebrew
brew update

# Install or upgrade packnplay
brew install obra/tap/packnplay
# or if already installed:
brew upgrade obra/tap/packnplay

# Verify version
packnplay version
```

#### Test Binary Downloads

Download and test a binary directly:

```bash
# Example for macOS ARM64
curl -LO https://github.com/obra/packnplay/releases/download/v1.0.1/packnplay_1.0.1_Darwin_arm64.tar.gz

# Extract
tar xzf packnplay_1.0.1_Darwin_arm64.tar.gz

# Run
./packnplay --version
```

## Troubleshooting

### Release Workflow Fails

1. Check the Actions log: https://github.com/obra/packnplay/actions
2. Common issues:
   - Tests failing (fix and re-tag)
   - GoReleaser configuration errors (fix `.goreleaser.yml`)
   - Missing permissions (check workflow permissions)

### Homebrew Formula Not Updated

1. Check if `obra/homebrew-tap` repository exists
2. Verify GitHub token has write access to the tap repository
3. Check GoReleaser logs in Actions for Homebrew-specific errors

### Need to Re-release

If you need to fix a release:

```bash
# Delete the tag locally and remotely
git tag -d v1.0.1
git push origin :refs/tags/v1.0.1

# Delete the GitHub release
gh release delete v1.0.1

# Make your fixes, then re-tag
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (v2.0.0): Breaking changes
- **MINOR** (v1.1.0): New features, backwards compatible
- **PATCH** (v1.0.1): Bug fixes, backwards compatible

## Release Checklist

- [ ] All changes merged to `main`
- [ ] `CHANGELOG.md` updated with release notes
- [ ] Version tag created and pushed
- [ ] GitHub Actions workflow completed successfully
- [ ] GitHub Release created with all artifacts
- [ ] Homebrew formula updated in tap repository
- [ ] Tested installation via Homebrew
- [ ] Tested binary download and execution
- [ ] Announced release (if applicable)
