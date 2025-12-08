// ABOUTME: Home Assistant plugin admin UI schema definitions
// ABOUTME: Defines resources exposed in the admin interface
package homeassistant

import "github.com/2389/ish/plugins/core"

func GetResourceSchemas() []core.ResourceSchema {
	return []core.ResourceSchema{
		{
			Name: "Instances",
			Slug: "instances",
			Fields: []core.FieldSchema{
				{Name: "id", Type: "integer", Display: "ID"},
				{Name: "name", Type: "string", Display: "Name"},
				{Name: "url", Type: "string", Display: "URL"},
				{Name: "created_at", Type: "datetime", Display: "Created"},
				{Name: "updated_at", Type: "datetime", Display: "Updated"},
			},
			ListColumns: []string{"id", "name", "url", "created_at"},
		},
		{
			Name: "Entities",
			Slug: "entities",
			Fields: []core.FieldSchema{
				{Name: "id", Type: "integer", Display: "ID"},
				{Name: "instance_id", Type: "integer", Display: "Instance ID"},
				{Name: "entity_id", Type: "string", Display: "Entity ID"},
				{Name: "friendly_name", Type: "string", Display: "Friendly Name"},
				{Name: "domain", Type: "string", Display: "Domain"},
				{Name: "platform", Type: "string", Display: "Platform"},
				{Name: "created_at", Type: "datetime", Display: "Created"},
				{Name: "updated_at", Type: "datetime", Display: "Updated"},
			},
			ListColumns: []string{"entity_id", "friendly_name", "domain", "platform"},
		},
		{
			Name: "States",
			Slug: "states",
			Fields: []core.FieldSchema{
				{Name: "id", Type: "integer", Display: "ID"},
				{Name: "instance_id", Type: "integer", Display: "Instance ID"},
				{Name: "entity_id", Type: "string", Display: "Entity ID"},
				{Name: "state", Type: "string", Display: "State"},
				{Name: "attributes", Type: "text", Display: "Attributes"},
				{Name: "last_changed", Type: "datetime", Display: "Last Changed"},
				{Name: "last_updated", Type: "datetime", Display: "Last Updated"},
				{Name: "created_at", Type: "datetime", Display: "Created"},
			},
			ListColumns: []string{"entity_id", "state", "last_changed", "last_updated"},
		},
		{
			Name: "Service Calls",
			Slug: "service_calls",
			Fields: []core.FieldSchema{
				{Name: "id", Type: "integer", Display: "ID"},
				{Name: "instance_id", Type: "integer", Display: "Instance ID"},
				{Name: "domain", Type: "string", Display: "Domain"},
				{Name: "service", Type: "string", Display: "Service"},
				{Name: "entity_id", Type: "string", Display: "Entity ID"},
				{Name: "service_data", Type: "text", Display: "Service Data"},
				{Name: "status", Type: "string", Display: "Status"},
				{Name: "called_at", Type: "datetime", Display: "Called At"},
				{Name: "created_at", Type: "datetime", Display: "Created"},
			},
			ListColumns: []string{"domain", "service", "entity_id", "status", "called_at"},
		},
	}
}
