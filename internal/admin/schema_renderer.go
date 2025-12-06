// ABOUTME: Schema-based HTML renderer for plugin UIs.
// ABOUTME: Generates semantic HTML with Tailwind CSS from plugin schemas.

package admin

import (
	"fmt"
	"html"
	"strings"

	"github.com/2389/ish/plugins/core"
)

// RenderResourceList generates a table list view from a ResourceSchema
func RenderResourceList(schema core.ResourceSchema, resources []map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(`<table class="min-w-full divide-y divide-gray-200">`)
	sb.WriteString(`<thead class="bg-gray-50"><tr>`)

	// Render column headers
	for _, colName := range schema.ListColumns {
		field := findField(schema.Fields, colName)
		if field != nil {
			sb.WriteString(fmt.Sprintf(`<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">%s</th>`,
				html.EscapeString(field.Display)))
		}
	}

	// Add actions column if there are actions
	if len(schema.Actions) > 0 {
		sb.WriteString(`<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>`)
	}

	sb.WriteString(`</tr></thead>`)
	sb.WriteString(`<tbody class="bg-white divide-y divide-gray-200">`)

	// Render rows
	for _, resource := range resources {
		sb.WriteString(`<tr>`)

		for _, colName := range schema.ListColumns {
			value := resource[colName]
			sb.WriteString(fmt.Sprintf(`<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">%s</td>`,
				html.EscapeString(formatValue(value))))
		}

		// Render actions cell
		if len(schema.Actions) > 0 {
			resourceID := fmt.Sprint(resource["id"])
			sb.WriteString(`<td class="px-6 py-4 whitespace-nowrap text-right text-sm space-x-3">`)
			sb.WriteString(RenderActions(schema.Actions, resourceID))
			sb.WriteString(`</td>`)
		}

		sb.WriteString(`</tr>`)
	}

	sb.WriteString(`</tbody></table>`)
	return sb.String()
}

// RenderResourceForm generates a create/edit form from a ResourceSchema
func RenderResourceForm(schema core.ResourceSchema, data map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(`<form class="bg-white rounded-lg shadow p-6 space-y-4 max-w-2xl">`)

	for _, field := range schema.Fields {
		if !field.Editable {
			continue
		}

		sb.WriteString(`<div>`)
		sb.WriteString(fmt.Sprintf(`<label class="block text-sm font-medium text-gray-700">%s</label>`,
			html.EscapeString(field.Display)))

		value := ""
		if data != nil {
			if v, ok := data[field.Name]; ok {
				value = fmt.Sprint(v)
			}
		}

		switch field.Type {
		case "text":
			sb.WriteString(fmt.Sprintf(`<textarea name="%s" %s class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">%s</textarea>`,
				html.EscapeString(field.Name),
				requiredAttr(field.Required),
				html.EscapeString(value)))

		case "checkbox":
			checked := ""
			if value == "true" || value == "1" {
				checked = "checked"
			}
			sb.WriteString(fmt.Sprintf(`<input type="checkbox" name="%s" %s class="mt-1 rounded border-gray-300">`,
				html.EscapeString(field.Name),
				checked))

		case "datetime":
			sb.WriteString(fmt.Sprintf(`<input type="datetime-local" name="%s" value="%s" %s class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				html.EscapeString(field.Name),
				html.EscapeString(value),
				requiredAttr(field.Required)))

		case "combobox":
			sb.WriteString(fmt.Sprintf(`<select name="%s" %s class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				html.EscapeString(field.Name),
				requiredAttr(field.Required)))
			sb.WriteString(`<option value="">Select...</option>`)
			sb.WriteString(`</select>`)

		case "radio":
			sb.WriteString(fmt.Sprintf(`<input type="radio" name="%s" class="mt-1 rounded border-gray-300">`,
				html.EscapeString(field.Name)))

		case "slider":
			sb.WriteString(fmt.Sprintf(`<input type="range" name="%s" value="%s" class="mt-1 block w-full">`,
				html.EscapeString(field.Name),
				html.EscapeString(value)))

		case "email":
			valueAttr := ""
			if value != "" {
				valueAttr = fmt.Sprintf(` value="%s"`, html.EscapeString(value))
			}
			sb.WriteString(fmt.Sprintf(`<input type="email" name="%s"%s %s class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				html.EscapeString(field.Name),
				valueAttr,
				requiredAttr(field.Required)))

		default: // string and others
			valueAttr := ""
			if value != "" {
				valueAttr = fmt.Sprintf(` value="%s"`, html.EscapeString(value))
			}
			sb.WriteString(fmt.Sprintf(`<input type="text" name="%s"%s %s class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">`,
				html.EscapeString(field.Name),
				valueAttr,
				requiredAttr(field.Required)))
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<div class="flex gap-4">`)
	sb.WriteString(`<button type="submit" class="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700">Save</button>`)
	sb.WriteString(`<button type="button" class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">Cancel</button>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`</form>`)
	return sb.String()
}

// RenderResourceDetail generates a detail view from a ResourceSchema
func RenderResourceDetail(schema core.ResourceSchema, data map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(`<div class="bg-white rounded-lg shadow overflow-hidden">`)
	sb.WriteString(`<dl class="divide-y divide-gray-200">`)

	for _, field := range schema.Fields {
		sb.WriteString(`<div class="px-6 py-4 grid grid-cols-3 gap-4">`)
		sb.WriteString(fmt.Sprintf(`<dt class="text-sm font-medium text-gray-500">%s</dt>`,
			html.EscapeString(field.Display)))

		value := data[field.Name]
		displayValue := formatDetailValue(field.Type, value)

		sb.WriteString(fmt.Sprintf(`<dd class="text-sm text-gray-900 col-span-2">%s</dd>`,
			displayValue))
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</dl></div>`)
	return sb.String()
}

// RenderActions generates action buttons from ActionSchema
func RenderActions(actions []core.ActionSchema, resourceID string) string {
	var sb strings.Builder

	for i, action := range actions {
		if i > 0 {
			sb.WriteString(" ")
		}

		endpoint := strings.ReplaceAll(action.Endpoint, "{id}", resourceID)

		if action.HTTPMethod == "GET" {
			// Regular link for GET
			sb.WriteString(fmt.Sprintf(`<a href="%s" class="text-blue-600 hover:text-blue-900">%s</a>`,
				html.EscapeString(endpoint),
				html.EscapeString(strings.Title(action.Name))))
		} else {
			// HTMX for other methods
			htmxAttr := getHTMXAttribute(action.HTTPMethod)
			confirmAttr := ""
			if action.Confirm {
				confirmAttr = ` hx-confirm="Delete this item?"`
			}

			cssClass := "text-blue-600 hover:text-blue-900"
			if action.HTTPMethod == "DELETE" {
				cssClass = "text-red-600 hover:text-red-900"
			}

			sb.WriteString(fmt.Sprintf(`<button %s="%s"%s class="%s">%s</button>`,
				htmxAttr,
				html.EscapeString(endpoint),
				confirmAttr,
				cssClass,
				html.EscapeString(strings.Title(action.Name))))
		}
	}

	return sb.String()
}

// Helper functions

func findField(fields []core.FieldSchema, name string) *core.FieldSchema {
	for _, field := range fields {
		if field.Name == name {
			return &field
		}
	}
	return nil
}

func formatValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func formatDetailValue(fieldType string, value interface{}) string {
	if value == nil {
		return `<span class="text-gray-400">No value</span>`
	}

	if fieldType == "checkbox" {
		if isTruthy(value) {
			return "Yes"
		}
		return "No"
	}

	strValue := fmt.Sprint(value)
	if strValue == "" {
		return `<span class="text-gray-400">No value</span>`
	}

	return html.EscapeString(strValue)
}

func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case int:
		return v != 0
	default:
		return false
	}
}

func requiredAttr(required bool) string {
	if required {
		return "required"
	}
	return ""
}

func getHTMXAttribute(method string) string {
	switch method {
	case "POST":
		return "hx-post"
	case "DELETE":
		return "hx-delete"
	case "PUT":
		return "hx-put"
	case "PATCH":
		return "hx-patch"
	default:
		return "hx-post"
	}
}
