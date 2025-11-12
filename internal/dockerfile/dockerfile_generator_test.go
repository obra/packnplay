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
