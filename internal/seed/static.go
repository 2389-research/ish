// ABOUTME: Static fallback data when OpenAI API key is not available.
// ABOUTME: Provides a diverse set of realistic-looking fake data.

package seed

import (
	"fmt"
	"time"
)

// generateStatic creates static fallback data.
func (g *Generator) generateStatic(numEmails, numEvents, numContacts int) *GeneratedData {
	data := &GeneratedData{
		Emails:   generateStaticEmails(numEmails),
		Events:   generateStaticEvents(numEvents),
		Contacts: generateStaticContacts(numContacts),
	}
	return data
}

func generateStaticEmails(count int) []EmailData {
	templates := []EmailData{
		{From: "alice.chen@techcorp.com", To: "harper@example.com", Subject: "Q4 Planning Meeting", Body: "Hi Harper, just wanted to confirm our Q4 planning session for next week. I've prepared the preliminary budget forecasts. Let me know if you need any additional data before the meeting.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "notifications@github.com", To: "harper@example.com", Subject: "[ish] PR #42 merged", Body: "The pull request 'Add Gmail query syntax support' has been merged into main. Great work on this feature!", Labels: []string{"INBOX"}},
		{From: "bob.martinez@acmeinc.com", To: "harper@example.com", Subject: "Re: Partnership Proposal", Body: "Thanks for sending over the proposal. Our legal team is reviewing it now. I expect we'll have comments by end of week. Looking forward to moving this forward.", Labels: []string{"INBOX", "STARRED", "UNREAD"}},
		{From: "noreply@amazon.com", To: "harper@example.com", Subject: "Your order has shipped!", Body: "Great news! Your order #112-4567890-1234567 has shipped and is on its way. Expected delivery: Wednesday.", Labels: []string{"INBOX"}},
		{From: "sarah.johnson@techcorp.com", To: "harper@example.com", Subject: "Team lunch Friday?", Body: "Hey! A few of us are planning to grab lunch at that new Thai place on Friday. Want to join? We're thinking around noon.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "newsletter@hackernews.com", To: "harper@example.com", Subject: "Weekly Digest: Top Stories", Body: "This week's top stories include: Rust 2.0 announced, AI breakthrough in protein folding, and why Postgres is still king.", Labels: []string{"INBOX"}},
		{From: "mom@gmail.com", To: "harper@example.com", Subject: "Sunday dinner?", Body: "Hi sweetie! Dad and I were wondering if you'd like to come over for dinner this Sunday. I'm making your favorite lasagna. Let me know!", Labels: []string{"INBOX", "IMPORTANT", "UNREAD"}},
		{From: "security@bankofamerica.com", To: "harper@example.com", Subject: "Unusual activity detected", Body: "We noticed a login from a new device. If this was you, no action is needed. Otherwise, please contact us immediately.", Labels: []string{"INBOX", "IMPORTANT", "UNREAD"}},
		{From: "dave.wilson@techcorp.com", To: "harper@example.com", Subject: "Code review needed", Body: "Hey, could you take a look at my PR when you get a chance? It's the refactoring of the auth module we discussed. No rush, whenever you have time.", Labels: []string{"INBOX"}},
		{From: "support@spotify.com", To: "harper@example.com", Subject: "Your Wrapped 2024 is ready!", Body: "It's that time of year again! Your personalized year in music is ready to explore. See your top artists, songs, and more.", Labels: []string{"INBOX"}},
		{From: "jenna.taylor@clientco.com", To: "harper@example.com", Subject: "Contract renewal discussion", Body: "Hi Harper, our current contract expires next month. I'd like to schedule a call to discuss renewal terms. What does your availability look like next Tuesday?", Labels: []string{"INBOX", "STARRED"}},
		{From: "no-reply@slack.com", To: "harper@example.com", Subject: "New message from #engineering", Body: "You have a new message in #engineering: @harper the deploy pipeline is green again, thanks for the fix!", Labels: []string{"INBOX"}},
		{From: "mike.brown@techcorp.com", To: "harper@example.com", Subject: "Offsite agenda", Body: "Attached is the agenda for next month's team offsite. Please review and let me know if you'd like to add any topics.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "receipts@uber.com", To: "harper@example.com", Subject: "Your ride receipt", Body: "Thanks for riding with Uber. Your trip from Downtown to Airport cost $34.50. Rate your driver to help improve the experience.", Labels: []string{"INBOX"}},
		{From: "alex.rivera@techcorp.com", To: "harper@example.com", Subject: "1:1 reschedule", Body: "Hey, something came up and I need to push our 1:1 to Thursday. Same time work for you?", Labels: []string{"INBOX", "UNREAD"}},
		{From: "deals@newegg.com", To: "harper@example.com", Subject: "Flash Sale: 30% off monitors!", Body: "Today only! Get 30% off select 4K monitors. Use code FLASH30 at checkout.", Labels: []string{"SPAM"}},
		{From: "emma.davis@vendor.io", To: "harper@example.com", Subject: "Invoice #2024-1234", Body: "Please find attached invoice for December services. Payment due within 30 days. Let me know if you have any questions.", Labels: []string{"INBOX"}},
		{From: "calendar@google.com", To: "harper@example.com", Subject: "Reminder: Dentist appointment tomorrow", Body: "This is a reminder for your dentist appointment tomorrow at 2:00 PM with Dr. Smith.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "chris.lee@techcorp.com", To: "harper@example.com", Subject: "Great job on the demo!", Body: "Just wanted to say the client demo went really well today. The new features looked polished. Team effort!", Labels: []string{"INBOX", "STARRED"}},
		{From: "no-reply@linkedin.com", To: "harper@example.com", Subject: "You have 3 new connection requests", Body: "Alice Chen, Bob Martinez, and 1 other want to connect with you. View and respond to your pending invitations.", Labels: []string{"INBOX"}},
		{From: "travel@expedia.com", To: "harper@example.com", Subject: "Your trip confirmation", Body: "Your hotel booking at Marriott Downtown is confirmed for Dec 15-17. Check-in starts at 3 PM.", Labels: []string{"INBOX", "IMPORTANT"}},
		{From: "hello@substack.com", To: "harper@example.com", Subject: "New post from Tech Weekly", Body: "New article: 'Why SQLite is the most deployed database in the world' - A deep dive into the architecture that made it ubiquitous.", Labels: []string{"INBOX"}},
		{From: "team@figma.com", To: "harper@example.com", Subject: "New comment on your design", Body: "Sarah left a comment on 'Dashboard Redesign v2': 'Love the new color scheme! Can we try a slightly darker shade for the sidebar?'", Labels: []string{"INBOX", "UNREAD"}},
		{From: "grandpa@aol.com", To: "harper@example.com", Subject: "Fwd: Fwd: Fwd: Funny joke", Body: "THOUGHT YOU WOULD LIKE THIS ONE!! 😂😂 WHY DID THE CHICKEN CROSS THE ROAD? TO GET TO THE OTHER SIDE!! HAHA LOVE GRANDPA", Labels: []string{"INBOX"}},
		{From: "recruiting@bigtech.com", To: "harper@example.com", Subject: "Exciting opportunity at BigTech", Body: "Hi Harper, I came across your profile and think you'd be a great fit for our Senior Engineer role. Would you be open to a quick chat?", Labels: []string{"INBOX"}},
		{From: "hr@techcorp.com", To: "harper@example.com", Subject: "Benefits enrollment deadline", Body: "Reminder: Open enrollment ends December 15th. Please log in to the benefits portal to review and confirm your selections.", Labels: []string{"INBOX", "IMPORTANT", "UNREAD"}},
		{From: "friend@protonmail.com", To: "harper@example.com", Subject: "Game night Saturday?", Body: "Hey! Thinking of hosting game night this Saturday. Settlers of Catan, maybe some poker? Let me know if you can make it!", Labels: []string{"INBOX", "UNREAD"}},
		{From: "orders@doordash.com", To: "harper@example.com", Subject: "Your order is on the way", Body: "Good news! Your order from Chipotle is being prepared and will arrive in approximately 25 minutes.", Labels: []string{"INBOX"}},
		{From: "it@techcorp.com", To: "harper@example.com", Subject: "Password expiration notice", Body: "Your network password will expire in 7 days. Please update it before expiration to avoid access interruption.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "legal@techcorp.com", To: "harper@example.com", Subject: "NDA for review", Body: "Please review and sign the attached NDA for the upcoming partnership discussions. Let me know if you have any questions.", Labels: []string{"INBOX", "IMPORTANT"}},
		{From: "noreply@zoom.us", To: "harper@example.com", Subject: "Cloud recording available", Body: "Your cloud recording 'Product Planning Session' is now available. The recording will be deleted in 30 days.", Labels: []string{"INBOX"}},
		{From: "sister@gmail.com", To: "harper@example.com", Subject: "Birthday present ideas??", Body: "Mom's birthday is coming up and I have NO ideas. What are you getting her? We should coordinate so we don't get the same thing again lol", Labels: []string{"INBOX", "UNREAD"}},
		{From: "prince.nigeria@definitely-legit.com", To: "harper@example.com", Subject: "URGENT: $10,000,000 inheritance", Body: "Greetings! I am Prince of Nigeria with urgent matter. You have inherited $10,000,000. Send bank details immediately.", Labels: []string{"SPAM"}},
		{From: "alerts@uptimerobot.com", To: "harper@example.com", Subject: "Monitor Down: api.myproject.com", Body: "Your monitor 'Production API' is DOWN. Started: 2024-12-01 14:32:05 UTC. We'll notify you when it's back up.", Labels: []string{"INBOX", "IMPORTANT", "UNREAD"}},
		{From: "jane.kim@techcorp.com", To: "harper@example.com", Subject: "Quick question about the API", Body: "Hey Harper, is there any rate limiting on the new endpoints? I'm updating the docs and want to make sure I have the right info.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "noreply@stripe.com", To: "harper@example.com", Subject: "Successful payment", Body: "A payment of $49.00 was successfully processed for your subscription to Premium Plan.", Labels: []string{"INBOX"}},
		{From: "college.friend@yahoo.com", To: "harper@example.com", Subject: "Reunion next summer?", Body: "Can you believe it's been 10 years since graduation? A bunch of us are planning a reunion. You should totally come!", Labels: []string{"INBOX"}},
		{From: "no-reply@vercel.com", To: "harper@example.com", Subject: "Deploy succeeded", Body: "Successfully deployed myproject to production. Commit: abc1234 'fix: handle edge case in auth flow'", Labels: []string{"INBOX"}},
		{From: "gym@fitlife.com", To: "harper@example.com", Subject: "Your membership renewal", Body: "Your annual membership expires on December 31st. Renew now and save 20% with code NEWYEAR.", Labels: []string{"INBOX"}},
		{From: "peter.zhang@consulting.com", To: "harper@example.com", Subject: "Follow up from conference", Body: "Great meeting you at the tech conference last week! I'd love to continue our conversation about distributed systems. Coffee sometime?", Labels: []string{"INBOX", "STARRED"}},
		{From: "no-reply@notion.so", To: "harper@example.com", Subject: "You were mentioned in a doc", Body: "Sarah mentioned you in 'Q1 2025 Roadmap': @harper can you add the technical requirements for the auth rewrite?", Labels: []string{"INBOX", "UNREAD"}},
		{From: "rewards@starbucks.com", To: "harper@example.com", Subject: "You earned a free drink!", Body: "Congrats! You've earned enough stars for a free drink of any size. Redeem it in the app before it expires.", Labels: []string{"INBOX"}},
		{From: "manager@techcorp.com", To: "harper@example.com", Subject: "Performance review scheduled", Body: "Hi Harper, I've scheduled your year-end performance review for next Friday at 2 PM. Please complete your self-assessment beforehand.", Labels: []string{"INBOX", "IMPORTANT", "UNREAD"}},
		{From: "automated@jira.atlassian.com", To: "harper@example.com", Subject: "[PROJ-1234] Status changed", Body: "The issue 'Implement caching layer' has been moved from 'In Progress' to 'Code Review' by Sarah Johnson.", Labels: []string{"INBOX"}},
		{From: "roommate@gmail.com", To: "harper@example.com", Subject: "Rent + utilities", Body: "Hey, rent and utilities come out to $1,450 this month. Internet was a bit higher because of the speed upgrade. Venmo works!", Labels: []string{"INBOX", "UNREAD"}},
		{From: "no-reply@medium.com", To: "harper@example.com", Subject: "Daily Digest", Body: "Today's top stories: 'Microservices are dead', 'Why I switched from React to HTMX', 'The database your startup should be using'.", Labels: []string{"INBOX"}},
		{From: "design@techcorp.com", To: "harper@example.com", Subject: "New mockups ready", Body: "Hey team, the updated mockups for the mobile app are ready for review in Figma. Please leave your feedback by EOD Thursday.", Labels: []string{"INBOX", "UNREAD"}},
		{From: "insurance@provider.com", To: "harper@example.com", Subject: "Policy renewal reminder", Body: "Your auto insurance policy is up for renewal next month. Log in to review your coverage options and rates.", Labels: []string{"INBOX"}},
		{From: "noreply@twitter.com", To: "harper@example.com", Subject: "New follower!", Body: "Tech Daily (@techdaily) is now following you on Twitter. View their profile.", Labels: []string{"INBOX"}},
		{From: "buddy@gmail.com", To: "harper@example.com", Subject: "Concert tickets!", Body: "Dude! Just scored tickets to the show next month. You still want to go? They're gonna sell out fast.", Labels: []string{"INBOX", "STARRED", "UNREAD"}},
	}

	// Return up to count emails, cycling through templates if needed
	result := make([]EmailData, count)
	for i := 0; i < count; i++ {
		result[i] = templates[i%len(templates)]
	}
	return result
}

func generateStaticEvents(count int) []EventData {
	now := time.Now()

	templates := []EventData{
		{Summary: "Team Standup", Description: "Daily sync with the engineering team", StartTime: now.AddDate(0, 0, 1).Format("2006-01-02") + "T09:00:00Z", EndTime: now.AddDate(0, 0, 1).Format("2006-01-02") + "T09:30:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "bob@techcorp.com"}},
		{Summary: "1:1 with Sarah", Description: "Weekly sync", StartTime: now.AddDate(0, 0, 2).Format("2006-01-02") + "T10:00:00Z", EndTime: now.AddDate(0, 0, 2).Format("2006-01-02") + "T10:30:00Z", Attendees: []string{"harper@example.com", "sarah@techcorp.com"}},
		{Summary: "Product Planning", Description: "Q1 roadmap discussion", StartTime: now.AddDate(0, 0, 3).Format("2006-01-02") + "T14:00:00Z", EndTime: now.AddDate(0, 0, 3).Format("2006-01-02") + "T15:00:00Z", Attendees: []string{"harper@example.com", "pm@techcorp.com", "design@techcorp.com"}},
		{Summary: "Client Call - Acme Inc", Description: "Quarterly review", StartTime: now.AddDate(0, 0, 4).Format("2006-01-02") + "T11:00:00Z", EndTime: now.AddDate(0, 0, 4).Format("2006-01-02") + "T12:00:00Z", Attendees: []string{"harper@example.com", "bob@acmeinc.com", "sales@techcorp.com"}},
		{Summary: "Dentist Appointment", Description: "Regular checkup with Dr. Smith", StartTime: now.AddDate(0, 0, 5).Format("2006-01-02") + "T14:00:00Z", EndTime: now.AddDate(0, 0, 5).Format("2006-01-02") + "T15:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Focus Time", Description: "No meetings - deep work block", StartTime: now.AddDate(0, 0, 6).Format("2006-01-02") + "T09:00:00Z", EndTime: now.AddDate(0, 0, 6).Format("2006-01-02") + "T12:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Team Lunch", Description: "Monthly team lunch at Thai Garden", StartTime: now.AddDate(0, 0, 7).Format("2006-01-02") + "T12:00:00Z", EndTime: now.AddDate(0, 0, 7).Format("2006-01-02") + "T13:30:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "bob@techcorp.com", "sarah@techcorp.com"}},
		{Summary: "Architecture Review", Description: "Review new microservices design", StartTime: now.AddDate(0, 0, 8).Format("2006-01-02") + "T15:00:00Z", EndTime: now.AddDate(0, 0, 8).Format("2006-01-02") + "T16:00:00Z", Attendees: []string{"harper@example.com", "architect@techcorp.com", "senior@techcorp.com"}},
		{Summary: "Coffee with Peter", Description: "Networking chat", StartTime: now.AddDate(0, 0, 9).Format("2006-01-02") + "T16:00:00Z", EndTime: now.AddDate(0, 0, 9).Format("2006-01-02") + "T17:00:00Z", Attendees: []string{"harper@example.com", "peter@consulting.com"}},
		{Summary: "Gym", Description: "Workout session", StartTime: now.AddDate(0, 0, 10).Format("2006-01-02") + "T07:00:00Z", EndTime: now.AddDate(0, 0, 10).Format("2006-01-02") + "T08:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Sprint Planning", Description: "Plan next sprint tasks", StartTime: now.AddDate(0, 0, 11).Format("2006-01-02") + "T10:00:00Z", EndTime: now.AddDate(0, 0, 11).Format("2006-01-02") + "T11:30:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "bob@techcorp.com", "pm@techcorp.com"}},
		{Summary: "Interview - Backend Engineer", Description: "Technical interview", StartTime: now.AddDate(0, 0, 12).Format("2006-01-02") + "T14:00:00Z", EndTime: now.AddDate(0, 0, 12).Format("2006-01-02") + "T15:00:00Z", Attendees: []string{"harper@example.com", "hr@techcorp.com"}},
		{Summary: "Happy Hour", Description: "Team social at The Pub", StartTime: now.AddDate(0, 0, 13).Format("2006-01-02") + "T17:30:00Z", EndTime: now.AddDate(0, 0, 13).Format("2006-01-02") + "T19:30:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "bob@techcorp.com"}},
		{Summary: "Vendor Demo", Description: "Demo of new monitoring tool", StartTime: now.AddDate(0, 0, 14).Format("2006-01-02") + "T13:00:00Z", EndTime: now.AddDate(0, 0, 14).Format("2006-01-02") + "T14:00:00Z", Attendees: []string{"harper@example.com", "emma@vendor.io", "ops@techcorp.com"}},
		{Summary: "Car Service", Description: "Oil change at AutoCare", StartTime: now.AddDate(0, 0, 15).Format("2006-01-02") + "T09:00:00Z", EndTime: now.AddDate(0, 0, 15).Format("2006-01-02") + "T10:30:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "All Hands Meeting", Description: "Monthly company update", StartTime: now.AddDate(0, 0, 16).Format("2006-01-02") + "T16:00:00Z", EndTime: now.AddDate(0, 0, 16).Format("2006-01-02") + "T17:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Code Review Session", Description: "Review pending PRs as a team", StartTime: now.AddDate(0, 0, 17).Format("2006-01-02") + "T14:00:00Z", EndTime: now.AddDate(0, 0, 17).Format("2006-01-02") + "T15:00:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "dave@techcorp.com"}},
		{Summary: "Doctor Appointment", Description: "Annual physical", StartTime: now.AddDate(0, 0, 18).Format("2006-01-02") + "T10:00:00Z", EndTime: now.AddDate(0, 0, 18).Format("2006-01-02") + "T11:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Project Kickoff", Description: "New project start with client", StartTime: now.AddDate(0, 0, 19).Format("2006-01-02") + "T11:00:00Z", EndTime: now.AddDate(0, 0, 19).Format("2006-01-02") + "T12:00:00Z", Attendees: []string{"harper@example.com", "jenna@clientco.com", "pm@techcorp.com"}},
		{Summary: "Dinner with Mom", Description: "Sunday family dinner", StartTime: now.AddDate(0, 0, 20).Format("2006-01-02") + "T18:00:00Z", EndTime: now.AddDate(0, 0, 20).Format("2006-01-02") + "T20:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Performance Review", Description: "Year-end review with manager", StartTime: now.AddDate(0, 0, 21).Format("2006-01-02") + "T14:00:00Z", EndTime: now.AddDate(0, 0, 21).Format("2006-01-02") + "T15:00:00Z", Attendees: []string{"harper@example.com", "manager@techcorp.com"}},
		{Summary: "Training: AWS Certification", Description: "Study session", StartTime: now.AddDate(0, 0, 22).Format("2006-01-02") + "T09:00:00Z", EndTime: now.AddDate(0, 0, 22).Format("2006-01-02") + "T11:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Game Night", Description: "Board games at friend's place", StartTime: now.AddDate(0, 0, 23).Format("2006-01-02") + "T19:00:00Z", EndTime: now.AddDate(0, 0, 23).Format("2006-01-02") + "T23:00:00Z", Attendees: []string{"harper@example.com"}},
		{Summary: "Retrospective", Description: "Sprint retro", StartTime: now.AddDate(0, 0, 24).Format("2006-01-02") + "T15:00:00Z", EndTime: now.AddDate(0, 0, 24).Format("2006-01-02") + "T16:00:00Z", Attendees: []string{"harper@example.com", "alice@techcorp.com", "bob@techcorp.com", "pm@techcorp.com"}},
		{Summary: "Concert", Description: "Live music downtown", StartTime: now.AddDate(0, 0, 25).Format("2006-01-02") + "T20:00:00Z", EndTime: now.AddDate(0, 0, 25).Format("2006-01-02") + "T23:30:00Z", Attendees: []string{"harper@example.com", "buddy@gmail.com"}},
	}

	result := make([]EventData, count)
	for i := 0; i < count; i++ {
		e := templates[i%len(templates)]
		// Offset days for cycling
		offset := (i / len(templates)) * len(templates)
		if offset > 0 {
			e.StartTime = now.AddDate(0, 0, (i%len(templates))+1+offset).Format("2006-01-02") + e.StartTime[10:]
			e.EndTime = now.AddDate(0, 0, (i%len(templates))+1+offset).Format("2006-01-02") + e.EndTime[10:]
		}
		result[i] = e
	}
	return result
}

func generateStaticContacts(count int) []ContactData {
	templates := []ContactData{
		{Name: "Alice Chen", Email: "alice.chen@techcorp.com", Phone: "555-234-5678", Company: "TechCorp"},
		{Name: "Bob Martinez", Email: "bob.martinez@acmeinc.com", Phone: "555-345-6789", Company: "Acme Inc"},
		{Name: "Sarah Johnson", Email: "sarah.johnson@techcorp.com", Phone: "555-456-7890", Company: "TechCorp"},
		{Name: "Dave Wilson", Email: "dave.wilson@techcorp.com", Phone: "555-567-8901", Company: "TechCorp"},
		{Name: "Emma Davis", Email: "emma.davis@vendor.io", Phone: "555-678-9012", Company: "Vendor.io"},
		{Name: "Mom", Email: "mom@gmail.com", Phone: "555-111-2222", Company: ""},
		{Name: "Dad", Email: "dad@gmail.com", Phone: "555-111-3333", Company: ""},
		{Name: "Sister", Email: "sister@gmail.com", Phone: "555-111-4444", Company: ""},
		{Name: "Grandpa Joe", Email: "grandpa@aol.com", Phone: "555-111-5555", Company: ""},
		{Name: "Dr. Smith", Email: "drsmith@dentalcare.com", Phone: "555-222-3333", Company: "Dental Care Associates"},
		{Name: "Peter Zhang", Email: "peter.zhang@consulting.com", Phone: "555-333-4444", Company: "Zhang Consulting"},
		{Name: "Jenna Taylor", Email: "jenna.taylor@clientco.com", Phone: "555-444-5555", Company: "ClientCo"},
		{Name: "Mike Brown", Email: "mike.brown@techcorp.com", Phone: "555-555-6666", Company: "TechCorp"},
		{Name: "Chris Lee", Email: "chris.lee@techcorp.com", Phone: "555-666-7777", Company: "TechCorp"},
		{Name: "Alex Rivera", Email: "alex.rivera@techcorp.com", Phone: "555-777-8888", Company: "TechCorp"},
		{Name: "Jane Kim", Email: "jane.kim@techcorp.com", Phone: "555-888-9999", Company: "TechCorp"},
		{Name: "Best Friend", Email: "buddy@gmail.com", Phone: "555-123-4567", Company: ""},
		{Name: "Roommate", Email: "roommate@gmail.com", Phone: "555-234-5670", Company: ""},
		{Name: "College Friend", Email: "college.friend@yahoo.com", Phone: "555-345-6780", Company: ""},
		{Name: "Gym Buddy", Email: "gym@gmail.com", Phone: "555-456-7891", Company: ""},
		{Name: "Accountant - Jim Ford", Email: "jim@fordaccounting.com", Phone: "555-567-8902", Company: "Ford Accounting"},
		{Name: "Lawyer - Lisa Park", Email: "lpark@legalfirm.com", Phone: "555-678-9013", Company: "Park Legal"},
		{Name: "Insurance - Tom Reyes", Email: "treyes@insurance.com", Phone: "555-789-0124", Company: "State Insurance"},
		{Name: "Auto Shop - Mike", Email: "mike@autocare.com", Phone: "555-890-1235", Company: "AutoCare"},
		{Name: "Landlord", Email: "landlord@propertyco.com", Phone: "555-901-2346", Company: "Property Management Co"},
	}

	result := make([]ContactData, count)
	for i := 0; i < count; i++ {
		c := templates[i%len(templates)]
		if i >= len(templates) {
			// Add suffix to make unique
			suffix := fmt.Sprintf(" %d", i/len(templates)+1)
			c.Name = c.Name + suffix
			c.Email = fmt.Sprintf("contact%d@example.com", i)
		}
		result[i] = c
	}
	return result
}
