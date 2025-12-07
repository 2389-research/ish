// ABOUTME: Tests for GitHub plugin seed functionality
// ABOUTME: Verifies seed data generation works correctly for all sizes

package github

import (
	"context"
	"testing"
)

func setupTestPlugin(t *testing.T) (*GitHubPlugin, *GitHubStore) {
	db := setupTestDB(t)
	store, err := NewGitHubStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	plugin := &GitHubPlugin{store: store}
	return plugin, store
}

func TestSeedSmall(t *testing.T) {
	plugin, store := setupTestPlugin(t)
	defer store.db.Close()

	seedData, err := plugin.Seed(context.Background(), "small")
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify summary
	if seedData.Summary == "" {
		t.Fatal("Seed summary should not be empty")
	}

	// Verify record counts
	if seedData.Records["users"] != 2 {
		t.Fatalf("Expected 2 users, got %d", seedData.Records["users"])
	}
	if seedData.Records["repos"] != 3 {
		t.Fatalf("Expected 3 repos, got %d", seedData.Records["repos"])
	}
	if seedData.Records["issues"] != 5 {
		t.Fatalf("Expected 5 issues, got %d", seedData.Records["issues"])
	}
	if seedData.Records["prs"] != 2 {
		t.Fatalf("Expected 2 PRs, got %d", seedData.Records["prs"])
	}
	if seedData.Records["comments"] != 10 {
		t.Fatalf("Expected 10 comments, got %d", seedData.Records["comments"])
	}
	if seedData.Records["reviews"] != 3 {
		t.Fatalf("Expected 3 reviews, got %d", seedData.Records["reviews"])
	}
	if seedData.Records["webhooks"] != 0 {
		t.Fatalf("Expected 0 webhooks, got %d", seedData.Records["webhooks"])
	}

	// Verify actual DB counts
	var count int

	store.db.QueryRow("SELECT COUNT(*) FROM github_users").Scan(&count)
	if count != 2 {
		t.Fatalf("Expected 2 users in DB, got %d", count)
	}

	store.db.QueryRow("SELECT COUNT(*) FROM github_repositories").Scan(&count)
	if count != 3 {
		t.Fatalf("Expected 3 repositories in DB, got %d", count)
	}

	store.db.QueryRow("SELECT COUNT(*) FROM github_issues WHERE is_pull_request = 0").Scan(&count)
	if count != 5 {
		t.Fatalf("Expected 5 issues in DB, got %d", count)
	}

	store.db.QueryRow("SELECT COUNT(*) FROM github_issues WHERE is_pull_request = 1").Scan(&count)
	if count != 2 {
		t.Fatalf("Expected 2 PRs in DB, got %d", count)
	}

	store.db.QueryRow("SELECT COUNT(*) FROM github_comments").Scan(&count)
	if count != 10 {
		t.Fatalf("Expected 10 comments in DB, got %d", count)
	}

	store.db.QueryRow("SELECT COUNT(*) FROM github_reviews").Scan(&count)
	if count != 3 {
		t.Fatalf("Expected 3 reviews in DB, got %d", count)
	}
}

func TestSeedMedium(t *testing.T) {
	plugin, store := setupTestPlugin(t)
	defer store.db.Close()

	seedData, err := plugin.Seed(context.Background(), "medium")
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify counts match expected values
	if seedData.Records["users"] != 5 {
		t.Fatalf("Expected 5 users, got %d", seedData.Records["users"])
	}
	if seedData.Records["repos"] != 10 {
		t.Fatalf("Expected 10 repos, got %d", seedData.Records["repos"])
	}
	if seedData.Records["issues"] != 30 {
		t.Fatalf("Expected 30 issues, got %d", seedData.Records["issues"])
	}
	if seedData.Records["prs"] != 15 {
		t.Fatalf("Expected 15 PRs, got %d", seedData.Records["prs"])
	}
	if seedData.Records["comments"] != 50 {
		t.Fatalf("Expected 50 comments, got %d", seedData.Records["comments"])
	}
	if seedData.Records["reviews"] != 20 {
		t.Fatalf("Expected 20 reviews, got %d", seedData.Records["reviews"])
	}
	if seedData.Records["webhooks"] != 5 {
		t.Fatalf("Expected 5 webhooks, got %d", seedData.Records["webhooks"])
	}
}

func TestSeedLarge(t *testing.T) {
	plugin, store := setupTestPlugin(t)
	defer store.db.Close()

	seedData, err := plugin.Seed(context.Background(), "large")
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify counts match expected values
	if seedData.Records["users"] != 20 {
		t.Fatalf("Expected 20 users, got %d", seedData.Records["users"])
	}
	if seedData.Records["repos"] != 30 {
		t.Fatalf("Expected 30 repos, got %d", seedData.Records["repos"])
	}
	if seedData.Records["issues"] != 100 {
		t.Fatalf("Expected 100 issues, got %d", seedData.Records["issues"])
	}
	if seedData.Records["prs"] != 50 {
		t.Fatalf("Expected 50 PRs, got %d", seedData.Records["prs"])
	}
	if seedData.Records["comments"] != 200 {
		t.Fatalf("Expected 200 comments, got %d", seedData.Records["comments"])
	}
	if seedData.Records["reviews"] != 80 {
		t.Fatalf("Expected 80 reviews, got %d", seedData.Records["reviews"])
	}
	if seedData.Records["webhooks"] != 15 {
		t.Fatalf("Expected 15 webhooks, got %d", seedData.Records["webhooks"])
	}
}

func TestSeedSizes(t *testing.T) {
	tests := []struct {
		size     string
		users    int
		repos    int
		issues   int
		prs      int
		comments int
		reviews  int
		webhooks int
	}{
		{"small", 2, 3, 5, 2, 10, 3, 0},
		{"medium", 5, 10, 30, 15, 50, 20, 5},
		{"large", 20, 30, 100, 50, 200, 80, 15},
		{"", 2, 3, 5, 2, 10, 3, 0}, // default to small
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			plugin, store := setupTestPlugin(t)
			defer store.db.Close()

			seedData, err := plugin.Seed(context.Background(), tt.size)
			if err != nil {
				t.Fatalf("Seed failed: %v", err)
			}

			if seedData.Records["users"] != tt.users {
				t.Errorf("Expected %d users, got %d", tt.users, seedData.Records["users"])
			}
			if seedData.Records["repos"] != tt.repos {
				t.Errorf("Expected %d repos, got %d", tt.repos, seedData.Records["repos"])
			}
			if seedData.Records["issues"] != tt.issues {
				t.Errorf("Expected %d issues, got %d", tt.issues, seedData.Records["issues"])
			}
			if seedData.Records["prs"] != tt.prs {
				t.Errorf("Expected %d PRs, got %d", tt.prs, seedData.Records["prs"])
			}
			if seedData.Records["comments"] != tt.comments {
				t.Errorf("Expected %d comments, got %d", tt.comments, seedData.Records["comments"])
			}
			if seedData.Records["reviews"] != tt.reviews {
				t.Errorf("Expected %d reviews, got %d", tt.reviews, seedData.Records["reviews"])
			}
			if seedData.Records["webhooks"] != tt.webhooks {
				t.Errorf("Expected %d webhooks, got %d", tt.webhooks, seedData.Records["webhooks"])
			}
		})
	}
}

func TestSeedRealisticData(t *testing.T) {
	plugin, store := setupTestPlugin(t)
	defer store.db.Close()

	_, err := plugin.Seed(context.Background(), "small")
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify users have realistic data
	var login string
	store.db.QueryRow("SELECT login FROM github_users LIMIT 1").Scan(&login)
	if login == "" {
		t.Fatal("User login should not be empty")
	}

	// Verify repos have realistic names
	var repoName, fullName string
	store.db.QueryRow("SELECT name, full_name FROM github_repositories LIMIT 1").Scan(&repoName, &fullName)
	if repoName == "" {
		t.Fatal("Repo name should not be empty")
	}
	if fullName == "" {
		t.Fatal("Repo full_name should not be empty")
	}

	// Verify issues have varied states
	var state string
	var hasOpen, hasClosed bool
	rows, _ := store.db.Query("SELECT state FROM github_issues WHERE is_pull_request = 0")
	for rows.Next() {
		rows.Scan(&state)
		if state == "open" {
			hasOpen = true
		} else if state == "closed" {
			hasClosed = true
		}
	}
	rows.Close()

	if !hasOpen {
		t.Fatal("Should have some open issues")
	}
	if !hasClosed {
		t.Fatal("Should have some closed issues")
	}

	// Verify PRs have branches
	var headRef, baseRef string
	store.db.QueryRow("SELECT pr.head_ref, pr.base_ref FROM github_pull_requests pr LIMIT 1").Scan(&headRef, &baseRef)
	if headRef == "" {
		t.Fatal("PR head_ref should not be empty")
	}
	if baseRef == "" {
		t.Fatal("PR base_ref should not be empty")
	}

	// Verify reviews have varied states
	var reviewState string
	var hasApproved, hasChangesRequested, hasCommented bool
	rows, _ = store.db.Query("SELECT state FROM github_reviews")
	for rows.Next() {
		rows.Scan(&reviewState)
		if reviewState == "APPROVED" {
			hasApproved = true
		} else if reviewState == "CHANGES_REQUESTED" {
			hasChangesRequested = true
		} else if reviewState == "COMMENTED" {
			hasCommented = true
		}
	}
	rows.Close()

	// At least one type should exist
	if !hasApproved && !hasChangesRequested && !hasCommented {
		t.Fatal("Reviews should have varied states")
	}
}
