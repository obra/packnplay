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
