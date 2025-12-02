// ABOUTME: AI-powered data generator for realistic fake Google API data.
// ABOUTME: Uses OpenAI gpt-5-nano to generate emails, events, and contacts.

package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
)

// Generator creates fake data using OpenAI or falls back to static data.
type Generator struct {
	client  *openai.Client
	useAI   bool
	userID  string
	model   string
}

// NewGenerator creates a generator, loading API key from .env if available.
func NewGenerator(userID string) *Generator {
	g := &Generator{userID: userID}

	// Try to load .env from current dir or parent dirs
	envPaths := []string{".env", "../.env", "../../.env"}
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	// Also check home directory
	if home, err := os.UserHomeDir(); err == nil {
		godotenv.Load(filepath.Join(home, ".env"))
	}

	// Get model from env, default to gpt-5-mini
	g.model = os.Getenv("OPENAI_MODEL")
	if g.model == "" {
		g.model = "gpt-5-mini"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		g.client = openai.NewClient(apiKey)
		g.useAI = true
		log.Printf("OpenAI API key found, using AI-generated data with model: %s", g.model)
	} else {
		log.Println("No OPENAI_API_KEY found, using static fallback data")
	}

	return g
}

// GeneratedData holds all the generated fake data.
type GeneratedData struct {
	Emails   []EmailData   `json:"emails"`
	Events   []EventData   `json:"events"`
	Contacts []ContactData `json:"contacts"`
}

// EmailData represents a generated email.
type EmailData struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Labels  []string `json:"labels"`
}

// EventData represents a generated calendar event.
type EventData struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	Attendees   []string `json:"attendees"`
}

// ContactData represents a generated contact.
type ContactData struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Company string `json:"company"`
}

// Generate creates all the fake data.
func (g *Generator) Generate(ctx context.Context, numEmails, numEvents, numContacts int) (*GeneratedData, error) {
	if !g.useAI {
		return g.generateStatic(numEmails, numEvents, numContacts), nil
	}

	data := &GeneratedData{}

	type result struct {
		name string
		err  error
	}

	// Generate in parallel for speed
	resultCh := make(chan result, 3)

	log.Printf("Generating %d emails, %d events, %d contacts via AI...", numEmails, numEvents, numContacts)

	go func() {
		log.Print("  ⏳ Generating emails...")
		emails, err := g.generateEmails(ctx, numEmails)
		if err != nil {
			resultCh <- result{"emails", err}
			return
		}
		data.Emails = emails
		log.Printf("  ✓ Generated %d emails", len(emails))
		resultCh <- result{"emails", nil}
	}()

	go func() {
		log.Print("  ⏳ Generating calendar events...")
		events, err := g.generateEvents(ctx, numEvents)
		if err != nil {
			resultCh <- result{"events", err}
			return
		}
		data.Events = events
		log.Printf("  ✓ Generated %d events", len(events))
		resultCh <- result{"events", nil}
	}()

	go func() {
		log.Print("  ⏳ Generating contacts...")
		contacts, err := g.generateContacts(ctx, numContacts)
		if err != nil {
			resultCh <- result{"contacts", err}
			return
		}
		data.Contacts = contacts
		log.Printf("  ✓ Generated %d contacts", len(contacts))
		resultCh <- result{"contacts", nil}
	}()

	// Collect results
	var errs []error
	for i := 0; i < 3; i++ {
		r := <-resultCh
		if r.err != nil {
			log.Printf("  ✗ Failed to generate %s: %v", r.name, r.err)
			errs = append(errs, fmt.Errorf("%s: %w", r.name, r.err))
		}
	}

	if len(errs) > 0 {
		log.Print("AI generation incomplete, falling back to static data...")
		return g.generateStatic(numEmails, numEvents, numContacts), nil
	}

	log.Print("AI generation complete!")
	return data, nil
}

// GenerateSingleEmail creates one realistic email using AI or static fallback.
func (g *Generator) GenerateSingleEmail(ctx context.Context) (*EmailData, error) {
	if !g.useAI {
		static := generateStaticEmails(1)
		return &static[0], nil
	}

	emails, err := g.generateEmails(ctx, 1)
	if err != nil {
		static := generateStaticEmails(1)
		return &static[0], nil
	}
	if len(emails) == 0 {
		static := generateStaticEmails(1)
		return &static[0], nil
	}
	return &emails[0], nil
}

// GenerateSingleEvent creates one realistic calendar event using AI or static fallback.
func (g *Generator) GenerateSingleEvent(ctx context.Context) (*EventData, error) {
	if !g.useAI {
		static := generateStaticEvents(1)
		return &static[0], nil
	}

	events, err := g.generateEvents(ctx, 1)
	if err != nil {
		static := generateStaticEvents(1)
		return &static[0], nil
	}
	if len(events) == 0 {
		static := generateStaticEvents(1)
		return &static[0], nil
	}
	return &events[0], nil
}

// GenerateSingleContact creates one realistic contact using AI or static fallback.
func (g *Generator) GenerateSingleContact(ctx context.Context) (*ContactData, error) {
	if !g.useAI {
		static := generateStaticContacts(1)
		return &static[0], nil
	}

	contacts, err := g.generateContacts(ctx, 1)
	if err != nil {
		static := generateStaticContacts(1)
		return &static[0], nil
	}
	if len(contacts) == 0 {
		static := generateStaticContacts(1)
		return &static[0], nil
	}
	return &contacts[0], nil
}

func (g *Generator) generateEmails(ctx context.Context, count int) ([]EmailData, error) {
	prompt := fmt.Sprintf(`Generate %d realistic fake emails for a professional's inbox. Include a mix of:
- Work emails (meetings, project updates, feedback)
- Newsletter subscriptions
- Personal emails from friends/family
- Automated notifications (shipping, receipts, alerts)
- Some spam that got through

Return as JSON array with objects containing: from, to, subject, body, labels (array of INBOX, UNREAD, STARRED, IMPORTANT, SENT, SPAM, TRASH).
About 40%% should have UNREAD label. The "to" field should always be "harper@example.com".
Make the content realistic and varied. Each body should be 2-4 sentences.`, count)

	return callOpenAI[[]EmailData](ctx, g.client, g.model, prompt)
}

func (g *Generator) generateEvents(ctx context.Context, count int) ([]EventData, error) {
	now := time.Now()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, 30).Format("2006-01-02")

	prompt := fmt.Sprintf(`Generate %d realistic calendar events for a professional between %s and %s. Include:
- Team meetings and standups
- 1:1s with colleagues
- External meetings with clients/vendors
- Personal appointments (doctor, dentist, car service)
- Social events (lunch, coffee, dinner)
- Reminders and blocks (focus time, workout)

Return as JSON array with objects containing: summary, description, start_time (ISO 8601), end_time (ISO 8601), attendees (array of email addresses).
Distribute events throughout the date range. Use realistic times (business hours mostly, some evening events).
Each event should have 0-4 attendees. Include harper@example.com as an attendee where appropriate.`, count, startDate, endDate)

	return callOpenAI[[]EventData](ctx, g.client, g.model, prompt)
}

func (g *Generator) generateContacts(ctx context.Context, count int) ([]ContactData, error) {
	prompt := fmt.Sprintf(`Generate %d realistic fake contacts for a professional's address book. Include:
- Work colleagues (same company domain)
- External business contacts (clients, vendors, partners)
- Friends and family
- Service providers (doctor, accountant, etc.)

Return as JSON array with objects containing: name, email, phone, company.
Use diverse but realistic names. Phone numbers should be US format (555-XXX-XXXX).
Some contacts may have empty phone or company fields.`, count)

	return callOpenAI[[]ContactData](ctx, g.client, g.model, prompt)
}

func callOpenAI[T any](ctx context.Context, client *openai.Client, model, prompt string) (T, error) {
	var result T

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a data generator. Always respond with valid JSON only, no markdown or explanation.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return result, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return result, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return result, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}
