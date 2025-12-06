// ABOUTME: Schema definition for OAuth token admin UI.
// ABOUTME: Defines fields, actions, and list view for token management.

package oauth

import "github.com/2389/ish/plugins/core"

func getOAuthSchema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name: "OAuth Tokens",
				Slug: "tokens",
				Fields: []core.FieldSchema{
					{
						Name:     "token",
						Type:     "string",
						Display:  "Token (partial)",
						Required: true,
						Editable: false,
					},
					{
						Name:     "plugin_name",
						Type:     "string",
						Display:  "Plugin",
						Required: true,
						Editable: false,
					},
					{
						Name:     "user_id",
						Type:     "string",
						Display:  "User ID",
						Required: false,
						Editable: false,
					},
					{
						Name:     "scopes",
						Type:     "string",
						Display:  "Scopes",
						Required: false,
						Editable: false,
					},
					{
						Name:     "expires_at",
						Type:     "datetime",
						Display:  "Expires At",
						Required: false,
						Editable: false,
					},
					{
						Name:     "revoked",
						Type:     "boolean",
						Display:  "Revoked",
						Required: false,
						Editable: false,
					},
					{
						Name:     "created_at",
						Type:     "datetime",
						Display:  "Created At",
						Required: false,
						Editable: false,
					},
				},
				Actions: []core.ActionSchema{
					{
						Name:       "revoke",
						HTTPMethod: "POST",
						Endpoint:   "/oauth/tokens/{token}/revoke",
						Confirm:    true,
					},
				},
				ListColumns: []string{
					"token",
					"plugin_name",
					"user_id",
					"revoked",
					"created_at",
				},
			},
		},
	}
}
