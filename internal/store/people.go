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

func (s *Store) CreatePersonFromForm(userID, name, email string) (*PersonView, error) {
	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	resourceName := "people/" + id

	data := fmt.Sprintf(`{"names":[{"displayName":"%s"}],"emailAddresses":[{"value":"%s"}]}`, name, email)

	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		resourceName, userID, data,
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
