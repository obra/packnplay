package runner

import (
	"os"
	"runtime"
	"testing"
)

// TestFindSSHAgentSocketLinux verifies socket discovery on Linux when SSH_AUTH_SOCK is set or empty.
func TestFindSSHAgentSocketLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	t.Run("returns SSH_AUTH_SOCK when set", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "/tmp/ssh-test/agent.123")
		sock, err := findSSHAgentSocketLinux()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sock != "/tmp/ssh-test/agent.123" {
			t.Errorf("got %q, want %q", sock, "/tmp/ssh-test/agent.123")
		}
	})

	t.Run("returns error when SSH_AUTH_SOCK is empty", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		_, err := findSSHAgentSocketLinux()
		if err == nil {
			t.Fatal("expected error when SSH_AUTH_SOCK is empty")
		}
	})
}

// TestFindSSHAgentSocketLinuxUnset verifies that an error is returned when SSH_AUTH_SOCK is not set.
func TestFindSSHAgentSocketLinuxUnset(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	// Unset SSH_AUTH_SOCK for this test
	orig, wasSet := os.LookupEnv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if wasSet {
			os.Setenv("SSH_AUTH_SOCK", orig)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	_, err := findSSHAgentSocketLinux()
	if err == nil {
		t.Fatal("expected error when SSH_AUTH_SOCK is unset")
	}
}

// TestFindSSHAgentSocketDarwin verifies socket discovery on macOS for Docker Desktop and Colima.
func TestFindSSHAgentSocketDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}

	t.Run("defaults to Docker Desktop socket path", func(t *testing.T) {
		// Properly unset DOCKER_HOST (not just empty string)
		orig, wasSet := os.LookupEnv("DOCKER_HOST")
		os.Unsetenv("DOCKER_HOST")
		defer func() {
			if wasSet {
				os.Setenv("DOCKER_HOST", orig)
			} else {
				os.Unsetenv("DOCKER_HOST")
			}
		}()

		sock, err := findSSHAgentSocketDarwin()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sock != "/run/host-services/ssh-auth.sock" {
			t.Errorf("got %q, want Docker Desktop socket path", sock)
		}
	})

	t.Run("detects Colima from DOCKER_HOST", func(t *testing.T) {
		// This will call colima ssh which may not work in CI,
		// but verifies the detection path is taken
		t.Setenv("DOCKER_HOST", "unix:///Users/test/.colima/default/docker.sock")
		sock, err := findSSHAgentSocketDarwin()
		if err != nil {
			// Expected if colima isn't running
			t.Skipf("Colima not available: %v", err)
		}
		if sock == "" {
			t.Error("got empty socket path from Colima")
		}
	})
}
