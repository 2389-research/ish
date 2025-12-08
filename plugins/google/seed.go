// ABOUTME: Test data generation for Google plugin (Gmail, Calendar, People, Tasks)
// ABOUTME: Creates sample messages, events, contacts, and tasks for testing

package google

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/2389/ish/internal/seed"
	"github.com/2389/ish/plugins/core"
)

// Seed creates test data for the Google plugin using AI by default
func (p *GooglePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var numMessages, numEvents, numPeople, numTasks int

	switch size {
	case "small":
		numMessages, numEvents, numPeople, numTasks = 3, 2, 2, 2
	case "medium":
		numMessages, numEvents, numPeople, numTasks = 8, 5, 5, 5
	case "large":
		numMessages, numEvents, numPeople, numTasks = 15, 10, 10, 10
	default:
		numMessages, numEvents, numPeople, numTasks = 8, 5, 5, 5
	}

	userID := "me"

	// Try AI generation first (default behavior)
	generator := seed.NewGenerator(userID)
	genData, err := generator.Generate(ctx, numMessages, numEvents, numPeople)
	if err == nil && len(genData.Emails) > 0 {
		// AI generation succeeded, use it
		return p.seedFromAI(ctx, userID, genData, numTasks)
	}

	// Fall back to static data if AI generation fails or OPENAI_API_KEY not set
	log.Println("Using static seed data for Google plugin")
	return p.seedStatic(ctx, userID, numMessages, numEvents, numPeople, numTasks)
}

// seedFromAI creates seed data using AI-generated content
func (p *GooglePlugin) seedFromAI(ctx context.Context, userID string, genData *seed.GeneratedData, numTasks int) (core.SeedData, error) {
	totalMessages := 0
	totalEvents := 0
	totalPeople := 0
	totalTasks := 0

	// Create messages from AI data
	for _, email := range genData.Emails {
		_, err := p.store.CreateGmailMessageFromForm(userID, email.From, email.Subject, email.Body, email.Labels)
		if err != nil {
			log.Printf("Failed to create AI message: %v", err)
			continue
		}
		totalMessages++
	}

	// Create events from AI data
	for _, event := range genData.Events {
		_, err := p.store.CreateCalendarEventFromForm(event.Summary, event.Description, event.StartTime, event.EndTime)
		if err != nil {
			log.Printf("Failed to create AI event: %v", err)
			continue
		}
		totalEvents++
	}

	// Create contacts from AI data
	for _, contact := range genData.Contacts {
		_, err := p.store.CreatePersonFromForm(userID, contact.Name, contact.Email)
		if err != nil {
			log.Printf("Failed to create AI contact: %v", err)
			continue
		}
		totalPeople++
	}

	// Create tasks (still using static data for now)
	taskLists := []struct {
		title string
		tasks []string
	}{
		{"Work", []string{"Review pull requests", "Update documentation", "Fix critical bug"}},
		{"Personal", []string{"Buy groceries", "Call dentist"}},
	}

	for _, list := range taskLists {
		if totalTasks >= numTasks {
			break
		}
		taskListObj := &TaskList{
			UserID: userID,
			Title:  list.title,
		}
		if err := p.store.CreateTaskList(taskListObj); err != nil {
			continue
		}

		for _, title := range list.tasks {
			if totalTasks >= numTasks {
				break
			}
			_, err := p.store.CreateTaskFromForm(title, "", "", "needsAction")
			if err != nil {
				continue
			}
			totalTasks++
		}
	}

	return core.SeedData{
		Summary: fmt.Sprintf("Created %d AI-generated messages, %d events, %d contacts, %d tasks",
			totalMessages, totalEvents, totalPeople, totalTasks),
		Records: map[string]int{
			"messages": totalMessages,
			"events":   totalEvents,
			"contacts": totalPeople,
			"tasks":    totalTasks,
		},
	}, nil
}

// seedStatic creates seed data using static hardcoded content
func (p *GooglePlugin) seedStatic(ctx context.Context, userID string, numMessages, numEvents, numPeople, numTasks int) (core.SeedData, error) {

	// === Gmail Messages ===
	messages := []struct {
		from    string
		subject string
		body    string
		labels  []string
	}{
		{
			from:    "team@example.com",
			subject: "Weekly Team Standup - Monday 9am",
			body:    "Hi team,\n\nJust a reminder about our weekly standup tomorrow at 9am. Please come prepared with your updates.\n\nAgenda:\n- Sprint progress\n- Blockers\n- Next week planning\n\nSee you there!",
			labels:  []string{"INBOX", "IMPORTANT"},
		},
		{
			from:    "noreply@github.com",
			subject: "[GitHub] Pull Request #42 merged",
			body:    "Your pull request 'feat: add user authentication' has been successfully merged into main by @harper.\n\nChanges:\n- Add JWT authentication\n- Update user schema\n- Add login/logout endpoints",
			labels:  []string{"INBOX"},
		},
		{
			from:    "billing@aws.amazon.com",
			subject: "Your AWS Invoice for December 2024",
			body:    "Your AWS bill for December 2024 is ready.\n\nTotal: $127.43\n\nBreakdown:\n- EC2: $45.20\n- S3: $12.15\n- RDS: $70.08\n\nView full details in the AWS Console.",
			labels:  []string{"INBOX"},
		},
		{
			from:    "alerts@datadog.com",
			subject: "[CRITICAL] High CPU usage on prod-api-1",
			body:    "ALERT: CPU usage on prod-api-1 has exceeded 90% for the past 15 minutes.\n\nCurrent: 94%\nThreshold: 90%\n\nPlease investigate immediately.",
			labels:  []string{"INBOX", "IMPORTANT"},
		},
		{
			from:    "calendar-notification@google.com",
			subject: "Event reminder: Product Demo",
			body:    "This is a reminder that your event 'Product Demo' starts in 1 hour at 2:00 PM.\n\nLocation: Conference Room A\nDuration: 1 hour",
			labels:  []string{"INBOX"},
		},
		{
			from:    "support@stripe.com",
			subject: "Successful payment of $49.99",
			body:    "Your payment of $49.99 has been processed successfully.\n\nCustomer: Acme Corp\nDescription: Monthly Pro Plan\nPayment method: Visa ending in 4242",
			labels:  []string{"INBOX"},
		},
		{
			from:    "no-reply@slack.com",
			subject: "You have 12 unread mentions in #engineering",
			body:    "You have unread activity in your Slack workspace.\n\n#engineering: 12 mentions\n#general: 3 mentions\n\nCatch up now: https://acme.slack.com",
			labels:  []string{"INBOX"},
		},
		{
			from:    "security@okta.com",
			subject: "New sign-in from San Francisco, CA",
			body:    "We detected a new sign-in to your account.\n\nLocation: San Francisco, CA\nDevice: Chrome on macOS\nTime: 2:34 PM PST\n\nIf this wasn't you, please secure your account immediately.",
			labels:  []string{"INBOX", "IMPORTANT"},
		},
		{
			from:    "notifications@linear.app",
			subject: "ISH-123 assigned to you",
			body:    "Harper assigned issue ISH-123 to you.\n\nTitle: Implement Google Calendar sync\nPriority: High\nDue: Friday\n\nView issue: https://linear.app/acme/issue/ISH-123",
			labels:  []string{"INBOX"},
		},
		{
			from:    "news@techcrunch.com",
			subject: "TC Daily: AI startup raises $100M Series B",
			body:    "Top stories today:\n\n1. AI startup Anthropic raises $100M\n2. GitHub launches new code search\n3. Tesla opens new factory in Austin\n\nRead more at techcrunch.com",
			labels:  []string{"INBOX"},
		},
		{
			from:    "hr@company.com",
			subject: "Benefits Enrollment Reminder",
			body:    "Reminder: Benefits enrollment closes this Friday.\n\nPlease review and select your:\n- Health insurance\n- Dental coverage\n- 401(k) contribution\n\nComplete enrollment at: hr.company.com/benefits",
			labels:  []string{"INBOX"},
		},
		{
			from:    "events@meetup.com",
			subject: "RSVP Confirmation: Go SF Meetup",
			body:    "You're confirmed for Go SF Meetup!\n\nWhen: Thursday, 6:30 PM\nWhere: WeWork SoMa\n\nTalks:\n- Building high-performance APIs\n- Concurrency patterns in Go\n\nSee you there!",
			labels:  []string{"INBOX"},
		},
		{
			from:    "orders@amazon.com",
			subject: "Your package has been delivered",
			body:    "Your package was delivered at 3:42 PM.\n\nOrder #123-4567890-1234567\nDelivered to: Front door\n\nItems:\n- Mechanical Keyboard\n\nRate your delivery experience.",
			labels:  []string{"INBOX"},
		},
		{
			from:    "analytics@mixpanel.com",
			subject: "Weekly Report: User engagement up 23%",
			body:    "Your weekly analytics report is ready.\n\nHighlights:\n- Active users: +23%\n- Session duration: +15%\n- Conversion rate: 3.2%\n\nView full report: mixpanel.com/reports/weekly",
			labels:  []string{"INBOX"},
		},
		{
			from:    "noreply@docker.com",
			subject: "Security vulnerability detected in base image",
			body:    "We've detected a high-severity vulnerability in your base image.\n\nImage: node:14-alpine\nCVE: CVE-2024-12345\nSeverity: High\n\nUpdate to node:14.21.2-alpine to fix.",
			labels:  []string{"INBOX", "IMPORTANT"},
		},
	}

	totalMessages := 0
	for i := 0; i < numMessages && i < len(messages); i++ {
		msg := messages[i]
		_, err := p.store.CreateGmailMessageFromForm(userID, msg.from, msg.subject, msg.body, msg.labels)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create message: %w", err)
		}
		totalMessages++
	}

	// === Calendar Events ===

	// First, ensure user has a calendar
	calendarID := "primary"
	calendar := &Calendar{
		ID:      calendarID,
		UserID:  userID,
		Summary: "Primary Calendar",
	}
	if err := p.store.CreateCalendar(calendar); err != nil {
		// Calendar might already exist, that's ok
	}

	now := time.Now()
	events := []struct {
		summary     string
		description string
		start       time.Time
		end         time.Time
		location    string
	}{
		{
			summary:     "Team Standup",
			description: "Daily sync with the engineering team",
			start:       now.Add(24 * time.Hour).Truncate(time.Hour).Add(9 * time.Hour),
			end:         now.Add(24 * time.Hour).Truncate(time.Hour).Add(9*time.Hour + 30*time.Minute),
			location:    "Zoom",
		},
		{
			summary:     "Product Demo",
			description: "Demo new features to stakeholders",
			start:       now.Add(48 * time.Hour).Truncate(time.Hour).Add(14 * time.Hour),
			end:         now.Add(48 * time.Hour).Truncate(time.Hour).Add(15 * time.Hour),
			location:    "Conference Room A",
		},
		{
			summary:     "1:1 with Manager",
			description: "Weekly check-in",
			start:       now.Add(72 * time.Hour).Truncate(time.Hour).Add(15 * time.Hour),
			end:         now.Add(72 * time.Hour).Truncate(time.Hour).Add(16 * time.Hour),
			location:    "Office",
		},
		{
			summary:     "Sprint Planning",
			description: "Plan work for next 2-week sprint",
			start:       now.Add(96 * time.Hour).Truncate(time.Hour).Add(10 * time.Hour),
			end:         now.Add(96 * time.Hour).Truncate(time.Hour).Add(12 * time.Hour),
			location:    "Conference Room B",
		},
		{
			summary:     "Code Review Session",
			description: "Review PRs with the team",
			start:       now.Add(120 * time.Hour).Truncate(time.Hour).Add(13 * time.Hour),
			end:         now.Add(120 * time.Hour).Truncate(time.Hour).Add(14 * time.Hour),
			location:    "Zoom",
		},
		{
			summary:     "Customer Interview",
			description: "User research session with beta customer",
			start:       now.Add(144 * time.Hour).Truncate(time.Hour).Add(11 * time.Hour),
			end:         now.Add(144 * time.Hour).Truncate(time.Hour).Add(12 * time.Hour),
			location:    "Google Meet",
		},
		{
			summary:     "Architecture Review",
			description: "Review database migration plan",
			start:       now.Add(168 * time.Hour).Truncate(time.Hour).Add(14 * time.Hour),
			end:         now.Add(168 * time.Hour).Truncate(time.Hour).Add(15*time.Hour + 30*time.Minute),
			location:    "Conference Room C",
		},
		{
			summary:     "Team Lunch",
			description: "Monthly team building lunch",
			start:       now.Add(192 * time.Hour).Truncate(time.Hour).Add(12 * time.Hour),
			end:         now.Add(192 * time.Hour).Truncate(time.Hour).Add(13 * time.Hour),
			location:    "Pizzeria Locale",
		},
		{
			summary:     "All Hands Meeting",
			description: "Company-wide quarterly update",
			start:       now.Add(216 * time.Hour).Truncate(time.Hour).Add(16 * time.Hour),
			end:         now.Add(216 * time.Hour).Truncate(time.Hour).Add(17 * time.Hour),
			location:    "Main Auditorium",
		},
		{
			summary:     "Deploy to Production",
			description: "Release v2.5.0 to production",
			start:       now.Add(240 * time.Hour).Truncate(time.Hour).Add(17 * time.Hour),
			end:         now.Add(240 * time.Hour).Truncate(time.Hour).Add(18 * time.Hour),
			location:    "Remote",
		},
	}

	totalEvents := 0
	for i := 0; i < numEvents && i < len(events); i++ {
		evt := events[i]
		event := &CalendarEvent{
			ID:             fmt.Sprintf("evt_%d", time.Now().UnixNano()+int64(i)),
			CalendarID:     calendarID,
			Summary:        evt.summary,
			Description:    evt.description,
			StartTime:      evt.start.Format(time.RFC3339),
			EndTime:        evt.end.Format(time.RFC3339),
			Location:       evt.location,
			OrganizerEmail: "harper@example.com",
			OrganizerName:  "Harper",
			UpdatedAt:      time.Now().Format(time.RFC3339),
		}
		_, err := p.store.CreateCalendarEvent(event)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create event: %w", err)
		}
		totalEvents++
	}

	// === People/Contacts ===
	people := []struct {
		name  string
		email string
		data  string
	}{
		{
			name:  "Alice Johnson",
			email: "alice@example.com",
			data:  `{"names":[{"displayName":"Alice Johnson"}],"emailAddresses":[{"value":"alice@example.com"}],"phoneNumbers":[{"value":"+1-555-0101"}],"organizations":[{"name":"Acme Corp","title":"Engineering Manager"}]}`,
		},
		{
			name:  "Bob Smith",
			email: "bob@startup.io",
			data:  `{"names":[{"displayName":"Bob Smith"}],"emailAddresses":[{"value":"bob@startup.io"}],"phoneNumbers":[{"value":"+1-555-0102"}],"organizations":[{"name":"StartupIO","title":"CTO"}]}`,
		},
		{
			name:  "Carol Williams",
			email: "carol@bigcorp.com",
			data:  `{"names":[{"displayName":"Carol Williams"}],"emailAddresses":[{"value":"carol@bigcorp.com"}],"phoneNumbers":[{"value":"+1-555-0103"}],"organizations":[{"name":"BigCorp Inc","title":"Product Manager"}]}`,
		},
		{
			name:  "David Brown",
			email: "david@consulting.com",
			data:  `{"names":[{"displayName":"David Brown"}],"emailAddresses":[{"value":"david@consulting.com"}],"phoneNumbers":[{"value":"+1-555-0104"}],"organizations":[{"name":"Brown Consulting","title":"Senior Consultant"}]}`,
		},
		{
			name:  "Eve Davis",
			email: "eve@techventures.com",
			data:  `{"names":[{"displayName":"Eve Davis"}],"emailAddresses":[{"value":"eve@techventures.com"}],"phoneNumbers":[{"value":"+1-555-0105"}],"organizations":[{"name":"Tech Ventures","title":"VP Engineering"}]}`,
		},
		{
			name:  "Frank Miller",
			email: "frank@cloudservices.io",
			data:  `{"names":[{"displayName":"Frank Miller"}],"emailAddresses":[{"value":"frank@cloudservices.io"}],"phoneNumbers":[{"value":"+1-555-0106"}],"organizations":[{"name":"Cloud Services","title":"DevOps Lead"}]}`,
		},
		{
			name:  "Grace Lee",
			email: "grace@designstudio.com",
			data:  `{"names":[{"displayName":"Grace Lee"}],"emailAddresses":[{"value":"grace@designstudio.com"}],"phoneNumbers":[{"value":"+1-555-0107"}],"organizations":[{"name":"Design Studio","title":"Lead Designer"}]}`,
		},
		{
			name:  "Henry Wilson",
			email: "henry@dataanalytics.com",
			data:  `{"names":[{"displayName":"Henry Wilson"}],"emailAddresses":[{"value":"henry@dataanalytics.com"}],"phoneNumbers":[{"value":"+1-555-0108"}],"organizations":[{"name":"Data Analytics Co","title":"Data Scientist"}]}`,
		},
		{
			name:  "Iris Chen",
			email: "iris@mobilefirst.io",
			data:  `{"names":[{"displayName":"Iris Chen"}],"emailAddresses":[{"value":"iris@mobilefirst.io"}],"phoneNumbers":[{"value":"+1-555-0109"}],"organizations":[{"name":"Mobile First","title":"iOS Developer"}]}`,
		},
		{
			name:  "Jack Taylor",
			email: "jack@securitypro.com",
			data:  `{"names":[{"displayName":"Jack Taylor"}],"emailAddresses":[{"value":"jack@securitypro.com"}],"phoneNumbers":[{"value":"+1-555-0110"}],"organizations":[{"name":"SecurityPro","title":"Security Engineer"}]}`,
		},
	}

	totalPeople := 0
	for i := 0; i < numPeople && i < len(people); i++ {
		personData := people[i]
		person := &Person{
			ResourceName: fmt.Sprintf("people/person_%d", time.Now().UnixNano()+int64(i)),
			UserID:       userID,
			Data:         personData.data,
		}
		err := p.store.CreatePerson(person)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create person: %w", err)
		}
		totalPeople++
	}

	// === Tasks ===

	// First, ensure user has a task list
	listID := "default"
	taskList := &TaskList{
		ID:        listID,
		UserID:    userID,
		Title:     "My Tasks",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	if err := p.store.CreateTaskList(taskList); err != nil {
		// List might already exist, that's ok
	}

	tasks := []struct {
		title  string
		notes  string
		due    string
		status string
	}{
		{
			title:  "Review pull requests",
			notes:  "Check PRs from Alice and Bob before EOD",
			due:    now.Add(24 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Update API documentation",
			notes:  "Add examples for new authentication endpoints",
			due:    now.Add(48 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Fix bug in user profile page",
			notes:  "Users report that avatar images aren't loading correctly",
			due:    now.Add(72 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Prepare demo for stakeholders",
			notes:  "Create slides and practice presentation",
			due:    now.Add(96 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Optimize database queries",
			notes:  "Profile slow queries and add indexes where needed",
			due:    now.Add(120 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Write unit tests for payment service",
			notes:  "Need 80% coverage before shipping to prod",
			due:    now.Add(144 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Schedule team retrospective",
			notes:  "Find time that works for everyone next week",
			due:    now.Add(168 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Upgrade React to latest version",
			notes:  "Test thoroughly in staging before deploying",
			due:    now.Add(192 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Research GraphQL migration options",
			notes:  "Evaluate Apollo vs. Relay for our use case",
			due:    now.Add(216 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
		{
			title:  "Onboard new team member",
			notes:  "Set up laptop, accounts, and schedule intro meetings",
			due:    now.Add(240 * time.Hour).Format(time.RFC3339),
			status: "needsAction",
		},
	}

	totalTasks := 0
	for i := 0; i < numTasks && i < len(tasks); i++ {
		t := tasks[i]
		task := &Task{
			ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()+int64(i)),
			ListID:    listID,
			Title:     t.title,
			Notes:     t.notes,
			Due:       t.due,
			Status:    t.status,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		_, err := p.store.CreateTask(task)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create task: %w", err)
		}
		totalTasks++
	}

	return core.SeedData{
		Summary: fmt.Sprintf("Created %d messages, %d events, %d contacts, %d tasks",
			totalMessages, totalEvents, totalPeople, totalTasks),
		Records: map[string]int{
			"messages": totalMessages,
			"events":   totalEvents,
			"contacts": totalPeople,
			"tasks":    totalTasks,
		},
	}, nil
}
