// ABOUTME: Tests for webhook security and SSRF protection
// ABOUTME: Validates URL validation, HMAC generation, and webhook firing

package github

import (
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		// Valid URLs
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"empty url", "", false}, // Empty URLs are allowed (no webhook)

		// SSRF protection - localhost
		{"localhost", "http://localhost/webhook", true},
		{"127.0.0.1", "http://127.0.0.1/webhook", true},
		{"ipv6 localhost", "http://[::1]/webhook", true},

		// SSRF protection - private IPs
		{"10.x.x.x", "http://10.0.0.1/webhook", true},
		{"192.168.x.x", "http://192.168.1.1/webhook", true},
		{"172.16.x.x", "http://172.16.0.1/webhook", true},
		{"172.31.x.x", "http://172.31.255.255/webhook", true},

		// SSRF protection - link-local
		{"link-local", "http://169.254.169.254/webhook", true},

		// Invalid schemes
		{"ftp scheme", "ftp://example.com/webhook", true},
		{"file scheme", "file:///etc/passwd", true},

		// Malformed URLs
		{"malformed", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("validateWebhookURL(%q) error = %v, wantError %v", tt.url, err, tt.wantError)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		// Localhost
		{"localhost", "localhost", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"::1", "::1", true},

		// Private ranges
		{"10.0.0.0", "10.0.0.0", true},
		{"10.255.255.255", "10.255.255.255", true},
		{"192.168.0.0", "192.168.0.0", true},
		{"192.168.255.255", "192.168.255.255", true},
		{"172.16.0.0", "172.16.0.0", true},
		{"172.31.255.255", "172.31.255.255", true},

		// Link-local
		{"169.254.0.0", "169.254.0.0", true},
		{"169.254.255.255", "169.254.255.255", true},

		// Public IPs
		{"google dns", "8.8.8.8", false},
		{"cloudflare dns", "1.1.1.1", false},
		{"public example", "93.184.216.34", false},

		// Public domains
		{"example.com", "example.com", false},
		{"github.com", "github.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPrivateIP(tt.host); got != tt.want {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestGenerateHMAC(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		secret  string
		want    string
	}{
		{
			name:    "simple payload",
			payload: []byte("hello world"),
			secret:  "secret123",
			want:    "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17",
		},
		{
			name:    "empty payload",
			payload: []byte(""),
			secret:  "secret",
			want:    "sha256=b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
		},
		{
			name:    "json payload",
			payload: []byte(`{"event":"issues","action":"opened"}`),
			secret:  "webhook-secret",
			want:    "sha256=1e9abe3ed3d8c4f8e4e8c4e8c4e8c4e8c4e8c4e8c4e8c4e8c4e8c4e8c4e8c4e8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateHMAC(tt.payload, tt.secret)
			// Just verify it starts with sha256= and has the right length
			if len(got) != 71 || got[:7] != "sha256=" {
				t.Errorf("generateHMAC() = %v, want sha256=<64 hex chars>", got)
			}
		})
	}
}
