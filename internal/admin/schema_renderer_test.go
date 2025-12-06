// ABOUTME: Tests for schema-based HTML renderer.
// ABOUTME: Ensures all field types and actions render correctly.

package admin

import (
	"strings"
	"testing"

	"github.com/2389/ish/plugins/core"
)

func TestRenderResourceList(t *testing.T) {
	tests := []struct {
		name      string
		schema    core.ResourceSchema
		resources []map[string]interface{}
		want      []string
		notWant   []string
	}{
		{
			name: "basic list with string fields",
			schema: core.ResourceSchema{
				Name: "Messages",
				Slug: "messages",
				Fields: []core.FieldSchema{
					{Name: "subject", Display: "Subject", Type: "string"},
					{Name: "from", Display: "From", Type: "email"},
				},
				ListColumns: []string{"subject", "from"},
			},
			resources: []map[string]interface{}{
				{"id": "msg1", "subject": "Hello", "from": "test@example.com"},
				{"id": "msg2", "subject": "World", "from": "user@example.com"},
			},
			want: []string{
				`<table class="min-w-full divide-y divide-gray-200">`,
				`<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Subject</th>`,
				`<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">From</th>`,
				`<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">Hello</td>`,
				`<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">test@example.com</td>`,
			},
		},
		{
			name: "empty resource list",
			schema: core.ResourceSchema{
				Name:        "Messages",
				Slug:        "messages",
				Fields:      []core.FieldSchema{{Name: "subject", Display: "Subject", Type: "string"}},
				ListColumns: []string{"subject"},
			},
			resources: []map[string]interface{}{},
			want: []string{
				`<table`,
				`<th`,
			},
		},
		{
			name: "list with actions column",
			schema: core.ResourceSchema{
				Name: "Messages",
				Slug: "messages",
				Fields: []core.FieldSchema{
					{Name: "subject", Display: "Subject", Type: "string"},
				},
				ListColumns: []string{"subject"},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/messages/{id}", Confirm: true},
				},
			},
			resources: []map[string]interface{}{
				{"id": "msg1", "subject": "Hello"},
			},
			want: []string{
				`<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderResourceList(tt.schema, tt.resources)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("RenderResourceList() missing expected content:\nwant: %s\ngot: %s", want, got)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("RenderResourceList() contains unexpected content:\nnotWant: %s\ngot: %s", notWant, got)
				}
			}
		})
	}
}

func TestRenderResourceForm(t *testing.T) {
	tests := []struct {
		name    string
		schema  core.ResourceSchema
		data    map[string]interface{}
		want    []string
		notWant []string
	}{
		{
			name: "create form with string and email fields",
			schema: core.ResourceSchema{
				Name: "Contact",
				Slug: "contacts",
				Fields: []core.FieldSchema{
					{Name: "name", Display: "Name", Type: "string", Required: true, Editable: true},
					{Name: "email", Display: "Email", Type: "email", Required: true, Editable: true},
				},
			},
			data: nil,
			want: []string{
				`<form`,
				`<label class="block text-sm font-medium text-gray-700">Name</label>`,
				`<input type="text" name="name" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				`<label class="block text-sm font-medium text-gray-700">Email</label>`,
				`<input type="email" name="email" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				`<button type="submit"`,
			},
		},
		{
			name: "edit form with pre-filled values",
			schema: core.ResourceSchema{
				Name: "Contact",
				Slug: "contacts",
				Fields: []core.FieldSchema{
					{Name: "name", Display: "Name", Type: "string", Editable: true},
					{Name: "email", Display: "Email", Type: "email", Editable: true},
				},
			},
			data: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			want: []string{
				`value="John Doe"`,
				`value="john@example.com"`,
			},
		},
		{
			name: "form with text area",
			schema: core.ResourceSchema{
				Name: "Message",
				Slug: "messages",
				Fields: []core.FieldSchema{
					{Name: "body", Display: "Body", Type: "text", Editable: true},
				},
			},
			data: nil,
			want: []string{
				`<textarea name="body"`,
				`class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border"`,
			},
		},
		{
			name: "form with checkbox",
			schema: core.ResourceSchema{
				Name: "Task",
				Slug: "tasks",
				Fields: []core.FieldSchema{
					{Name: "completed", Display: "Completed", Type: "checkbox", Editable: true},
				},
			},
			data: nil,
			want: []string{
				`<input type="checkbox" name="completed"`,
			},
		},
		{
			name: "form with datetime field",
			schema: core.ResourceSchema{
				Name: "Event",
				Slug: "events",
				Fields: []core.FieldSchema{
					{Name: "start_time", Display: "Start Time", Type: "datetime", Editable: true},
				},
			},
			data: nil,
			want: []string{
				`<input type="datetime-local" name="start_time"`,
			},
		},
		{
			name: "form with non-editable fields excluded",
			schema: core.ResourceSchema{
				Name: "Contact",
				Slug: "contacts",
				Fields: []core.FieldSchema{
					{Name: "id", Display: "ID", Type: "string", Editable: false},
					{Name: "name", Display: "Name", Type: "string", Editable: true},
				},
			},
			data: nil,
			want: []string{
				`name="name"`,
			},
			notWant: []string{
				`name="id"`,
			},
		},
		{
			name: "form with combobox field",
			schema: core.ResourceSchema{
				Name: "Task",
				Slug: "tasks",
				Fields: []core.FieldSchema{
					{Name: "priority", Display: "Priority", Type: "combobox", Editable: true},
				},
			},
			data: nil,
			want: []string{
				`<select name="priority"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderResourceForm(tt.schema, tt.data)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("RenderResourceForm() missing expected content:\nwant: %s\ngot: %s", want, got)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("RenderResourceForm() contains unexpected content:\nnotWant: %s\ngot: %s", notWant, got)
				}
			}
		})
	}
}

func TestRenderResourceDetail(t *testing.T) {
	tests := []struct {
		name   string
		schema core.ResourceSchema
		data   map[string]interface{}
		want   []string
	}{
		{
			name: "detail view with string fields",
			schema: core.ResourceSchema{
				Name: "Contact",
				Slug: "contacts",
				Fields: []core.FieldSchema{
					{Name: "name", Display: "Name", Type: "string"},
					{Name: "email", Display: "Email", Type: "email"},
				},
			},
			data: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			want: []string{
				`<dl class="divide-y divide-gray-200">`,
				`<dt class="text-sm font-medium text-gray-500">Name</dt>`,
				`<dd class="text-sm text-gray-900 col-span-2">John Doe</dd>`,
				`<dt class="text-sm font-medium text-gray-500">Email</dt>`,
				`<dd class="text-sm text-gray-900 col-span-2">john@example.com</dd>`,
			},
		},
		{
			name: "detail view with empty values",
			schema: core.ResourceSchema{
				Name: "Contact",
				Slug: "contacts",
				Fields: []core.FieldSchema{
					{Name: "email", Display: "Email", Type: "email"},
				},
			},
			data: map[string]interface{}{},
			want: []string{
				`<span class="text-gray-400">No value</span>`,
			},
		},
		{
			name: "detail view with checkbox",
			schema: core.ResourceSchema{
				Name: "Task",
				Slug: "tasks",
				Fields: []core.FieldSchema{
					{Name: "completed", Display: "Completed", Type: "checkbox"},
				},
			},
			data: map[string]interface{}{
				"completed": true,
			},
			want: []string{
				`Yes`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderResourceDetail(tt.schema, tt.data)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("RenderResourceDetail() missing expected content:\nwant: %s\ngot: %s", want, got)
				}
			}
		})
	}
}

func TestRenderActions(t *testing.T) {
	tests := []struct {
		name       string
		actions    []core.ActionSchema
		resourceID string
		want       []string
	}{
		{
			name: "actions with confirmation",
			actions: []core.ActionSchema{
				{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/messages/{id}", Confirm: true},
			},
			resourceID: "msg123",
			want: []string{
				`hx-delete="/messages/msg123"`,
				`hx-confirm="Delete this item?"`,
				`class="text-red-600 hover:text-red-900"`,
			},
		},
		{
			name: "actions without confirmation",
			actions: []core.ActionSchema{
				{Name: "archive", HTTPMethod: "POST", Endpoint: "/messages/{id}/archive", Confirm: false},
			},
			resourceID: "msg123",
			want: []string{
				`hx-post="/messages/msg123/archive"`,
			},
		},
		{
			name: "multiple actions",
			actions: []core.ActionSchema{
				{Name: "view", HTTPMethod: "GET", Endpoint: "/messages/{id}"},
				{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/messages/{id}", Confirm: true},
			},
			resourceID: "msg123",
			want: []string{
				`href="/messages/msg123"`,
				`hx-delete="/messages/msg123"`,
			},
		},
		{
			name:       "no actions",
			actions:    []core.ActionSchema{},
			resourceID: "msg123",
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderActions(tt.actions, tt.resourceID)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("RenderActions() missing expected content:\nwant: %s\ngot: %s", want, got)
				}
			}
		})
	}
}

func TestRenderActions_HTTPMethodMapping(t *testing.T) {
	tests := []struct {
		name       string
		httpMethod string
		want       string
	}{
		{"GET uses href", "GET", `href="`},
		{"POST uses hx-post", "POST", `hx-post="`},
		{"DELETE uses hx-delete", "DELETE", `hx-delete="`},
		{"PUT uses hx-put", "PUT", `hx-put="`},
		{"PATCH uses hx-patch", "PATCH", `hx-patch="`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := []core.ActionSchema{
				{Name: "test", HTTPMethod: tt.httpMethod, Endpoint: "/test/{id}"},
			}
			got := RenderActions(actions, "123")
			if !strings.Contains(got, tt.want) {
				t.Errorf("RenderActions() for %s method missing %s in output: %s", tt.httpMethod, tt.want, got)
			}
		})
	}
}

func TestRenderResourceForm_AllFieldTypes(t *testing.T) {
	schema := core.ResourceSchema{
		Name: "CompleteForm",
		Slug: "complete",
		Fields: []core.FieldSchema{
			{Name: "text_field", Display: "Text", Type: "string", Editable: true},
			{Name: "email_field", Display: "Email", Type: "email", Editable: true},
			{Name: "textarea_field", Display: "Description", Type: "text", Editable: true},
			{Name: "datetime_field", Display: "Date", Type: "datetime", Editable: true},
			{Name: "checkbox_field", Display: "Active", Type: "checkbox", Editable: true},
			{Name: "radio_field", Display: "Choice", Type: "radio", Editable: true},
			{Name: "combobox_field", Display: "Select", Type: "combobox", Editable: true},
			{Name: "slider_field", Display: "Volume", Type: "slider", Editable: true},
		},
	}

	got := RenderResourceForm(schema, nil)

	expectedTypes := []string{
		`type="text"`,
		`type="email"`,
		`<textarea`,
		`type="datetime-local"`,
		`type="checkbox"`,
		`type="radio"`,
		`<select`,
		`type="range"`,
	}

	for _, expected := range expectedTypes {
		if !strings.Contains(got, expected) {
			t.Errorf("RenderResourceForm() missing field type: %s", expected)
		}
	}
}

func TestRenderResourceDetail_CheckboxFormatting(t *testing.T) {
	schema := core.ResourceSchema{
		Name: "Task",
		Fields: []core.FieldSchema{
			{Name: "completed", Display: "Completed", Type: "checkbox"},
		},
	}

	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"true value", true, "Yes"},
		{"false value", false, "No"},
		{"nil value", nil, "No"},
		{"string true", "true", "Yes"},
		{"string false", "false", "No"},
		{"int 1", 1, "Yes"},
		{"int 0", 0, "No"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{"completed": tt.value}
			got := RenderResourceDetail(schema, data)
			if !strings.Contains(got, tt.want) {
				t.Errorf("RenderResourceDetail() checkbox value = %v, want %s in output, got: %s", tt.value, tt.want, got)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("formatValue with nil", func(t *testing.T) {
		if got := formatValue(nil); got != "" {
			t.Errorf("formatValue(nil) = %q, want empty string", got)
		}
	})

	t.Run("formatValue with string", func(t *testing.T) {
		if got := formatValue("test"); got != "test" {
			t.Errorf("formatValue(\"test\") = %q, want \"test\"", got)
		}
	})

	t.Run("formatValue with int", func(t *testing.T) {
		if got := formatValue(123); got != "123" {
			t.Errorf("formatValue(123) = %q, want \"123\"", got)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty schema", func(t *testing.T) {
		schema := core.ResourceSchema{
			Name:   "Empty",
			Slug:   "empty",
			Fields: []core.FieldSchema{},
		}
		got := RenderResourceForm(schema, nil)
		if !strings.Contains(got, "<form") {
			t.Error("RenderResourceForm() should still generate form tag for empty schema")
		}
	})

	t.Run("nil data in form", func(t *testing.T) {
		schema := core.ResourceSchema{
			Name: "Test",
			Fields: []core.FieldSchema{
				{Name: "field1", Display: "Field 1", Type: "string", Editable: true},
			},
		}
		got := RenderResourceForm(schema, nil)
		if !strings.Contains(got, `name="field1"`) {
			t.Error("RenderResourceForm() should handle nil data")
		}
	})
}
