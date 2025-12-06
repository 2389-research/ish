// ABOUTME: Google plugin schema definitions.
// ABOUTME: Defines UI structure for Gmail, Calendar, People, Tasks.

package google

import "github.com/2389/ish/plugins/core"

func getGoogleSchema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"subject", "from", "date"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "subject", Type: "string", Display: "Subject", Required: true, Editable: true},
					{Name: "from", Type: "email", Display: "From", Required: true, Editable: true},
					{Name: "to", Type: "email", Display: "To", Required: true, Editable: true},
					{Name: "body", Type: "text", Display: "Body", Required: false, Editable: true},
					{Name: "date", Type: "datetime", Display: "Date", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/gmail/v1/users/me/messages/{id}", Confirm: true},
				},
			},
			{
				Name:        "Events",
				Slug:        "events",
				ListColumns: []string{"summary", "start", "end"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "summary", Type: "string", Display: "Summary", Required: true, Editable: true},
					{Name: "description", Type: "text", Display: "Description", Required: false, Editable: true},
					{Name: "start", Type: "datetime", Display: "Start", Required: true, Editable: true},
					{Name: "end", Type: "datetime", Display: "End", Required: true, Editable: true},
					{Name: "location", Type: "string", Display: "Location", Required: false, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/calendar/v3/calendars/primary/events/{id}", Confirm: true},
				},
			},
			{
				Name:        "Contacts",
				Slug:        "contacts",
				ListColumns: []string{"name", "email", "phone"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "name", Type: "string", Display: "Name", Required: true, Editable: true},
					{Name: "email", Type: "email", Display: "Email", Required: false, Editable: true},
					{Name: "phone", Type: "string", Display: "Phone", Required: false, Editable: true},
					{Name: "company", Type: "string", Display: "Company", Required: false, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/people/v1/people/{id}", Confirm: true},
				},
			},
			{
				Name:        "Tasks",
				Slug:        "tasks",
				ListColumns: []string{"title", "due", "status"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "title", Type: "string", Display: "Title", Required: true, Editable: true},
					{Name: "notes", Type: "text", Display: "Notes", Required: false, Editable: true},
					{Name: "due", Type: "datetime", Display: "Due", Required: false, Editable: true},
					{Name: "status", Type: "string", Display: "Status", Required: true, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "complete", HTTPMethod: "PATCH", Endpoint: "/tasks/v1/lists/@default/tasks/{id}", Confirm: false},
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/tasks/v1/lists/@default/tasks/{id}", Confirm: true},
				},
			},
		},
	}
}
