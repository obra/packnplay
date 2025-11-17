# Changelog

All notable changes to packnplay will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.2.0] - 2025-11-16

### Added

#### OpenCode AI Platform Support
- Complete integration with OpenCode AI coding platform
- Automatic configuration directory mounting (`.config/opencode/`)
- Environment variable passthrough (`OPENCODE_API_KEY`)
- Added opencode-ai to default container AI tool suite
- Full CLI tool suite restoration with enhanced AI agent support

#### Microsoft DevContainer Features Specification Compliance
- **OCI Registry Support**: Full `ghcr.io/devcontainers/features/*` support with oras integration
- **Local Features**: Complete `.devcontainer/local-features/` directory support
- **Feature Options Processing**: Type validation, enum checking, and version constraints
- **Feature Dependencies**: Automatic resolution algorithm with circular dependency detection
- **Lifecycle Hooks**: postCreateCommand, updateContentCommand, postAttachCommand, and init/entrypoint support
- **Feature Mounts**: Feature-contributed mount points with proper integration
- **Multi-stage Builds**: Automatic detection and OCI feature build context copying
- **Container Properties**: Support for feature-contributed container configuration

#### Enhanced DevContainer Support
- Variable substitution engine for devcontainer.json properties
- Custom mounts processing with host path validation
- Custom runArgs integration with Docker command generation
- Port forwarding configuration from devcontainer forwardPorts
- Environment variables with full substitution support
- Build configuration parsing with cacheFrom and options arrays
- Lifecycle command execution with run-once behavior tracking
- Signal handling and secure port defaults for Microsoft compatibility

#### Testing and Quality Assurance
- Comprehensive E2E test suite covering Microsoft universal devcontainer patterns
- Feature specification compliance tests with real OCI registry integration
- Advanced test coverage for feature options validation and lifecycle commands
- Multi-stage build detection and feature integration testing
- Microsoft DevContainer Features specification compliance verification

### Changed
- Updated default container to use Microsoft devcontainer features instead of manual tool installation
- Enhanced container image references to use published devcontainer image
- Improved symlink resolution for consistent container reconnection paths
- Updated container name generation to match GitHub build standards

### Fixed
- Resolved symlinks for consistent container reconnection paths
- Fixed workspaceFolder mapping and shell execution issues
- Corrected port range validation with comprehensive edge case testing
- Addressed E2E test port conflicts and build args scoping issues
- Fixed container cleanup timeout warnings in E2E test suite
- Resolved golangci-lint issues with local linting configuration

### Technical Details
- Microsoft DevContainer Features specification: Full compliance with official spec
- Feature resolution priority: OCI registry → local features → fallback to manual installation
- Build system: Multi-stage Docker builds with automatic feature detection
- Feature caching: Image ID-based caching with oras CLI integration
- Variable substitution: Complete ${localEnv:VAR} and ${containerEnv:VAR} support

### Documentation
- Comprehensive devcontainer implementation documentation
- Microsoft DevContainer Features specification compliance analysis
- Complete feature analysis and implementation recommendations
- Enhanced AI agent documentation with clear value propositions
- Updated README with OpenCode AI and enhanced devcontainer support

## [v1.1.0] - 2025-11-03

### Added

#### Configuration UI Scrolling Support
- Viewport scrolling for configuration interface in small terminal windows
- Auto-scroll to keep focused elements visible during navigation
- Manual scroll controls with PgUp/PgDown and Ctrl+U/Ctrl+D keyboard shortcuts
- Visual scroll indicators ("↑ More content above ↑" / "↓ More content below ↓")
- Fixed Save/Cancel button accessibility in scrollable content with proper spacing
- Improved navigation bounds - stops at top/bottom instead of looping around
- Header visibility guaranteed when navigating to top of configuration

#### OrbStack Container Runtime Support
- OrbStack detected automatically as container runtime option alongside Docker and Podman
- Smart detection via OrbStack CLI (`orb`) and Docker context verification
- Automatic Docker context switching to `orbstack` when selected as runtime
- Full Docker CLI compatibility maintained for seamless operation
- Updated test suite to include OrbStack as valid container runtime

#### Visual Progress Bars for Container Operations
- Real-time progress bars for `docker pull` and `docker build` operations
- JSON progress parsing for precise download percentages and data transfer rates
- Smart output stream handling (stdout for pulls, stderr for builds)
- Throttled updates (100ms intervals) to prevent terminal flooding during rapid output
- Success (✅) and error (❌) completion indicators with operation timing
- Byte-formatted progress details showing download progress (e.g., "245MB/306MB")
- Automatic terminal detection - only displays in interactive environments
- Preserves verbose mode functionality with raw Docker output when `--verbose` used
- Graceful fallback to current behavior if progress parsing fails

### Improved
- Configuration interface accessibility for users with limited terminal space
- Container runtime flexibility with additional macOS-optimized option
- Navigation behavior consistency throughout configuration interface
- User experience during container setup with clear feedback on download/build progress

## [v1.0.0] - 2024-10-25

### Added

#### Smart User Detection System
- Automatic container user detection with zero configuration
- Intelligent caching by Docker image ID for performance optimization
- Direct container interrogation using `whoami && echo $HOME`
- Universal compatibility with node, ubuntu, python, and custom images
- devcontainer.json `remoteUser` field support with proper priority handling
- XDG-compliant cache storage in `~/.cache/packnplay/userdetect/`

#### Docker-Compatible Port Mapping
- Native `-p/--publish` flag with full Docker syntax compatibility
- Support for multiple port mappings: `-p 8080:3000 -p 9000:9001`
- Complete format support:
  - Basic port mapping: `8080:3000`
  - Host IP binding: `127.0.0.1:8080:3000`
  - Protocol specification: `8080:3000/tcp`, `5353:53/udp`
  - Combined format: `127.0.0.1:8080:3000/tcp`
- Seamless integration with Docker run command arguments

#### Container Management
- Automatic worktree management in XDG-compliant locations
- Dev container discovery with `.devcontainer/devcontainer.json` support
- Persistent container lifecycle with proper labeling
- Container reconnection and attachment capabilities
- Clean container cleanup with `packnplay stop --all`

#### Credential Integration
- Interactive first-run setup with terminal UI
- Secure credential mounting with read-only access
- macOS Keychain integration for automatic credential extraction
- Support for git, SSH, GitHub CLI, GPG, and npm credentials
- Per-invocation credential override flags

#### Development Experience
- Sandboxed execution with host environment isolation
- Clean environment with safe variable whitelisting
- Git integration with proper worktree and repository handling
- Multiple AI agent support (Claude Code, Codex, Gemini, Copilot, Qwen, Cursor, Amp, DeepSeek)
- Environment configuration system with variable substitution

#### Testing and Quality
- Comprehensive test coverage using Test-Driven Development (TDD)
- Integration tests for end-to-end workflows
- User detection caching tests with Docker image verification
- Port mapping compatibility tests across all Docker formats
- Performance testing for optimization verification

### Changed
- Removed "untested and experimental" warning - now production ready
- Updated documentation with comprehensive usage examples
- Enhanced README with new feature descriptions and usage patterns

### Technical Details
- User detection priority: devcontainer.json → cache → runtime detection → fallback
- Port mapping integration through RunConfig to Docker arguments
- Atomic cache writes with corruption prevention
- Image ID-based caching instead of image name caching

### Documentation
- Complete usage guide in README.md
- Release engineering process documentation
- Project-specific Claude Code instructions (CLAUDE.md)
- Comprehensive release notes with examples

---

**Note**: This is the first stable release of packnplay. All features listed above represent the initial feature set.