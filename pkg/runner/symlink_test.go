package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSymlinkedFile(t *testing.T) {
	// Test the simple case: ~/.gitconfig is a symlink to ~/dotfiles/gitconfig
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	dotfilesDir := filepath.Join(tempDir, "dotfiles")

	err := os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	err = os.MkdirAll(dotfilesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create dotfiles dir: %v", err)
	}

	// Create actual gitconfig file in dotfiles directory
	gitconfigContent := `[user]
	name = Test User
	email = test@example.com
`
	actualGitconfig := filepath.Join(dotfilesDir, "gitconfig")
	err = os.WriteFile(actualGitconfig, []byte(gitconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write actual gitconfig: %v", err)
	}

	// Create symlink from ~/.gitconfig to the actual file
	gitconfigSymlink := filepath.Join(homeDir, ".gitconfig")
	err = os.Symlink(actualGitconfig, gitconfigSymlink)
	if err != nil {
		t.Fatalf("Failed to create gitconfig symlink: %v", err)
	}

	// Test our function that should resolve symlinks for mounting
	resolvedPath, err := resolveMountPath(gitconfigSymlink)
	if err != nil {
		t.Fatalf("resolveMountPath failed: %v", err)
	}

	// Should return the actual file path, not the symlink
	// Use filepath.EvalSymlinks on expected path to handle platform differences (e.g., /var vs /private/var on macOS)
	expectedPath, err := filepath.EvalSymlinks(actualGitconfig)
	if err != nil {
		t.Fatalf("Failed to evaluate symlinks on expected path: %v", err)
	}

	if resolvedPath != expectedPath {
		t.Errorf("resolveMountPath() = %s, want %s", resolvedPath, expectedPath)
	}
}

func TestResolveRegularFile(t *testing.T) {
	// Test that regular files (not symlinks) are returned as-is
	tempDir := t.TempDir()

	// Create a regular gitconfig file
	gitconfigPath := filepath.Join(tempDir, ".gitconfig")
	gitconfigContent := `[user]
	name = Regular User
	email = regular@example.com
`
	err := os.WriteFile(gitconfigPath, []byte(gitconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write gitconfig: %v", err)
	}

	// Test that regular files are returned unchanged
	resolvedPath, err := resolveMountPath(gitconfigPath)
	if err != nil {
		t.Fatalf("resolveMountPath failed: %v", err)
	}

	// Should return the same path for regular files (after resolving any system symlinks)
	// Use filepath.EvalSymlinks on expected path to handle platform differences (e.g., /var vs /private/var on macOS)
	expectedPath, err := filepath.EvalSymlinks(gitconfigPath)
	if err != nil {
		t.Fatalf("Failed to evaluate symlinks on expected path: %v", err)
	}

	if resolvedPath != expectedPath {
		t.Errorf("resolveMountPath() = %s, want %s", resolvedPath, expectedPath)
	}
}

func TestRealSymlinkedGitconfig(t *testing.T) {
	// Test with actual user's gitconfig if it's a symlink
	gitconfigPath := filepath.Join(os.Getenv("HOME"), ".gitconfig")

	if !fileExists(gitconfigPath) {
		t.Skip("No ~/.gitconfig found")
	}

	// Check if it's a symlink
	info, err := os.Lstat(gitconfigPath)
	if err != nil {
		t.Fatalf("Failed to lstat gitconfig: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Skip("~/.gitconfig is not a symlink")
	}

	// Test that our resolution works
	resolvedPath, err := resolveMountPath(gitconfigPath)
	if err != nil {
		t.Fatalf("resolveMountPath failed on real gitconfig: %v", err)
	}

	// Resolved path should be different from original (since it's a symlink)
	if resolvedPath == gitconfigPath {
		t.Errorf("resolveMountPath() should resolve symlink, got same path: %s", resolvedPath)
	}

	// Resolved path should exist
	if !fileExists(resolvedPath) {
		t.Errorf("Resolved path %s should exist", resolvedPath)
	}

	// Both paths should have same content
	originalContent, err := os.ReadFile(gitconfigPath)
	if err != nil {
		t.Fatalf("Failed to read original gitconfig: %v", err)
	}

	resolvedContent, err := os.ReadFile(resolvedPath)
	if err != nil {
		t.Fatalf("Failed to read resolved gitconfig: %v", err)
	}

	if string(originalContent) != string(resolvedContent) {
		t.Errorf("Content mismatch between symlink and target")
	}

	t.Logf("Successfully resolved symlink %s -> %s", gitconfigPath, resolvedPath)
}
