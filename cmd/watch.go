package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:    "watch-credentials",
	Short:  "Watch container credential files and sync to keychain",
	Long:   `Background daemon that watches container credential files and syncs them to keychain and other containers.`,
	Hidden: true, // Hide from help - internal command
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCredentialWatcher()
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

type credentialWatcher struct {
	credentialsDir string
	keychainKey    string
	lastUpdate     time.Time
	watcher        *fsnotify.Watcher
}

func runCredentialWatcher() error {
	w := &credentialWatcher{
		credentialsDir: getCredentialsDir(),
		keychainKey:    "packnplay-containers-credentials",
	}

	// Ensure credentials directory exists
	if err := os.MkdirAll(w.credentialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create credentials dir: %w", err)
	}

	// Create filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()
	w.watcher = watcher

	// Watch the credentials directory
	if err := watcher.Add(w.credentialsDir); err != nil {
		return fmt.Errorf("failed to watch credentials dir: %w", err)
	}

	log.Printf("Watching credential files in %s", w.credentialsDir)

	// Event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				if strings.Contains(event.Name, "packnplay-claude-credentials") {
					if err := w.handleCredentialUpdate(event.Name); err != nil {
						log.Printf("Error handling credential update: %v", err)
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			log.Printf("Watcher error: %v", err)

		case <-time.After(30 * time.Second):
			// Periodic check if we should exit (no containers running)
			if !hasRunningContainers() {
				log.Printf("No containers running, exiting credential watcher")
				return nil
			}
		}
	}
}

func (w *credentialWatcher) handleCredentialUpdate(filePath string) error {
	// Check if this update is newer than our last keychain write (avoid recursion)
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if stat.ModTime().Before(w.lastUpdate) {
		// This is our own writeback, ignore
		return nil
	}

	log.Printf("Credential file updated: %s", filePath)

	// Read the updated credentials
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read credential file: %w", err)
	}

	// macOS: Update keychain
	if isDarwin() {
		if err := w.updateKeychain(string(content)); err != nil {
			return fmt.Errorf("failed to update keychain: %w", err)
		}
	}

	// Update other container credential files
	if err := w.syncToOtherContainers(filePath, content); err != nil {
		return fmt.Errorf("failed to sync to other containers: %w", err)
	}

	w.lastUpdate = time.Now()
	return nil
}

func (w *credentialWatcher) updateKeychain(credentials string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-s", w.keychainKey,
		"-a", "packnplay",
		"-w", credentials,
		"-U") // -U updates if exists
	return cmd.Run()
}

func (w *credentialWatcher) syncToOtherContainers(changedFile string, content []byte) error {
	// Find all credential files except the one that changed
	files, err := filepath.Glob(filepath.Join(w.credentialsDir, "container-*.credentials.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if file == changedFile {
			continue // Skip the file that triggered this update
		}

		if err := os.WriteFile(file, content, 0600); err != nil {
			log.Printf("Warning: failed to update %s: %v", file, err)
			continue
		}

		// Set mtime to our update time to prevent recursive updates
		if err := os.Chtimes(file, w.lastUpdate, w.lastUpdate); err != nil {
			log.Printf("Warning: failed to set timestamp on %s: %v", file, err)
		}
	}

	return nil
}

func hasRunningContainers() bool {
	// Quick check if any packnplay containers are running
	cmd := exec.Command("docker", "ps", "--filter", "label=managed-by=packnplay", "-q")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func isDarwin() bool {
	return false // We're on Linux, would be runtime.GOOS == "darwin"
}

func getCredentialsDir() string {
	home, _ := os.UserHomeDir()
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdgDataHome, "packnplay", "credentials")
}
