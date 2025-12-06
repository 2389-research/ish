// ABOUTME: Admin UI schema definitions for Discord plugin
// ABOUTME: Defines Webhooks and Messages resources for schema-driven UI

package discord

import "github.com/2389/ish/plugins/core"

func (p *DiscordPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Webhooks",
				Slug:        "webhooks",
				ListColumns: []string{"id", "name", "token", "channel_id", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "token", Type: "string", Display: "Token", Required: true, Editable: false},
					{Name: "type", Type: "string", Display: "Type", Required: false, Editable: false},
					{Name: "name", Type: "string", Display: "Name", Required: false, Editable: false},
					{Name: "avatar", Type: "string", Display: "Avatar", Required: false, Editable: false},
					{Name: "channel_id", Type: "string", Display: "Channel ID", Required: false, Editable: false},
					{Name: "guild_id", Type: "string", Display: "Guild ID", Required: false, Editable: false},
					{Name: "application_id", Type: "string", Display: "Application ID", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{
						Name:       "delete",
						HTTPMethod: "DELETE",
						Endpoint:   "/api/webhooks/{id}/{token}",
						Confirm:    true,
					},
				},
			},
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"id", "webhook_id", "content", "username", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "webhook_id", Type: "string", Display: "Webhook ID", Required: true, Editable: false},
					{Name: "content", Type: "text", Display: "Content", Required: false, Editable: false},
					{Name: "username", Type: "string", Display: "Username", Required: false, Editable: false},
					{Name: "avatar_url", Type: "string", Display: "Avatar URL", Required: false, Editable: false},
					{Name: "embeds", Type: "text", Display: "Embeds (JSON)", Required: false, Editable: false},
					{Name: "components", Type: "text", Display: "Components (JSON)", Required: false, Editable: false},
					{Name: "attachments", Type: "text", Display: "Attachments (JSON)", Required: false, Editable: false},
					{Name: "thread_id", Type: "string", Display: "Thread ID", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
					{Name: "edited_at", Type: "datetime", Display: "Edited", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{
						Name:       "delete",
						HTTPMethod: "DELETE",
						Endpoint:   "/api/webhooks/{webhook_id}/token/messages/{id}",
						Confirm:    true,
					},
				},
			},
		},
	}
}
