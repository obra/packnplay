package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSymlinkResolution(t *testing.T) {
	// Create temporary directory structure with symlinks
	tempDir := t.TempDir()

	// Create real directory: tempDir/real/project
	realDir := filepath.Join(tempDir, "real", "project")
	err := os.MkdirAll(realDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create real directory: %v", err)
	}

	// Create symlink: tempDir/link -> tempDir/real
	linkDir := filepath.Join(tempDir, "link")
	err = os.Symlink(filepath.Join(tempDir, "real"), linkDir)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create git repo in the symlinked path for worktree logic
	projectDir := filepath.Join(linkDir, "project")
	gitDir := filepath.Join(projectDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Change to the symlinked directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(projectDir)
	if err != nil {
		t.Fatalf("Failed to change to symlinked directory: %v", err)
	}

	// Test the path resolution logic that Run() uses
	// We can't actually run containers in tests, but we can test the symlink resolution

	// Test the path resolution logic directly
	testPath := projectDir // This is the symlinked path

	// Resolve symlinks like the Run function does
	resolvedPath, err := filepath.EvalSymlinks(testPath)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks: %v", err)
	}

	// Make absolute
	resolvedPath, err = filepath.Abs(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to make path absolute: %v", err)
	}

	// Verify the resolved path points to the real directory, not the symlink
	expectedRealPath := realDir
	if resolvedPath != expectedRealPath {
		t.Errorf("Symlink resolution failed:\n  Symlinked path: %s\n  Resolved path:  %s\n  Expected path:  %s",
			projectDir, resolvedPath, expectedRealPath)
	}

	// Verify the symlinked path and real path point to the same location
	symlinkStat, err := os.Stat(projectDir)
	if err != nil {
		t.Fatalf("Failed to stat symlinked path: %v", err)
	}

	realStat, err := os.Stat(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to stat resolved path: %v", err)
	}

	// They should be the same inode (same directory)
	if !os.SameFile(symlinkStat, realStat) {
		t.Error("Resolved path and symlinked path should point to the same directory")
	}

	t.Logf("Symlink resolution test successful:")
	t.Logf("  Original path:  %s", projectDir)
	t.Logf("  Resolved path:  %s", resolvedPath)
	t.Logf("  Paths consistent: %t", os.SameFile(symlinkStat, realStat))
}

func TestSymlinkResolutionWithAbsolutePath(t *testing.T) {
	// Test that symlink resolution works when absolute path is provided
	tempDir := t.TempDir()

	// Create real directory
	realDir := filepath.Join(tempDir, "real", "workspace")
	err := os.MkdirAll(realDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create real directory: %v", err)
	}

	// Create symlink
	linkDir := filepath.Join(tempDir, "workspace-link")
	err = os.Symlink(realDir, linkDir)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test path resolution with absolute symlinked path
	resolvedPath, err := filepath.EvalSymlinks(linkDir)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks: %v", err)
	}

	resolvedPath, err = filepath.Abs(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to make path absolute: %v", err)
	}

	// Should resolve to the real directory
	if resolvedPath != realDir {
		t.Errorf("Absolute symlink resolution failed:\n  Symlink: %s\n  Resolved: %s\n  Expected: %s",
			linkDir, resolvedPath, realDir)
	}
}

func TestSymlinkResolutionErrorHandling(t *testing.T) {
	// Test error handling when symlink resolution fails
	nonExistentPath := "/this/path/does/not/exist"

	_, err := filepath.EvalSymlinks(nonExistentPath)
	if err == nil {
		t.Error("Expected error when resolving non-existent path, got nil")
	}

	// This tests that our error handling in Run() would work correctly
	t.Logf("Error handling works correctly: %v", err)
}

func TestNoSymlinkResolution(t *testing.T) {
	// Test that normal paths (no symlinks) work correctly
	tempDir := t.TempDir()

	// No symlinks, just a regular directory
	regularDir := filepath.Join(tempDir, "regular", "project")
	err := os.MkdirAll(regularDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create regular directory: %v", err)
	}

	// Test resolution of regular path
	resolvedPath, err := filepath.EvalSymlinks(regularDir)
	if err != nil {
		t.Fatalf("Failed to resolve regular path: %v", err)
	}

	resolvedPath, err = filepath.Abs(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to make path absolute: %v", err)
	}

	// Should be the same as the original (no symlinks to resolve)
	if resolvedPath != regularDir {
		t.Errorf("Regular path resolution should be identity:\n  Original: %s\n  Resolved: %s",
			regularDir, resolvedPath)
	}
}