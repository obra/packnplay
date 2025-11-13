package dockerfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
)

func TestGenerateWithFeatures(t *testing.T) {
	// Create a temporary directory for test feature
	tempDir := t.TempDir()
	featureDir := filepath.Join(tempDir, "test-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature directory: %v", err)
	}

	// Create a simple install.sh
	installScript := `#!/bin/bash
echo "Installing test feature"
apt-get update
apt-get install -y curl
`
	if err := os.WriteFile(filepath.Join(featureDir, "install.sh"), []byte(installScript), 0755); err != nil {
		t.Fatalf("Failed to write install.sh: %v", err)
	}

	// Create a ResolvedFeature
	resolvedFeature := &devcontainer.ResolvedFeature{
		ID:          "test-feature",
		Version:     "1.0.0",
		InstallPath: featureDir,
	}

	// Generate Dockerfile
	generator := NewDockerfileGenerator()
	dockerfile, err := generator.Generate("ubuntu:22.04", "vscode", []*devcontainer.ResolvedFeature{resolvedFeature}, tempDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify Dockerfile contents
	if !strings.Contains(dockerfile, "FROM ubuntu:22.04") {
		t.Errorf("Dockerfile missing FROM statement")
	}

	if !strings.Contains(dockerfile, "USER root") {
		t.Errorf("Dockerfile missing USER root statement")
	}

	// Verify COPY command for feature
	if !strings.Contains(dockerfile, "COPY") {
		t.Errorf("Dockerfile missing COPY command for feature")
	}

	// Verify RUN command to execute install.sh
	if !strings.Contains(dockerfile, "RUN cd /tmp/devcontainer-features") {
		t.Errorf("Dockerfile missing RUN command to execute feature install.sh")
	}

	if !strings.Contains(dockerfile, "./install.sh") {
		t.Errorf("Dockerfile missing install.sh execution")
	}

	if !strings.Contains(dockerfile, "USER vscode") {
		t.Errorf("Dockerfile missing USER vscode statement at end")
	}

	// Verify order: FROM before USER root before COPY before RUN before USER vscode
	fromIdx := strings.Index(dockerfile, "FROM")
	userRootIdx := strings.Index(dockerfile, "USER root")
	copyIdx := strings.Index(dockerfile, "COPY")
	runIdx := strings.Index(dockerfile, "RUN")
	userVscodeIdx := strings.LastIndex(dockerfile, "USER vscode")

	if fromIdx > userRootIdx {
		t.Errorf("FROM should come before USER root")
	}
	if userRootIdx > copyIdx {
		t.Errorf("USER root should come before COPY")
	}
	if copyIdx > runIdx {
		t.Errorf("COPY should come before RUN")
	}
	if runIdx > userVscodeIdx {
		t.Errorf("RUN should come before final USER statement")
	}
}

func TestGenerateMultiStageWithOCIFeatures(t *testing.T) {
	// Create OCI feature (simulated cached feature)
	tmpDir := t.TempDir()
	ociFeatureDir := filepath.Join(tmpDir, "oci-cache", "common-utils")
	err := os.MkdirAll(ociFeatureDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create OCI feature directory: %v", err)
	}

	// Create install script
	installScript := "#!/bin/bash\necho 'Installing common-utils'"
	err = os.WriteFile(filepath.Join(ociFeatureDir, "install.sh"), []byte(installScript), 0755)
	if err != nil {
		t.Fatalf("Failed to write install.sh: %v", err)
	}

	// Create feature with OCI path
	ociFeature := &devcontainer.ResolvedFeature{
		ID:          "common-utils",
		Version:     "2.0.0",
		InstallPath: ociFeatureDir,
		Options:     map[string]interface{}{},
	}

	generator := NewDockerfileGenerator()
	buildContextPath := filepath.Join(tmpDir, "project", ".devcontainer")
	dockerfile, err := generator.Generate("ubuntu:22.04", "testuser", []*devcontainer.ResolvedFeature{ociFeature}, buildContextPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Print the generated Dockerfile for verification
	t.Logf("Generated multi-stage Dockerfile:\n%s", dockerfile)

	// Verify multi-stage build structure
	if !strings.Contains(dockerfile, "FROM ubuntu:22.04 as base") {
		t.Errorf("Dockerfile should have base stage with 'FROM ubuntu:22.04 as base'\nGot:\n%s", dockerfile)
	}
	if !strings.Contains(dockerfile, "FROM alpine:latest as feature-prep") {
		t.Errorf("Dockerfile should have feature-prep stage with 'FROM alpine:latest as feature-prep'\nGot:\n%s", dockerfile)
	}
	if !strings.Contains(dockerfile, "COPY --from=") {
		t.Errorf("Dockerfile should have COPY --from= for multi-stage\nGot:\n%s", dockerfile)
	}
	// Check that we're using multi-stage by checking for the named stages
	if !strings.Contains(dockerfile, "as feature-prep") {
		t.Errorf("Dockerfile should have named feature-prep stage\nGot:\n%s", dockerfile)
	}
	if !strings.Contains(dockerfile, "as base") {
		t.Errorf("Dockerfile should have named base stage\nGot:\n%s", dockerfile)
	}
}

func TestGenerateSingleStageWithLocalFeature(t *testing.T) {
	// Create a local feature within the build context
	tmpDir := t.TempDir()
	buildContextPath := filepath.Join(tmpDir, ".devcontainer")
	localFeatureDir := filepath.Join(buildContextPath, "local-features", "test-feature")
	err := os.MkdirAll(localFeatureDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create local feature directory: %v", err)
	}

	// Create install script
	installScript := "#!/bin/bash\necho 'Installing local feature'"
	err = os.WriteFile(filepath.Join(localFeatureDir, "install.sh"), []byte(installScript), 0755)
	if err != nil {
		t.Fatalf("Failed to write install.sh: %v", err)
	}

	// Create feature with local path
	localFeature := &devcontainer.ResolvedFeature{
		ID:          "test-feature",
		Version:     "1.0.0",
		InstallPath: localFeatureDir,
		Options:     map[string]interface{}{},
	}

	generator := NewDockerfileGenerator()
	dockerfile, err := generator.Generate("ubuntu:22.04", "testuser", []*devcontainer.ResolvedFeature{localFeature}, buildContextPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Generated single-stage Dockerfile:\n%s", dockerfile)

	// Verify single-stage build structure (no multi-stage)
	if strings.Contains(dockerfile, "as feature-prep") {
		t.Errorf("Local feature should use single-stage build, not multi-stage\nGot:\n%s", dockerfile)
	}
	if strings.Contains(dockerfile, "COPY --from=") {
		t.Errorf("Single-stage build should not use COPY --from=\nGot:\n%s", dockerfile)
	}
	if !strings.Contains(dockerfile, "FROM ubuntu:22.04") {
		t.Errorf("Dockerfile should have FROM statement")
	}
	if !strings.Contains(dockerfile, "USER root") {
		t.Errorf("Dockerfile should switch to root for installation")
	}
	if !strings.Contains(dockerfile, "USER testuser") {
		t.Errorf("Dockerfile should switch back to testuser")
	}
}

func TestFeatureUserContextVariables(t *testing.T) {
	tests := []struct {
		name          string
		remoteUser    string
		useMultiStage bool
		expectedEnvs  []string
	}{
		{
			name:          "single stage with vscode user",
			remoteUser:    "vscode",
			useMultiStage: false,
			expectedEnvs: []string{
				"ENV _REMOTE_USER=vscode",
				"ENV _REMOTE_USER_HOME=/home/vscode",
				"ENV _CONTAINER_USER=vscode",
			},
		},
		{
			name:          "multi stage with custom user",
			remoteUser:    "developer",
			useMultiStage: true,
			expectedEnvs: []string{
				"ENV _REMOTE_USER=developer",
				"ENV _REMOTE_USER_HOME=/home/developer",
				"ENV _CONTAINER_USER=developer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			buildContextPath := filepath.Join(tmpDir, ".devcontainer")

			var featureDir string
			if tt.useMultiStage {
				// OCI feature (outside build context for multi-stage)
				featureDir = filepath.Join(tmpDir, "oci-cache", "test-feature")
			} else {
				// Local feature (inside build context for single-stage)
				featureDir = filepath.Join(buildContextPath, "local-features", "test-feature")
			}

			err := os.MkdirAll(featureDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create feature directory: %v", err)
			}

			// Create install script
			installScript := "#!/bin/bash\necho 'Installing test feature'"
			err = os.WriteFile(filepath.Join(featureDir, "install.sh"), []byte(installScript), 0755)
			if err != nil {
				t.Fatalf("Failed to write install.sh: %v", err)
			}

			// Create feature
			feature := &devcontainer.ResolvedFeature{
				ID:          "test-feature",
				Version:     "1.0.0",
				InstallPath: featureDir,
				Options:     map[string]interface{}{},
			}

			generator := NewDockerfileGenerator()
			dockerfile, err := generator.Generate("ubuntu:22.04", tt.remoteUser, []*devcontainer.ResolvedFeature{feature}, buildContextPath)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			t.Logf("Generated Dockerfile:\n%s", dockerfile)

			// Verify all expected environment variables are present
			for _, expectedEnv := range tt.expectedEnvs {
				if !strings.Contains(dockerfile, expectedEnv) {
					t.Errorf("Dockerfile missing expected environment variable: %s\nGot:\n%s", expectedEnv, dockerfile)
				}
			}

			// Verify env vars come after USER root
			userRootIdx := strings.Index(dockerfile, "USER root")
			if userRootIdx == -1 {
				t.Fatalf("Dockerfile missing USER root statement")
			}

			for _, expectedEnv := range tt.expectedEnvs {
				envIdx := strings.Index(dockerfile, expectedEnv)
				if envIdx != -1 && envIdx < userRootIdx {
					t.Errorf("Environment variable %s should come after USER root", expectedEnv)
				}
			}
		})
	}
}
