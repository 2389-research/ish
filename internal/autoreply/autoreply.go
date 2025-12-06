// ABOUTME: Auto-reply feature for sent emails using OpenAI.
// ABOUTME: Generates realistic email responses with random delays.

package autoreply

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GmailMessageSender is the interface for sending Gmail messages
type GmailMessageSender interface {
	SendGmailMessage(userID, from, to, subject, body string) (any, error)
}

// AutoReply handles automatic email responses
type AutoReply struct {
	store     GmailMessageSender
	openaiKey string
	enabled   bool
	minDelay  int
	maxDelay  int
	templates []string
}

// New creates a new AutoReply instance
func New(s GmailMessageSender) *AutoReply {
	enabled := os.Getenv("ISH_AUTO_REPLY") == "true"
	openaiKey := os.Getenv("ISH_OPENAI_KEY")

	minDelay := 2
	if val := os.Getenv("ISH_REPLY_DELAY_MIN"); val != "" {
		fmt.Sscanf(val, "%d", &minDelay)
	}

	maxDelay := 30
	if val := os.Getenv("ISH_REPLY_DELAY_MAX"); val != "" {
		fmt.Sscanf(val, "%d", &maxDelay)
	}

	return &AutoReply{
		store:     s,
		openaiKey: openaiKey,
		enabled:   enabled,
		minDelay:  minDelay,
		maxDelay:  maxDelay,
		templates: []string{
			"Thanks for your email! I'll get back to you shortly.",
			"Got it, I'll take a look at this.",
			"Sounds good, let's sync up soon.",
			"Thanks for reaching out! I'll review this and follow up.",
			"Received, thanks! I'll get this taken care of.",
		},
	}
}

// GenerateReply generates an auto-reply for a sent message
func (ar *AutoReply) GenerateReply(userID, from, to, subject, body string, threadID string) {
	if !ar.enabled {
		return
	}

	// Run in background goroutine
	go func() {
		// Random delay
		delay := time.Duration(ar.minDelay+rand.Intn(ar.maxDelay-ar.minDelay+1)) * time.Second
		time.Sleep(delay)

		// Generate reply content
		var replyBody string
		if ar.openaiKey != "" {
			var err error
			replyBody, err = ar.generateWithOpenAI(subject, from, body)
			if err != nil {
				log.Printf("OpenAI generation failed, using template: %v", err)
				replyBody = ar.getRandomTemplate()
			}
		} else {
			replyBody = ar.getRandomTemplate()
		}

		// Create reply message
		replySubject := subject
		if len(subject) > 3 && subject[:3] != "Re:" {
			replySubject = "Re: " + subject
		}

		_, err := ar.store.SendGmailMessage(userID, to, from, replySubject, replyBody)
		if err != nil {
			log.Printf("Failed to create auto-reply: %v", err)
		} else {
			log.Printf("Auto-reply sent from %s to %s", to, from)
		}
	}()
}

func (ar *AutoReply) generateWithOpenAI(subject, from, body string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := openai.NewClient(ar.openaiKey)

	prompt := fmt.Sprintf(`You received an email with:
From: %s
Subject: %s
Body: %s

Generate a realistic, professional email reply that:
- Acknowledges the email
- Provides a helpful response
- Maintains appropriate tone
- Keeps it concise (2-4 sentences)

Reply (body only, no greeting or signature):`, from, subject, body)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   150,
		Temperature: 0.7,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

func (ar *AutoReply) getRandomTemplate() string {
	return ar.templates[rand.Intn(len(ar.templates))]
}
