// ABOUTME: OAuth token storage and management.
// ABOUTME: Handles token lifecycle (create, validate, revoke).

package store

import (
	"time"
)

// OAuthToken represents an OAuth token
type OAuthToken struct {
	Token        string
	PluginName   string
	UserID       string
	Scopes       string
	ExpiresAt    time.Time
	RefreshToken string
	Revoked      bool
	CreatedAt    time.Time
}

// StoreToken stores a new OAuth token
func (s *Store) StoreToken(token *OAuthToken) error {
	_, err := s.db.Exec(`
		INSERT INTO oauth_tokens (token, plugin_name, user_id, scopes, expires_at, refresh_token, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, token.Token, token.PluginName, token.UserID, token.Scopes, token.ExpiresAt, token.RefreshToken, token.Revoked)
	return err
}

// GetToken retrieves a token by value
func (s *Store) GetToken(token string) (*OAuthToken, error) {
	t := &OAuthToken{}
	err := s.db.QueryRow(`
		SELECT token, plugin_name, user_id, COALESCE(scopes, ''), expires_at, COALESCE(refresh_token, ''), revoked, created_at
		FROM oauth_tokens WHERE token = ?
	`, token).Scan(&t.Token, &t.PluginName, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.RefreshToken, &t.Revoked, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// RevokeToken marks a token as revoked
func (s *Store) RevokeToken(token string) error {
	_, err := s.db.Exec(`UPDATE oauth_tokens SET revoked = 1 WHERE token = ?`, token)
	return err
}

// ListTokens retrieves all tokens (optionally filtered by plugin)
func (s *Store) ListTokens(pluginName string) ([]*OAuthToken, error) {
	query := `SELECT token, plugin_name, user_id, COALESCE(scopes, ''), expires_at, COALESCE(refresh_token, ''), revoked, created_at
	          FROM oauth_tokens WHERE 1=1`
	args := []any{}

	if pluginName != "" {
		query += " AND plugin_name = ?"
		args = append(args, pluginName)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*OAuthToken
	for rows.Next() {
		t := &OAuthToken{}
		if err := rows.Scan(&t.Token, &t.PluginName, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.RefreshToken, &t.Revoked, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
