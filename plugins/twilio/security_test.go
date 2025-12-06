// ABOUTME: Security validation tests for Twilio plugin
// ABOUTME: Tests SSRF protection and input validation

package twilio

import (
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		// Valid URLs
		{"valid http", "http://example.com/webhook", false},
		{"valid https", "https://example.com/webhook", false},
		{"empty url", "", false},

		// SSRF attacks - should be blocked
		{"localhost", "http://localhost/webhook", true},
		{"127.0.0.1", "http://127.0.0.1/webhook", true},
		{"ipv6 localhost", "http://[::1]/webhook", true},
		{"private 10.x", "http://10.0.0.1/webhook", true},
		{"private 192.168.x", "http://192.168.1.1/webhook", true},
		{"private 172.16.x", "http://172.16.0.1/webhook", true},
		{"private 172.31.x", "http://172.31.255.254/webhook", true},
		{"link local", "http://169.254.169.254/metadata", true},

		// Invalid schemes
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/file", true},

		// Malformed
		{"invalid url", "not a url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			hasError := err != nil

			if hasError != tt.expectErr {
				t.Errorf("validateWebhookURL(%q) error = %v, expectErr = %v", tt.url, err, tt.expectErr)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		host    string
		isPriv  bool
	}{
		// Private IPs
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"169.254.1.1", true},

		// Public IPs/domains
		{"example.com", false},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		{"11.0.0.1", false},
		{"193.168.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := isPrivateIP(tt.host)
			if result != tt.isPriv {
				t.Errorf("isPrivateIP(%q) = %v, expected %v", tt.host, result, tt.isPriv)
			}
		})
	}
}
