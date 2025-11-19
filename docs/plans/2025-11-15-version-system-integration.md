# Version System Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the new Makefile version injection system with documentation, CI autobuilder, and Homebrew tap to ensure consistent version information across all distribution channels.

**Architecture:** Update documentation to reflect new version system, verify CI workflows work correctly with Makefile changes, and ensure GoReleaser and Homebrew tap continue working properly with the enhanced version injection.

**Tech Stack:** Makefile, GoReleaser, GitHub Actions, Homebrew Formula, Markdown documentation

---

## Task 1: Update Documentation for Version System

**Files:**
- Modify: `README.md:40-60` (Installation section)
- Modify: `docs/release-engineering.md:150-180` (Version determination section)
- Modify: `docs/release-process.md:40-80` (Prerequisites and version steps)

**Step 1: Update README.md installation section**

Update the build from source instructions to reflect the new version system:

```markdown
### Build from Source

**Prerequisites:**
- Go 1.21 or later
- Git (for version information)

**Build with version information:**
```bash
git clone https://github.com/obra/packnplay.git
cd packnplay
make build
./packnplay version
```

**Quick install without Makefile:**
```bash
go install github.com/obra/packnplay@latest
```

Note: Building with `make build` includes proper version, commit, and build date information. Direct `go build` or `go install` will show default values.
```

**Step 2: Update docs/release-engineering.md**

Add section about version injection system:

```markdown
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
```

**Step 3: Update docs/release-process.md**

Add verification step for version system:

```markdown
### Step 1.5: Verify Version System (Before Release)

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
```

**Step 4: Run documentation test**

Test that all documentation references are accurate:

```bash
# Verify README instructions work
make clean
make build
./packnplay version
```

Expected: Version information appears correctly, not default values.

**Step 5: Commit documentation updates**

```bash
git add README.md docs/release-engineering.md docs/release-process.md
git commit -m "docs: update version system documentation

- Add build instructions that preserve version info
- Document version injection system architecture
- Add verification steps for release process"
```

---

## Task 2: Verify CI/CD Integration

**Files:**
- Check: `.github/workflows/ci.yml`
- Check: `.github/workflows/release.yml`
- Check: `.goreleaser.yml`
- Test: CI pipeline behavior

**Step 1: Review CI pipeline compatibility**

Examine current CI configuration for compatibility with Makefile changes:

```bash
# Check that CI uses make targets that include version injection
grep -n "make build\|go build" .github/workflows/ci.yml
```

**Step 2: Verify GoReleaser configuration**

Check that GoReleaser uses consistent version injection:

```yaml
# Verify .goreleaser.yml has correct ldflags
builds:
  - main: ./main.go
    ldflags:
      - -X github.com/obra/packnplay/cmd.version={{.Version}}
      - -X github.com/obra/packnplay/cmd.commit={{.Commit}}
      - -X github.com/obra/packnplay/cmd.date={{.Date}}
```

**Step 3: Test local GoReleaser build**

Verify GoReleaser works with new Makefile:

```bash
# Install goreleaser if not present
# Test snapshot build (doesn't require tag)
goreleaser build --snapshot --clean --single-target

# Check version in built binary
./dist/packnplay_darwin_amd64_v1/packnplay version
```

Expected: Shows GoReleaser-injected version information.

**Step 4: Verify release workflow test**

Check that release workflow's version test still works:

```yaml
# From .github/workflows/release.yml - verify this step:
test: |
  system "#{bin}/packnplay", "--version"
```

**Step 5: Document CI verification**

Add verification note to release docs:

```bash
git add -A
git commit -m "ci: verify GoReleaser compatibility with Makefile version system"
```

---

## Task 3: Test Homebrew Formula Integration

**Files:**
- Test: Homebrew formula generation
- Check: `obra/homebrew-tap` repository (external)
- Verify: Version command in formula test

**Step 1: Create test release tag**

Create a test tag to verify full release pipeline:

```bash
# Create lightweight test tag (can be deleted later)
git tag v1.1.1-test
git push origin v1.1.1-test
```

**Step 2: Monitor release workflow**

Watch GitHub Actions for the release workflow:

```bash
# Check workflow status
gh workflow list
gh run list --workflow=release.yml
```

**Step 3: Verify Homebrew formula generation**

Check that formula is created correctly in homebrew-tap:

```bash
# View the generated formula (after workflow completes)
curl -s https://raw.githubusercontent.com/obra/homebrew-tap/main/Formula/packnplay.rb
```

Verify formula contains correct version reference and test:

```ruby
test do
  system "#{bin}/packnplay", "--version"
end
```

**Step 4: Test Homebrew installation (if applicable)**

If safe to test with test version:

```bash
# Add tap and install test version
brew tap obra/tap
brew install packnplay

# Verify version command works
packnplay version
```

Expected: Shows proper version information from release build.

**Step 5: Clean up test tag**

```bash
# Remove test tag after verification
git tag -d v1.1.1-test
git push origin :refs/tags/v1.1.1-test
```

**Step 6: Document Homebrew verification**

```bash
git add docs/release-process.md
git commit -m "docs: add Homebrew formula verification to release process"
```

---

## Task 4: Update Build Documentation

**Files:**
- Create: `docs/building.md`
- Modify: `Makefile` (add documentation target)

**Step 1: Create comprehensive build documentation**

```markdown
# Building packnplay

## Quick Start

**Standard build with version information:**
```bash
make build
./packnplay version
```

**Development install:**
```bash
make install
packnplay version
```

## Version Information System

### How It Works

packnplay embeds version information at build time using Go's `-ldflags` feature:

- **Version**: `git describe --tags --always` - Git tag with commit offset
- **Commit**: `git rev-parse HEAD` - Full commit hash
- **Build Date**: `date -u +%Y-%m-%dT%H:%M:%SZ` - UTC timestamp

### Build Methods

| Method | Version Info | Use Case |
|--------|-------------|----------|
| `make build` | ✅ Full | Local development, testing |
| `make install` | ✅ Full | Install to GOPATH/bin |
| `go build` | ❌ Default | Quick builds, doesn't inject version |
| `go install github.com/obra/packnplay@latest` | ❌ Default | End-user install |
| GoReleaser | ✅ Release | Official releases, Homebrew |

### Verification

Check version information is properly injected:

```bash
./packnplay version
```

**Good output (version injected):**
```
packnplay v1.1.0-89-g8a39345
  commit: 8a393453a45aa38bc5cc0ca60c7560fabdabc13d
  built:  2025-11-15T17:19:46Z
```

**Bad output (default values):**
```
packnplay dev
  commit: none
  built:  unknown
```

## Makefile Targets

- `make build` - Build binary with version info
- `make install` - Install to GOPATH/bin with version info
- `make test` - Run test suite
- `make clean` - Remove build artifacts
- `make docker-build` - Build container image
- `make help` - Show all available targets

## Troubleshooting

**Q: Version shows "dev", "none", "unknown"**
- Use `make build` instead of `go build`
- Ensure you're in a git repository with commits
- Check that git is installed and working

**Q: Build fails with git command errors**
- Ensure git is installed: `git --version`
- Ensure you're in a git repository: `git status`
- Check git repository has commits: `git log --oneline -1`

**Q: Different version info between local and release builds**
- Local builds use `git describe` (shows commits ahead of tag)
- Release builds use GoReleaser (shows exact tag version)
- Both are correct for their context
```

**Step 2: Add build documentation target to Makefile**

```makefile
.PHONY: help build install test clean docker-build docker-push lint lint-fix docs

docs: ## Open build documentation
	@echo "Build documentation: docs/building.md"
	@echo "Release process: docs/release-process.md"
	@echo "Release engineering: docs/release-engineering.md"
```

**Step 3: Test build documentation**

```bash
# Verify new target works
make docs

# Test build instructions from docs
make clean
make build
./packnplay version
```

**Step 4: Update help target**

```makefile
help: ## Show this help
	@echo "Available targets:"
	@echo "  Build targets:"
	@echo "    build       Build binary with version info"
	@echo "    install     Install to GOPATH/bin with version info"
	@echo "  Quality targets:"
	@echo "    test        Run tests"
	@echo "    lint        Run golangci-lint"
	@echo "  Utility targets:"
	@echo "    clean       Clean build artifacts"
	@echo "    docs        Show documentation links"
	@echo "    help        Show this help"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
```

**Step 5: Commit build documentation**

```bash
git add docs/building.md Makefile
git commit -m "docs: add comprehensive build documentation

- Document version injection system
- Explain different build methods and their outputs
- Add troubleshooting guide
- Update Makefile help target"
```

---

## Task 5: Integration Testing

**Files:**
- Test: Full build/release workflow
- Test: Version consistency across methods
- Test: Documentation accuracy

**Step 1: Test development build workflow**

```bash
# Clean slate test
make clean
rm -f packnplay

# Build and verify
make build
./packnplay version

# Install and verify
make install
packnplay version
```

**Step 2: Test documentation instructions**

Follow README instructions exactly as written:

```bash
# Test source build instructions from README
cd /tmp
git clone https://github.com/obra/packnplay.git packnplay-doc-test
cd packnplay-doc-test
make build
./packnplay version
```

Expected: Instructions work without modification.

**Step 3: Verify version consistency**

Check that version information is consistent:

```bash
# Get git-derived version
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "Git version: $VERSION"
echo "Git commit: $COMMIT"

# Compare with binary output
./packnplay version
```

**Step 4: Test edge cases**

```bash
# Test in detached HEAD state
git checkout HEAD~5
make clean && make build
./packnplay version

# Return to main
git checkout main
```

**Step 5: Cleanup and final verification**

```bash
# Clean up test directories
cd /Users/jesse/Documents/GitHub/packnplay
rm -rf /tmp/packnplay-doc-test

# Final build verification
make clean
make build
./packnplay version

# Verify output format
./packnplay version | grep -E "^packnplay v[0-9]"
./packnplay version | grep -E "commit: [a-f0-9]{40}"
./packnplay version | grep -E "built: [0-9]{4}-[0-9]{2}-[0-9]{2}T"
```

**Step 6: Final integration commit**

```bash
git add -A
git commit -m "feat: complete version system integration

- Update documentation for new version injection system
- Verify CI/CD compatibility with Makefile changes
- Add comprehensive build documentation
- Test full integration workflow

Version system now provides consistent information across:
- Development builds (make build)
- Release builds (GoReleaser)
- Homebrew formula testing
- Documentation examples"
```

---

## Verification Checklist

Before marking complete, verify:

- [ ] `make build` produces binary with correct version info
- [ ] Documentation accurately reflects build process
- [ ] CI workflows remain compatible
- [ ] GoReleaser configuration is consistent
- [ ] Homebrew formula test passes version check
- [ ] All documentation examples work as written
- [ ] Version format is consistent across build methods

**Success Criteria:**
1. Local builds show git-derived version (e.g., `v1.1.0-89-g8a39345`)
2. Release builds show tag-derived version (e.g., `v1.2.0`)
3. Documentation instructions are accurate and complete
4. CI/CD pipeline works without modification
5. Homebrew formula continues to work correctly
