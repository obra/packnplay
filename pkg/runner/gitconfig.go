package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// isSSHURL returns true if the given URL uses SSH transport.
// It detects both SCP-style (git@host:path) and URL-style (ssh://host/path) formats.
func isSSHURL(url string) bool {
	// URL-style SSH
	if strings.HasPrefix(url, "ssh://") {
		return true
	}

	// SCP-style: user@host:path (must have @ before : and no //)
	if strings.Contains(url, "@") && strings.Contains(url, ":") {
		atIdx := strings.Index(url, "@")
		colonIdx := strings.Index(url, ":")
		// The @ must come before the : and there should be no :// (which would be a URL scheme)
		if atIdx < colonIdx && !strings.Contains(url[:colonIdx+1], "://") {
			return true
		}
	}

	return false
}

// hasSSHInsteadOfRules checks the host's git config for insteadOf rules that
// rewrite URLs to SSH. Returns the list of rewrite descriptions found.
func hasSSHInsteadOfRules() []string {
	output, err := exec.Command("git", "config", "--global", "--get-regexp",
		`url\..*\.(insteadof|pushinsteadof)`).Output()
	if err != nil {
		return nil
	}

	var rewrites []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]   // e.g., "url.git@github.com:.insteadof"
		value := parts[1] // e.g., "https://github.com/"

		if !strings.HasPrefix(key, "url.") {
			continue
		}

		rest := key[4:]
		restLower := strings.ToLower(rest)

		var baseURL string
		if idx := strings.LastIndex(restLower, ".insteadof"); idx != -1 {
			baseURL = rest[:idx]
		} else if idx := strings.LastIndex(restLower, ".pushinsteadof"); idx != -1 {
			baseURL = rest[:idx]
		} else {
			continue
		}

		if isSSHURL(baseURL) {
			rewrites = append(rewrites, fmt.Sprintf("  %s -> %s", value, baseURL))
		}
	}

	return rewrites
}

// warnSSHInsteadOfRules prints a warning if the user's gitconfig rewrites URLs
// to SSH but SSH keys are not being forwarded into the container.
func warnSSHInsteadOfRules() {
	warnSSHInsteadOfRulesTo(os.Stderr)
}

// warnSSHInsteadOfRulesTo writes the SSH insteadOf warning to w.
// Separated from warnSSHInsteadOfRules to allow testing.
func warnSSHInsteadOfRulesTo(w io.Writer) {
	rewrites := hasSSHInsteadOfRules()
	if len(rewrites) == 0 {
		return
	}

	fmt.Fprintf(w, "\n"+
		"WARNING: Your ~/.gitconfig contains insteadOf rules that rewrite URLs to SSH:\n"+
		"\n")
	for _, r := range rewrites {
		fmt.Fprintf(w, "%s\n", r)
	}
	fmt.Fprintf(w, "\n"+
		"Since SSH keys are not being forwarded to the container, git operations\n"+
		"using these rewritten URLs will fail. Consider either:\n"+
		"  - Forwarding your SSH keys with: packnplay --credentials ssh\n"+
		"  - Removing the insteadOf rules from your ~/.gitconfig\n"+
		"\n")
}
