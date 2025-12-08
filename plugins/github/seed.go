// ABOUTME: Test data seeding for GitHub plugin
// ABOUTME: Generates realistic users, repos, issues, PRs, comments, reviews, and webhooks

package github

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/2389/ish/plugins/core"
)

func (p *GitHubPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var users, repos, issues, prs, comments, reviews, webhooks int

	switch size {
	case "small":
		users, repos, issues, prs, comments, reviews, webhooks = 2, 3, 5, 2, 10, 3, 0
	case "medium":
		users, repos, issues, prs, comments, reviews, webhooks = 5, 10, 30, 15, 50, 20, 5
	case "large":
		users, repos, issues, prs, comments, reviews, webhooks = 20, 30, 100, 50, 200, 80, 15
	default:
		users, repos, issues, prs, comments, reviews, webhooks = 2, 3, 5, 2, 10, 3, 0
	}

	// Create users
	userLogins := []string{"alice", "bob", "charlie", "diana", "eric", "frank", "grace", "henry",
		"iris", "jack", "kate", "leo", "mary", "nick", "olive", "paul", "quinn", "rose", "sam", "tina"}

	createdUsers := make([]*User, 0, users)
	for i := 0; i < users; i++ {
		login := userLogins[i%len(userLogins)]
		if i >= len(userLogins) {
			login = fmt.Sprintf("%s%d", login, i)
		}
		token := fmt.Sprintf("ghp_%032d", i+1)

		user, err := p.store.GetOrCreateUser(login, token)
		if err != nil {
			return core.SeedData{}, err
		}
		createdUsers = append(createdUsers, user)
	}

	// Create repositories
	repoNames := []string{"frontend", "backend", "api", "mobile-app", "web-app", "database",
		"infrastructure", "docs", "tools", "analytics", "dashboard", "microservice",
		"auth-service", "payment-service", "notification-service", "user-service",
		"admin-panel", "client-sdk", "server-sdk", "cli-tool", "monitoring",
		"logging", "config", "secrets", "deploy", "ci-cd", "testing", "qa",
		"staging", "production"}
	repoDescriptions := []string{
		"Frontend application for the platform",
		"Backend API server",
		"RESTful API service",
		"Mobile application for iOS and Android",
		"Web application interface",
		"Database schema and migrations",
		"Infrastructure as code",
		"Documentation and guides",
		"Development tools and utilities",
		"Analytics and reporting",
	}

	createdRepos := make([]*Repository, 0, repos)
	reposPerUser := repos / users
	if reposPerUser == 0 {
		reposPerUser = 1
	}
	for i := 0; i < repos; i++ {
		userIdx := i / reposPerUser
		if userIdx >= len(createdUsers) {
			userIdx = len(createdUsers) - 1
		}
		user := createdUsers[userIdx]

		name := repoNames[i%len(repoNames)]
		if i >= len(repoNames) {
			name = fmt.Sprintf("%s-%d", name, i)
		}
		description := repoDescriptions[i%len(repoDescriptions)]
		private := i%3 == 0 // 33% private repos

		repo, err := p.store.CreateRepository(user.ID, name, description, private)
		if err != nil {
			return core.SeedData{}, err
		}
		createdRepos = append(createdRepos, repo)
	}

	// Create issues
	issueTitles := []string{
		"Fix authentication bug in login flow",
		"Add dark mode support",
		"Memory leak in background worker",
		"Implement user profile page",
		"Update dependencies to latest versions",
		"Add unit tests for API endpoints",
		"Improve error handling in middleware",
		"Optimize database queries",
		"Fix CORS configuration",
		"Add logging to critical paths",
		"Implement rate limiting",
		"Fix mobile responsive layout",
		"Add pagination to list views",
		"Security vulnerability in auth",
		"Performance issues on dashboard",
		"Add search functionality",
		"Fix timezone handling",
		"Implement email notifications",
		"Add caching layer",
		"Update API documentation",
	}
	issueBodies := []string{
		"Users are reporting issues with the login flow. Need to investigate and fix.",
		"We should add support for dark mode across the application.",
		"The background worker is consuming too much memory over time.",
		"Users need to be able to view and edit their profile information.",
		"Dependencies are outdated and have security vulnerabilities.",
		"We need better test coverage for the API endpoints.",
		"Error handling needs improvement to provide better feedback.",
		"Some queries are running slowly and need optimization.",
		"CORS is not configured properly for production domains.",
		"Critical code paths need better logging for debugging.",
	}

	createdIssues := make([]*Issue, 0, issues)
	for i := 0; i < issues; i++ {
		repo := createdRepos[i%len(createdRepos)]
		user := createdUsers[rand.Intn(len(createdUsers))]
		title := issueTitles[i%len(issueTitles)]
		body := issueBodies[i%len(issueBodies)]

		issue, err := p.store.CreateIssue(repo.ID, user.ID, title, body, false)
		if err != nil {
			return core.SeedData{}, err
		}

		// Close some issues
		if i%3 == 0 {
			issue.State = "closed"
			issue.StateReason = "completed"
			if err := p.store.UpdateIssue(issue); err != nil {
				return core.SeedData{}, err
			}
		}

		createdIssues = append(createdIssues, issue)
	}

	// Create pull requests
	prTitles := []string{
		"feat: add user authentication",
		"fix: resolve timeout in API",
		"chore: update dependencies",
		"refactor: simplify database layer",
		"feat: implement search feature",
		"fix: correct validation logic",
		"docs: update API documentation",
		"perf: optimize query performance",
		"feat: add email notifications",
		"fix: handle edge case in parser",
		"feat: implement dark mode",
		"fix: resolve memory leak",
		"chore: update build config",
		"refactor: clean up unused code",
		"feat: add pagination support",
	}
	prBodies := []string{
		"This PR adds user authentication with JWT tokens.",
		"Fixes the timeout issue reported in the API endpoint.",
		"Updates all dependencies to their latest stable versions.",
		"Simplifies the database layer to make it more maintainable.",
		"Implements the new search feature as designed.",
		"Corrects the validation logic to handle all cases.",
		"Updates the API documentation with new endpoints.",
		"Optimizes query performance by adding indexes.",
		"Adds email notification support for key events.",
		"Handles an edge case that was causing parser failures.",
	}
	branches := []string{"fix/auth-bug", "feat/dark-mode", "fix/memory-leak", "feat/search",
		"fix/timeout", "feat/notifications", "fix/validation", "refactor/db",
		"chore/deps", "perf/queries", "feat/pagination", "fix/cors",
		"feat/logging", "fix/timezone", "feat/caching"}

	createdPRs := make([]*Issue, 0, prs)
	for i := 0; i < prs; i++ {
		repo := createdRepos[i%len(createdRepos)]
		user := createdUsers[rand.Intn(len(createdUsers))]
		title := prTitles[i%len(prTitles)]
		body := prBodies[i%len(prBodies)]
		headRef := branches[i%len(branches)]
		baseRef := "main"

		issue, pr, err := p.store.CreatePullRequest(repo.ID, user.ID, title, body, headRef, baseRef)
		if err != nil {
			return core.SeedData{}, err
		}

		// Close some PRs
		if i%3 == 0 {
			issue.State = "closed"
			if err := p.store.UpdateIssue(issue); err != nil {
				return core.SeedData{}, err
			}
		}

		// Merge some PRs
		if i%4 == 0 && issue.State == "closed" {
			mergedBy := createdUsers[rand.Intn(len(createdUsers))]
			if err := p.store.MergePullRequest(issue.ID, mergedBy.ID); err != nil {
				return core.SeedData{}, err
			}
		}

		createdPRs = append(createdPRs, issue)
		_ = pr // Store the PR for potential future use
	}

	// Create comments on issues and PRs
	commentBodies := []string{
		"I can reproduce this issue locally.",
		"This looks good to me, thanks!",
		"Can you add unit tests for this?",
		"I think we should also consider...",
		"Great work on this feature!",
		"This might break backward compatibility.",
		"Have you tested this on production?",
		"Please update the documentation as well.",
		"I found a small bug in the implementation.",
		"This is a duplicate of #123.",
		"Can you rebase on main?",
		"LGTM, shipping it!",
		"We need to add error handling here.",
		"This will improve performance significantly.",
		"I have some concerns about security.",
	}

	allIssuesAndPRs := append([]*Issue{}, createdIssues...)
	allIssuesAndPRs = append(allIssuesAndPRs, createdPRs...)

	commentCount := 0
	for i := 0; i < comments && len(allIssuesAndPRs) > 0; i++ {
		issue := allIssuesAndPRs[i%len(allIssuesAndPRs)]
		user := createdUsers[rand.Intn(len(createdUsers))]
		body := commentBodies[i%len(commentBodies)]

		_, err := p.store.CreateComment(issue.ID, user.ID, body)
		if err != nil {
			return core.SeedData{}, err
		}
		commentCount++
	}

	// Create reviews on PRs
	reviewStates := []string{"APPROVED", "CHANGES_REQUESTED", "COMMENTED"}
	reviewWeights := []int{5, 3, 2} // 50% approved, 30% changes requested, 20% commented
	reviewBodies := []string{
		"Looks good, approved!",
		"Please address the comments before merging.",
		"Just a few minor suggestions.",
		"Excellent work, shipping this!",
		"We need to fix the security issue first.",
		"Great implementation, very clean code.",
		"I have some concerns about performance.",
		"Documentation needs to be updated.",
		"This is exactly what we needed!",
		"Can we add more test coverage?",
	}

	reviewCount := 0
	for i := 0; i < reviews && len(createdPRs) > 0; i++ {
		issue := createdPRs[i%len(createdPRs)]
		user := createdUsers[rand.Intn(len(createdUsers))]

		// Weighted random selection
		total := 0
		for _, w := range reviewWeights {
			total += w
		}
		r := rand.Intn(total)
		stateIdx := 0
		for j, w := range reviewWeights {
			r -= w
			if r < 0 {
				stateIdx = j
				break
			}
		}
		state := reviewStates[stateIdx]
		body := reviewBodies[i%len(reviewBodies)]

		review, err := p.store.CreateReview(issue.ID, user.ID, state, body)
		if err != nil {
			return core.SeedData{}, err
		}

		// Submit the review
		if err := p.store.SubmitReview(review.ID); err != nil {
			return core.SeedData{}, err
		}
		reviewCount++
	}

	// Create webhooks
	// Using example.com which is a real, resolvable domain for testing
	webhookURLs := []string{
		"https://example.com/hooks/github/events",
		"https://example.com/api/webhooks/github",
		"https://example.com/ci/github/trigger",
		"https://example.com/notify/github",
		"https://example.com/slack/integrations/github",
	}
	eventTypes := [][]string{
		{"push", "pull_request"},
		{"issues", "issue_comment"},
		{"push", "release"},
		{"pull_request", "pull_request_review"},
		{"push", "pull_request", "issues"},
	}

	webhookCount := 0
	for i := 0; i < webhooks && len(createdRepos) > 0; i++ {
		repo := createdRepos[i%len(createdRepos)]
		url := webhookURLs[i%len(webhookURLs)]
		events := eventTypes[i%len(eventTypes)]
		secret := fmt.Sprintf("secret_%d", i+1)

		_, err := p.store.CreateWebhook(repo.ID, url, "application/json", secret, events)
		if err != nil {
			return core.SeedData{}, err
		}
		webhookCount++
	}

	summary := fmt.Sprintf("Created %d users, %d repos, %d issues, %d PRs, %d comments, %d reviews, %d webhooks",
		len(createdUsers), len(createdRepos), len(createdIssues), len(createdPRs),
		commentCount, reviewCount, webhookCount)

	return core.SeedData{
		Summary: summary,
		Records: map[string]int{
			"users":    len(createdUsers),
			"repos":    len(createdRepos),
			"issues":   len(createdIssues),
			"prs":      len(createdPRs),
			"comments": commentCount,
			"reviews":  reviewCount,
			"webhooks": webhookCount,
		},
	}, nil
}
