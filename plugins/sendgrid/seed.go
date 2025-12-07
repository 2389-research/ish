// ABOUTME: Test data generation for SendGrid plugin
// ABOUTME: Creates sample accounts, API keys, messages, and suppressions

package sendgrid

import (
	"context"
	"fmt"

	"github.com/2389/ish/plugins/core"
)

// Seed creates test data for the SendGrid plugin
func (p *SendGridPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var numAccounts, numMessagesPerAccount, numSuppressionsPerAccount int

	switch size {
	case "small":
		numAccounts, numMessagesPerAccount, numSuppressionsPerAccount = 1, 2, 1
	case "medium":
		numAccounts, numMessagesPerAccount, numSuppressionsPerAccount = 2, 5, 2
	case "large":
		numAccounts, numMessagesPerAccount, numSuppressionsPerAccount = 3, 10, 4
	default:
		numAccounts, numMessagesPerAccount, numSuppressionsPerAccount = 3, 5, 4
	}
	// Create test accounts
	accounts := []struct {
		email string
		name  string
	}{
		{"harper@example.com", "Harper"},
		{"alice@startup.io", "Alice Johnson"},
		{"bob@bigcorp.com", "Bob Smith"},
	}

	accountIDs := make([]int64, 0, len(accounts))
	for i := 0; i < numAccounts && i < len(accounts); i++ {
		acc := accounts[i]
		account, err := p.store.CreateAccount(acc.email, acc.name)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create account %s: %w", acc.email, err)
		}
		accountIDs = append(accountIDs, account.ID)
	}

	// Create API keys for each account
	totalAPIKeys := 0
	for i, accountID := range accountIDs {
		_, err := p.store.CreateAPIKey(
			accountID,
			fmt.Sprintf("Production API Key %d", i+1),
			"mail.send,messages.read",
		)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create API key for account %d: %w", accountID, err)
		}
		totalAPIKeys++

		// Create a second API key with limited scopes
		_, err = p.store.CreateAPIKey(
			accountID,
			fmt.Sprintf("Read-Only Key %d", i+1),
			"messages.read",
		)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create read-only API key for account %d: %w", accountID, err)
		}
		totalAPIKeys++
	}

	// Create sample messages for the first account
	messages := []struct {
		fromEmail   string
		fromName    string
		toEmail     string
		toName      string
		subject     string
		textContent string
		htmlContent string
	}{
		{
			fromEmail:   "noreply@example.com",
			fromName:    "Example App",
			toEmail:     "user1@test.com",
			toName:      "Test User 1",
			subject:     "Welcome to Example App!",
			textContent: "Thanks for signing up. We're excited to have you on board.",
			htmlContent: "<h1>Welcome!</h1><p>Thanks for signing up. We're excited to have you on board.</p>",
		},
		{
			fromEmail:   "support@example.com",
			fromName:    "Example Support",
			toEmail:     "user2@test.com",
			toName:      "Test User 2",
			subject:     "Password Reset Request",
			textContent: "Click here to reset your password: https://example.com/reset/abc123",
			htmlContent: "<p>Click <a href='https://example.com/reset/abc123'>here</a> to reset your password.</p>",
		},
		{
			fromEmail:   "marketing@example.com",
			fromName:    "Example Marketing",
			toEmail:     "user3@test.com",
			toName:      "Test User 3",
			subject:     "Check out our new features!",
			textContent: "We've just launched some amazing new features. Check them out at https://example.com/features",
			htmlContent: "<h2>New Features!</h2><p>We've just launched some amazing new features. <a href='https://example.com/features'>Check them out</a></p>",
		},
		{
			fromEmail:   "billing@example.com",
			fromName:    "Example Billing",
			toEmail:     "user4@test.com",
			toName:      "Test User 4",
			subject:     "Your invoice is ready",
			textContent: "Your monthly invoice for $99.00 is now available.",
			htmlContent: "<h2>Invoice Ready</h2><p>Your monthly invoice for <strong>$99.00</strong> is now available.</p>",
		},
		{
			fromEmail:   "notifications@example.com",
			fromName:    "Example Notifications",
			toEmail:     "user5@test.com",
			toName:      "Test User 5",
			subject:     "You have 3 new notifications",
			textContent: "Check your dashboard for 3 new notifications.",
			htmlContent: "<p>Check your <a href='https://example.com/dashboard'>dashboard</a> for 3 new notifications.</p>",
		},
	}

	totalMessages := 0
	for i := 0; i < numMessagesPerAccount && i < len(messages); i++ {
		msg := messages[i]
		_, err := p.store.CreateMessage(
			accountIDs[0],
			msg.fromEmail,
			msg.fromName,
			msg.toEmail,
			msg.toName,
			msg.subject,
			msg.textContent,
			msg.htmlContent,
		)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create message: %w", err)
		}
		totalMessages++
	}

	// Create some suppressions for the first account
	suppressions := []struct {
		email string
		stype string
		reason string
	}{
		{
			email:  "bounced@test.com",
			stype:  "bounce",
			reason: "550 5.1.1 The email account that you tried to reach does not exist",
		},
		{
			email:  "blocked@test.com",
			stype:  "block",
			reason: "Recipient has previously unsubscribed from this sender",
		},
		{
			email:  "spam@test.com",
			stype:  "spam_report",
			reason: "Recipient marked email as spam",
		},
		{
			email:  "invalid@test.com",
			stype:  "bounce",
			reason: "550 5.7.1 Unable to relay",
		},
	}

	totalSuppressions := 0
	for i := 0; i < numSuppressionsPerAccount && i < len(suppressions); i++ {
		supp := suppressions[i]
		_, err := p.store.CreateSuppression(
			accountIDs[0],
			supp.email,
			supp.stype,
			supp.reason,
		)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create suppression for %s: %w", supp.email, err)
		}
		totalSuppressions++
	}

	// Print API keys for testing
	fmt.Println("\n=== SendGrid Test API Keys ===")
	for i, accountID := range accountIDs {
		var apiKey string
		err := p.store.db.QueryRow(`
			SELECT key FROM sendgrid_api_keys
			WHERE account_id = ? AND name LIKE 'Production%'
			LIMIT 1
		`, accountID).Scan(&apiKey)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to retrieve API key for account %d: %w", accountID, err)
		}
		fmt.Printf("Account %d (%s): %s\n", i+1, accounts[i].email, apiKey)
	}
	fmt.Println()

	return core.SeedData{
		Summary: fmt.Sprintf("Created %d accounts, %d API keys, %d messages, %d suppressions",
			len(accountIDs), totalAPIKeys, totalMessages, totalSuppressions),
		Records: map[string]int{
			"accounts":     len(accountIDs),
			"api_keys":     totalAPIKeys,
			"messages":     totalMessages,
			"suppressions": totalSuppressions,
		},
	}, nil
}
