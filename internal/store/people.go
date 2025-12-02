// ABOUTME: People-related store operations for contacts.
// ABOUTME: Handles CRUD, listing, and search functionality.

package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Person struct {
	ResourceName string
	UserID       string
	Data         string
}

func (s *Store) CreatePerson(p *Person) error {
	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		p.ResourceName, p.UserID, p.Data,
	)
	return err
}

func (s *Store) SearchPeople(userID string, query string, pageSize int, pageToken string) ([]Person, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT resource_name, user_id, data FROM people WHERE user_id = ?"
	args := []any{userID}

	if query != "" {
		sqlQuery += " AND data LIKE ?"
		args = append(args, "%"+query+"%")
	}

	sqlQuery += " ORDER BY resource_name ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize+1, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data)
		if err != nil {
			return nil, "", err
		}
		people = append(people, p)
	}

	var nextToken string
	if len(people) > pageSize {
		people = people[:pageSize]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + pageSize)))
	}

	return people, nextToken, nil
}

func (s *Store) GetPerson(userID, resourceName string) (*Person, error) {
	var p Person
	err := s.db.QueryRow(
		"SELECT resource_name, user_id, data FROM people WHERE user_id = ? AND resource_name = ?",
		userID, resourceName,
	).Scan(&p.ResourceName, &p.UserID, &p.Data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("person not found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type PersonView struct {
	ID           string
	ResourceName string
	DisplayName  string
	Email        string
}

func (s *Store) ListAllPeople() ([]PersonView, error) {
	rows, err := s.db.Query("SELECT resource_name, data FROM people ORDER BY resource_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []PersonView
	for rows.Next() {
		var p PersonView
		var data string
		if err := rows.Scan(&p.ResourceName, &data); err != nil {
			return nil, err
		}

		// Extract ID from resource_name (people/c123 -> c123)
		p.ID = strings.TrimPrefix(p.ResourceName, "people/")

		// Parse data JSON
		var d struct {
			Names          []struct{ DisplayName string } `json:"names"`
			EmailAddresses []struct{ Value string }       `json:"emailAddresses"`
		}
		json.Unmarshal([]byte(data), &d)
		if len(d.Names) > 0 {
			p.DisplayName = d.Names[0].DisplayName
		}
		if len(d.EmailAddresses) > 0 {
			p.Email = d.EmailAddresses[0].Value
		}
		people = append(people, p)
	}
	return people, nil
}

func (s *Store) GetPersonView(userID, resourceName string) (*PersonView, error) {
	p, err := s.GetPerson(userID, resourceName)
	if err != nil {
		return nil, err
	}

	view := &PersonView{
		ID:           strings.TrimPrefix(p.ResourceName, "people/"),
		ResourceName: p.ResourceName,
	}

	var d struct {
		Names          []struct{ DisplayName string } `json:"names"`
		EmailAddresses []struct{ Value string }       `json:"emailAddresses"`
	}
	json.Unmarshal([]byte(p.Data), &d)
	if len(d.Names) > 0 {
		view.DisplayName = d.Names[0].DisplayName
	}
	if len(d.EmailAddresses) > 0 {
		view.Email = d.EmailAddresses[0].Value
	}

	return view, nil
}

func (s *Store) CreatePersonFromForm(userID, name, email string) (*PersonView, error) {
	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	resourceName := "people/" + id

	// Build data using json.Marshal to properly escape special characters
	personData := map[string]any{
		"names":          []map[string]string{{"displayName": name}},
		"emailAddresses": []map[string]string{{"value": email}},
	}
	dataBytes, _ := json.Marshal(personData)

	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		resourceName, userID, string(dataBytes),
	)
	if err != nil {
		return nil, err
	}

	return &PersonView{
		ID:           id,
		ResourceName: resourceName,
		DisplayName:  name,
		Email:        email,
	}, nil
}

func (s *Store) DeletePerson(id string) error {
	resourceName := "people/" + id
	_, err := s.db.Exec("DELETE FROM people WHERE resource_name = ?", resourceName)
	return err
}

// GetPeopleSyncToken returns the current sync token for a user's contacts.
func (s *Store) GetPeopleSyncToken(userID string) (string, error) {
	var token string
	err := s.db.QueryRow("SELECT token FROM sync_tokens WHERE resource_type = 'people' AND user_id = ?", userID).Scan(&token)
	if err == sql.ErrNoRows {
		// Generate new sync token if none exists
		token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		s.db.Exec("INSERT INTO sync_tokens (id, resource_type, user_id, token) VALUES (?, 'people', ?, ?)",
			fmt.Sprintf("sync_people_%s", userID), userID, token)
	} else if err != nil {
		return "", err
	}
	return token, nil
}

// UpdatePeopleSyncToken updates the sync token for a user's contacts.
func (s *Store) UpdatePeopleSyncToken(userID string) (string, error) {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	_, err := s.db.Exec("UPDATE sync_tokens SET token = ? WHERE resource_type = 'people' AND user_id = ?", token, userID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ListPeopleSince returns people updated since the given sync token.
func (s *Store) ListPeopleSince(userID string, syncToken string, pageSize int) ([]Person, string, error) {
	// Decode the sync token to get a timestamp
	var sinceTime int64
	if syncToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(syncToken)
		if err == nil {
			sinceTime, _ = strconv.ParseInt(string(decoded), 10, 64)
		}
	}

	// Convert nanoseconds to timestamp for comparison
	sinceTimestamp := time.Unix(0, sinceTime).Format(time.RFC3339)

	// Use COALESCE to handle NULL updated_at values (treat them as epoch 0)
	sqlQuery := `SELECT resource_name, user_id, data FROM people
		WHERE user_id = ? AND COALESCE(updated_at, '1970-01-01T00:00:00Z') > ? ORDER BY updated_at ASC LIMIT ?`

	rows, err := s.db.Query(sqlQuery, userID, sinceTimestamp, pageSize)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data); err != nil {
			return nil, "", err
		}
		people = append(people, p)
	}

	// Generate new sync token based on current time
	newToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return people, newToken, nil
}

// ListPeopleConnections lists people for the connections endpoint with sync token support.
func (s *Store) ListPeopleConnections(userID string, pageSize int, pageToken string, syncToken string) ([]Person, string, string, error) {
	if syncToken != "" {
		// Incremental sync
		people, newSyncToken, err := s.ListPeopleSince(userID, syncToken, pageSize)
		return people, "", newSyncToken, err
	}

	// Full sync with pagination
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT resource_name, user_id, data FROM people WHERE user_id = ? ORDER BY resource_name ASC LIMIT ? OFFSET ?"

	rows, err := s.db.Query(sqlQuery, userID, pageSize+1, offset)
	if err != nil {
		return nil, "", "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data); err != nil {
			return nil, "", "", err
		}
		people = append(people, p)
	}

	var nextPageToken string
	if len(people) > pageSize {
		people = people[:pageSize]
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + pageSize)))
	}

	// If no more pages, generate sync token
	var nextSyncToken string
	if nextPageToken == "" {
		nextSyncToken, _ = s.GetPeopleSyncToken(userID)
	}

	return people, nextPageToken, nextSyncToken, nil
}
