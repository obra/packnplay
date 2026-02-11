package runner

import (
	"testing"
)

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		// SCP-style SSH URLs
		{name: "scp style github", url: "git@github.com:", want: true},
		{name: "scp style gitlab", url: "git@gitlab.com:", want: true},
		{name: "scp style with path", url: "git@github.com:user/repo", want: true},
		{name: "scp style custom user", url: "deploy@myhost.com:repos/", want: true},

		// URL-style SSH URLs
		{name: "ssh scheme", url: "ssh://git@github.com/", want: true},
		{name: "ssh scheme with port", url: "ssh://git@github.com:22/user/repo", want: true},
		{name: "ssh scheme no user", url: "ssh://github.com/user/repo", want: true},

		// Non-SSH URLs
		{name: "https github", url: "https://github.com/", want: false},
		{name: "http github", url: "http://github.com/", want: false},
		{name: "https with user", url: "https://user@github.com/repo", want: false},
		{name: "file protocol", url: "file:///path/to/repo", want: false},
		{name: "bare path", url: "/path/to/repo", want: false},
		{name: "empty string", url: "", want: false},

		// Edge cases
		{name: "git protocol", url: "git://github.com/user/repo", want: false},
		{name: "ftp protocol", url: "ftp://example.com/repo", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSSHURL(tt.url)
			if got != tt.want {
				t.Errorf("isSSHURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
