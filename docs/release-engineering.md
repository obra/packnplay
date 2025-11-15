# Release Engineering Process

This document outlines the systematic process for releasing new versions of packnplay.

## Overview

Our release process ensures:
- **Consistent versioning** using semantic versioning (semver)
- **Complete documentation** of changes for users
- **Proper tagging** for tracking and rollback
- **Automated verification** that releases work

## Release Types

### Patch Release (x.y.Z)
- Bug fixes
- Documentation updates
- Performance improvements
- No breaking changes

### Minor Release (x.Y.0)
- New features
- New command-line options
- New capabilities
- Backward compatible

### Major Release (X.0.0)
- Breaking changes
- API changes
- Incompatible CLI changes
- Architecture changes

## Pre-Release Checklist

Before starting a release:

- [ ] All tests pass: `go test ./...`
- [ ] Code builds successfully: `go build`
- [ ] No pending critical bugs
- [ ] Documentation is up to date
- [ ] All intended features are complete

## Release Process

### 1. Determine Version Number

Follow semantic versioning:
- Look at changes since last release
- Determine if patch, minor, or major release
- Check current version: `git describe --tags --abbrev=0`

### 2. Update Version Information

**Update version.go (if exists) or main.go:**
```go
const Version = "v1.2.3"
```

**Or create version.go:**
```go
package main

const Version = "v1.2.3"
```

### 3. Update Documentation

**Update README.md:**
- Add new features to feature list
- Update installation instructions if needed
- Update usage examples with new options
- Verify all links work

**Create/update CHANGELOG.md:**
```markdown
# Changelog

## [v1.2.3] - 2024-MM-DD

### Added
- New feature descriptions

### Changed
- Modified behavior descriptions

### Fixed
- Bug fix descriptions

### Removed
- Deprecated feature removals
```

### 4. Create Release Notes

**Create docs/releases/vX.Y.Z.md:**
```markdown
# Release v1.2.3

## What's New

Brief summary of major changes and why users should upgrade.

## Features Added

- Feature 1: Description and usage example
- Feature 2: Description and usage example

## Bug Fixes

- Fix 1: What was broken and how it's fixed
- Fix 2: Impact and resolution

## Breaking Changes

(Only for major releases)
- Change 1: What changed and migration steps
- Change 2: Impact and required actions

## Installation

```bash
# Installation instructions
```

## Usage Examples

```bash
# Examples of new features
```

## Thanks

Recognition for contributors, issue reporters, etc.
```

### 5. Test the Release

**Build and test:**
```bash
go build -o packnplay
./packnplay --version
./packnplay --help
# Test major functionality
```

**Run comprehensive tests:**
```bash
go test ./... -v
```

### Version Information System

packnplay uses build-time variable injection to provide accurate version information:

**Components:**
- **Version**: From `git describe --tags --always` (e.g., `v1.1.0-89-g8a39345`)
- **Commit**: From `git rev-parse HEAD` (full commit hash)
- **Build Date**: UTC timestamp in ISO format

**Build Methods:**
- **Development builds**: `make build` - Uses git-derived version
- **Release builds**: GoReleaser - Uses tag-based version with {{.Version}}
- **Manual builds**: `go build` - Shows default values ("dev", "none", "unknown")

**Verification:**
Always verify version information after builds:
```bash
./packnplay version
# Should show: packnplay v1.1.0-89-g8a39345
#              commit: 8a393453a45aa38bc...
#              built:  2025-11-15T17:19:46Z
```

### 6. Commit Release Changes

```bash
git add .
git commit -m "chore: prepare release v1.2.3

- Update version to v1.2.3
- Update README with new features
- Add release notes for v1.2.3
- Update CHANGELOG
"
```

### 7. Create Git Tag

```bash
git tag -a v1.2.3 -m "Release v1.2.3

Brief description of what's in this release.
See docs/releases/v1.2.3.md for full release notes."
```

### 8. Push Release

```bash
git push origin main
git push origin v1.2.3
```

### 9. Verify Release

**Check that:**
- [ ] Tag appears on GitHub
- [ ] Release notes are accessible
- [ ] Installation instructions work
- [ ] Major features work as documented

## Post-Release Tasks

### Update Documentation Sites

If documentation is hosted elsewhere:
- [ ] Update project website
- [ ] Update package manager entries
- [ ] Update Docker images (if applicable)

### Announce Release

Consider announcing on:
- [ ] Project Discord/Slack
- [ ] Twitter/social media
- [ ] Internal team channels
- [ ] Relevant forums/communities

## Hotfix Process

For critical bugs in released versions:

1. **Create hotfix branch:** `git checkout -b hotfix/v1.2.4 v1.2.3`
2. **Fix the issue** with minimal changes
3. **Test thoroughly**
4. **Follow release process** for patch version
5. **Merge back to main:** `git checkout main && git merge hotfix/v1.2.4`

## Version History

Track versions and their significance:

- **v1.0.0**: Initial stable release
- **v1.1.0**: Added user detection system
- **v1.2.0**: Added port mapping support
- **v1.2.1**: Bug fixes for port mapping

## Emergency Procedures

### Rollback a Release

If a release has critical issues:

```bash
# Revert the tag
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3

# Create communication about the issue
# Fix and re-release as v1.2.4
```

### Security Release

For security issues:
1. **Don't discuss publicly** until fixed
2. **Create patch quickly**
3. **Follow expedited release process**
4. **Coordinate disclosure** with security team

## Tools and Automation

Consider these tools for automation:
- **GitHub Actions** for CI/CD
- **Semantic-release** for automated versioning
- **Conventional commits** for automatic changelog generation

## Release Engineering Checklist Template

Copy this for each release:

```markdown
## Release vX.Y.Z Checklist

### Pre-Release
- [ ] All tests pass
- [ ] Version determined (patch/minor/major)
- [ ] Version updated in code
- [ ] README updated
- [ ] CHANGELOG updated
- [ ] Release notes written

### Release
- [ ] Changes committed
- [ ] Tag created
- [ ] Tag pushed
- [ ] Release verified

### Post-Release
- [ ] Documentation updated
- [ ] Release announced
- [ ] Next version planning started
```

## Contact

For questions about the release process:
- Check this documentation first
- Ask in team channels
- Review previous releases as examples