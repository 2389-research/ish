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
	"net/http"
	"net/url"
	"strings"
)

// isPrivateIP checks if a hostname is a private or internal address
func isPrivateIP(host string) bool {
	// Block localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Block private IP ranges
	if strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "172.16.") ||
		strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") ||
		strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.20.") ||
		strings.HasPrefix(host, "172.21.") ||
		strings.HasPrefix(host, "172.22.") ||
		strings.HasPrefix(host, "172.23.") ||
		strings.HasPrefix(host, "172.24.") ||
		strings.HasPrefix(host, "172.25.") ||
		strings.HasPrefix(host, "172.26.") ||
		strings.HasPrefix(host, "172.27.") ||
		strings.HasPrefix(host, "172.28.") ||
		strings.HasPrefix(host, "172.29.") ||
		strings.HasPrefix(host, "172.30.") ||
		strings.HasPrefix(host, "172.31.") ||
		strings.HasPrefix(host, "169.254.") {
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
func fireWebhook(webhook *Webhook, eventType string, payload interface{}) error {
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
		Timeout: 5 * 1000000000, // 5 seconds
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
