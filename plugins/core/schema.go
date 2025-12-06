// ABOUTME: Schema definitions for plugin UI generation.
// ABOUTME: Plugins define schemas, ISH core renders the UI.

package core

// PluginSchema defines the admin UI for a plugin
type PluginSchema struct {
	Resources []ResourceSchema
}

// ResourceSchema defines a resource (Messages, Events, etc.)
type ResourceSchema struct {
	Name        string         // "Messages", "Events"
	Slug        string         // "messages", "events" (URL path)
	Fields      []FieldSchema  // What data to show/edit
	Actions     []ActionSchema // Available operations
	ListColumns []string       // Which fields in list view
}

// FieldSchema defines a field in a resource
type FieldSchema struct {
	Name     string // "subject", "from", "date"
	Type     string // "string", "datetime", "email", "text"
	Display  string // "Subject", "From", "Date"
	Required bool
	Editable bool
}

// ActionSchema defines an action on a resource
type ActionSchema struct {
	Name       string // "send", "delete", "archive"
	HTTPMethod string // "POST", "DELETE"
	Endpoint   string // Template: "/messages/{id}/send"
	Confirm    bool   // Show confirmation dialog?
}
