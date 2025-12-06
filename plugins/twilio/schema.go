// ABOUTME: Admin UI schema definitions for Twilio plugin
// ABOUTME: Defines resources, fields, and actions for schema-driven UI

package twilio

import "github.com/2389/ish/plugins/core"

func (p *TwilioPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name: "Accounts",
				Slug: "accounts",
				Fields: []core.FieldSchema{
					{Name: "account_sid", Type: "string", Display: "Account SID", Required: true},
					{Name: "auth_token", Type: "string", Display: "Auth Token"},
					{Name: "friendly_name", Type: "string", Display: "Friendly Name", Editable: true},
					{Name: "status", Type: "string", Display: "Status"},
					{Name: "created_at", Type: "datetime", Display: "Created"},
				},
				ListColumns: []string{"account_sid", "friendly_name", "status", "created_at"},
			},
			{
				Name: "Messages",
				Slug: "messages",
				Fields: []core.FieldSchema{
					{Name: "sid", Type: "string", Display: "Message SID", Required: true},
					{Name: "account_sid", Type: "string", Display: "Account"},
					{Name: "from_number", Type: "string", Display: "From"},
					{Name: "to_number", Type: "string", Display: "To"},
					{Name: "body", Type: "text", Display: "Body"},
					{Name: "status", Type: "string", Display: "Status"},
					{Name: "direction", Type: "string", Display: "Direction"},
					{Name: "date_created", Type: "datetime", Display: "Created"},
				},
				ListColumns: []string{"sid", "from_number", "to_number", "status", "date_created"},
			},
			{
				Name: "Calls",
				Slug: "calls",
				Fields: []core.FieldSchema{
					{Name: "sid", Type: "string", Display: "Call SID", Required: true},
					{Name: "account_sid", Type: "string", Display: "Account"},
					{Name: "from_number", Type: "string", Display: "From"},
					{Name: "to_number", Type: "string", Display: "To"},
					{Name: "status", Type: "string", Display: "Status"},
					{Name: "duration", Type: "string", Display: "Duration (s)"},
					{Name: "date_created", Type: "datetime", Display: "Created"},
				},
				ListColumns: []string{"sid", "from_number", "to_number", "status", "duration", "date_created"},
			},
			{
				Name: "Phone Numbers",
				Slug: "phone_numbers",
				Fields: []core.FieldSchema{
					{Name: "sid", Type: "string", Display: "SID", Required: true},
					{Name: "account_sid", Type: "string", Display: "Account"},
					{Name: "phone_number", Type: "string", Display: "Phone Number"},
					{Name: "friendly_name", Type: "string", Display: "Friendly Name", Editable: true},
					{Name: "voice_url", Type: "string", Display: "Voice URL", Editable: true},
					{Name: "sms_url", Type: "string", Display: "SMS URL", Editable: true},
					{Name: "created_at", Type: "datetime", Display: "Created"},
				},
				ListColumns: []string{"sid", "phone_number", "friendly_name", "created_at"},
			},
		},
	}
}
