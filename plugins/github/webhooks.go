// ABOUTME: Webhook delivery system for GitHub plugin
// ABOUTME: Implements SSRF protection, HMAC signatures, and event firing

package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// isPrivateIP checks if an IP address is private or internal
// Resolves hostnames to IPs to prevent DNS rebinding attacks
func isPrivateIP(host string) bool {
	// First check if it's a direct IP address (not a hostname)
	if ip := net.ParseIP(host); ip != nil {
		// Direct IP provided - check if it's private
		return isPrivateIPAddress(ip)
	}

	// It's a hostname - resolve to IP addresses
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS resolution failed - block it to be safe
		// This prevents bypass via non-resolving domains
		return true
	}

	// Check all resolved IPs - if ANY resolve to private, block the whole hostname
	for _, ip := range ips {
		if isPrivateIPAddress(ip) {
			return true
		}
	}

	return false
}

// isPrivateIPAddress checks if a net.IP is private or internal
func isPrivateIPAddress(ip net.IP) bool {
	// Check for loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private IPv4 ranges
	ipv4 := ip.To4()
	if ipv4 != nil {
		// 10.0.0.0/8
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}
	}

	// Check for private IPv6 ranges
	// fc00::/7 (unique local addresses)
	if len(ip) == 16 && (ip[0] == 0xfc || ip[0] == 0xfd) {
		return true
	}

	return false
}

// validateWebhookURL validates webhook URLs to prevent SSRF attacks
func validateWebhookURL(urlStr string) error {
	if urlStr == "" {
		return nil // Empty URLs are okay (no webhook configured)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Require http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https")
	}

	// Block internal addresses
	host := u.Hostname()
	if isPrivateIP(host) {
		return fmt.Errorf("webhook URL cannot target private IP addresses")
	}

	return nil
}

// generateHMAC creates X-Hub-Signature-256 header value
func generateHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	return "sha256=" + signature
}

// fireWebhook sends an HTTP POST request to the webhook URL with the event payload
// Validates URL at delivery time to prevent DNS rebinding attacks
func fireWebhook(webhook *Webhook, eventType string, payload interface{}) error {
	// Validate URL at delivery time to prevent DNS rebinding attacks
	if err := validateWebhookURL(webhook.URL); err != nil {
		return fmt.Errorf("webhook URL validation failed at delivery: %w", err)
	}

	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if webhook.ContentType == "json" {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-GitHub-Delivery", fmt.Sprintf("%d", webhook.ID))

	// Add HMAC signature if secret is configured
	if webhook.Secret != "" {
		signature := generateHMAC(payloadBytes, webhook.Secret)
		req.Header.Set("X-Hub-Signature-256", signature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
