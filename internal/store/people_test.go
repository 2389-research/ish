// ABOUTME: Tests for People-related store operations.
// ABOUTME: Covers contact listing, retrieval, and search.

package store

import (
	"os"
	"testing"
)

func TestStore_People(t *testing.T) {
	dbPath := "test_people.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user
	s.CreateUser("harper")

	// Create person
	p := &Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	}
	err = s.CreatePerson(p)
	if err != nil {
		t.Fatalf("CreatePerson() error = %v", err)
	}

	// Search people
	people, nextToken, err := s.SearchPeople("harper", "alice", 10, "")
	if err != nil {
		t.Fatalf("SearchPeople() error = %v", err)
	}
	if len(people) != 1 {
		t.Errorf("SearchPeople() got %d people, want 1", len(people))
	}
	if nextToken != "" {
		t.Errorf("SearchPeople() nextToken = %q, want empty", nextToken)
	}

	// Get single person
	got, err := s.GetPerson("harper", "people/c123")
	if err != nil {
		t.Fatalf("GetPerson() error = %v", err)
	}
	if got.ResourceName != "people/c123" {
		t.Errorf("GetPerson() ResourceName = %q, want %q", got.ResourceName, "people/c123")
	}
}
