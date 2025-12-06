// ABOUTME: Test data seeding for Twilio plugin
// ABOUTME: Generates realistic accounts, phone numbers, messages, and calls

package twilio

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/2389/ish/plugins/core"
)

func (p *TwilioPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var accounts, phoneNumbers, messages, calls int

	switch size {
	case "small":
		accounts, phoneNumbers, messages, calls = 1, 3, 10, 5
	case "medium":
		accounts, phoneNumbers, messages, calls = 3, 10, 50, 20
	case "large":
		accounts, phoneNumbers, messages, calls = 10, 30, 200, 100
	default:
		accounts, phoneNumbers, messages, calls = 1, 3, 10, 5
	}

	// Create accounts
	accountSids := make([]string, accounts)
	for i := 0; i < accounts; i++ {
		sid := fmt.Sprintf("AC%032d", i+1)
		accountSids[i] = sid
		if _, err := p.store.GetOrCreateAccount(sid); err != nil {
			return core.SeedData{}, err
		}
	}

	// Create phone numbers
	phoneNumberList := make([]string, phoneNumbers)
	numbersPerAccount := phoneNumbers / accounts
	for i := 0; i < phoneNumbers; i++ {
		accountIdx := i / numbersPerAccount
		if accountIdx >= accounts {
			accountIdx = accounts - 1
		}

		phoneNum := fmt.Sprintf("+1555%07d", 1000000+i)
		phoneNumberList[i] = phoneNum
		friendlyName := fmt.Sprintf("Test Number %d", i+1)

		if _, err := p.store.CreatePhoneNumber(accountSids[accountIdx], phoneNum, friendlyName); err != nil {
			return core.SeedData{}, err
		}
	}

	// Create messages
	messageBodies := []string{
		"Your verification code is 123456",
		"Your package has been shipped",
		"Reminder: You have an appointment tomorrow at 2 PM",
		"Thanks for your order! Order #12345 is confirmed",
		"Your account balance is $100.00",
		"Hello! How can we help you today?",
		"Your reservation is confirmed for 6 PM",
		"Security alert: New login from Chrome",
	}

	for i := 0; i < messages; i++ {
		accountIdx := rand.Intn(accounts)
		fromIdx := rand.Intn(phoneNumbers)
		toPhone := fmt.Sprintf("+1555%07d", 2000000+rand.Intn(1000000))
		body := messageBodies[rand.Intn(len(messageBodies))]

		msg, err := p.store.CreateMessage(accountSids[accountIdx], phoneNumberList[fromIdx], toPhone, body)
		if err != nil {
			return core.SeedData{}, err
		}

		// Set some messages to delivered status
		if i%2 == 0 {
			p.store.UpdateMessageStatus(msg.Sid, "delivered")
		} else if i%3 == 0 {
			p.store.UpdateMessageStatus(msg.Sid, "sent")
		}
	}

	// Create calls
	for i := 0; i < calls; i++ {
		accountIdx := rand.Intn(accounts)
		fromIdx := rand.Intn(phoneNumbers)
		toPhone := fmt.Sprintf("+1555%07d", 3000000+rand.Intn(1000000))

		call, err := p.store.CreateCall(accountSids[accountIdx], phoneNumberList[fromIdx], toPhone)
		if err != nil {
			return core.SeedData{}, err
		}

		// Set some calls to completed with duration
		if i%2 == 0 {
			duration := 30 + rand.Intn(570) // 30-600 seconds
			p.store.UpdateCallStatus(call.Sid, "completed", &duration)
		} else if i%3 == 0 {
			p.store.UpdateCallStatus(call.Sid, "in-progress", nil)
		}
	}

	summary := fmt.Sprintf("Created %d accounts, %d phone numbers, %d messages, %d calls",
		accounts, phoneNumbers, messages, calls)

	return core.SeedData{
		Summary: summary,
		Records: map[string]int{
			"accounts":      accounts,
			"phone_numbers": phoneNumbers,
			"messages":      messages,
			"calls":         calls,
		},
	}, nil
}
