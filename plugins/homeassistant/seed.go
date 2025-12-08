// ABOUTME: Test data generation for Home Assistant plugin
// ABOUTME: Creates sample instances, entities, states, and service calls
package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/2389/ish/plugins/core"
)

// Seed creates test data for the Home Assistant plugin
func (p *HomeAssistantPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var numInstances, numEntitiesPerInstance, numStatesPerEntity, numServiceCalls int

	switch size {
	case "small":
		numInstances, numEntitiesPerInstance, numStatesPerEntity, numServiceCalls = 1, 3, 2, 2
	case "medium":
		numInstances, numEntitiesPerInstance, numStatesPerEntity, numServiceCalls = 2, 6, 3, 5
	case "large":
		numInstances, numEntitiesPerInstance, numStatesPerEntity, numServiceCalls = 3, 10, 5, 10
	default:
		numInstances, numEntitiesPerInstance, numStatesPerEntity, numServiceCalls = 2, 6, 3, 5
	}

	// Create test instances
	instances := []struct {
		url   string
		token string
		name  string
	}{
		{"http://homeassistant.local:8123", "token_home_main", "Home"},
		{"http://192.168.1.100:8123", "token_vacation_house", "Vacation House"},
		{"http://cabin.example.com:8123", "token_cabin", "Mountain Cabin"},
	}

	instanceIDs := make([]int64, 0, len(instances))
	for i := 0; i < numInstances && i < len(instances); i++ {
		inst := instances[i]
		instance, err := p.store.CreateInstance(inst.url, inst.token, inst.name)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to create instance %s: %w", inst.name, err)
		}
		instanceIDs = append(instanceIDs, instance.ID)
	}

	// Create sample entities for each instance
	entities := []struct {
		entityID     string
		friendlyName string
		domain       string
		platform     string
	}{
		{"light.living_room", "Living Room Light", "light", "hue"},
		{"light.bedroom", "Bedroom Light", "light", "hue"},
		{"switch.kitchen_outlet", "Kitchen Outlet", "switch", "zwave"},
		{"sensor.temperature_living_room", "Living Room Temperature", "sensor", "mqtt"},
		{"sensor.humidity_bedroom", "Bedroom Humidity", "sensor", "mqtt"},
		{"climate.thermostat", "Main Thermostat", "climate", "nest"},
		{"lock.front_door", "Front Door Lock", "lock", "august"},
		{"binary_sensor.motion_hallway", "Hallway Motion", "binary_sensor", "zwave"},
		{"cover.garage_door", "Garage Door", "cover", "myq"},
		{"media_player.living_room_tv", "Living Room TV", "media_player", "roku"},
	}

	totalEntities := 0
	for _, instanceID := range instanceIDs {
		for i := 0; i < numEntitiesPerInstance && i < len(entities); i++ {
			ent := entities[i]
			err := p.store.CreateOrUpdateEntity(instanceID, ent.entityID, ent.friendlyName, ent.domain, ent.platform)
			if err != nil {
				return core.SeedData{}, fmt.Errorf("failed to create entity %s: %w", ent.entityID, err)
			}
			totalEntities++
		}
	}

	// Create sample states for entities
	stateTemplates := []struct {
		entityID   string
		states     []string
		attributes map[string]interface{}
	}{
		{
			"light.living_room",
			[]string{"on", "off"},
			map[string]interface{}{"brightness": 255, "color_temp": 370},
		},
		{
			"light.bedroom",
			[]string{"on", "off"},
			map[string]interface{}{"brightness": 128, "color_temp": 400},
		},
		{
			"switch.kitchen_outlet",
			[]string{"on", "off"},
			map[string]interface{}{"current_power_w": 150.5},
		},
		{
			"sensor.temperature_living_room",
			[]string{"72.5", "71.8", "73.2"},
			map[string]interface{}{"unit_of_measurement": "°F", "device_class": "temperature"},
		},
		{
			"sensor.humidity_bedroom",
			[]string{"45", "46", "44"},
			map[string]interface{}{"unit_of_measurement": "%", "device_class": "humidity"},
		},
		{
			"climate.thermostat",
			[]string{"heat", "cool", "auto"},
			map[string]interface{}{"temperature": 72, "target_temp_high": 75, "target_temp_low": 68},
		},
		{
			"lock.front_door",
			[]string{"locked", "unlocked"},
			map[string]interface{}{"battery_level": 85},
		},
		{
			"binary_sensor.motion_hallway",
			[]string{"on", "off"},
			map[string]interface{}{"device_class": "motion"},
		},
		{
			"cover.garage_door",
			[]string{"open", "closed", "opening", "closing"},
			map[string]interface{}{"current_position": 100},
		},
		{
			"media_player.living_room_tv",
			[]string{"playing", "paused", "idle"},
			map[string]interface{}{"volume_level": 0.5, "source": "Netflix"},
		},
	}

	totalStates := 0
	now := time.Now()
	for _, instanceID := range instanceIDs {
		for i := 0; i < numEntitiesPerInstance && i < len(stateTemplates); i++ {
			tmpl := stateTemplates[i]
			for j := 0; j < numStatesPerEntity && j < len(tmpl.states); j++ {
				state := tmpl.states[j]
				attributesJSON, err := json.Marshal(tmpl.attributes)
				if err != nil {
					return core.SeedData{}, fmt.Errorf("failed to marshal attributes: %w", err)
				}

				timestamp := now.Add(-time.Duration(j) * time.Hour)
				err = p.store.RecordState(instanceID, tmpl.entityID, state, string(attributesJSON), timestamp, timestamp)
				if err != nil {
					return core.SeedData{}, fmt.Errorf("failed to record state for %s: %w", tmpl.entityID, err)
				}
				totalStates++
			}
		}
	}

	// Create sample service calls
	serviceCalls := []struct {
		domain      string
		service     string
		entityID    string
		serviceData map[string]interface{}
		status      string
	}{
		{
			"light", "turn_on", "light.living_room",
			map[string]interface{}{"brightness": 255, "color_temp": 370},
			"success",
		},
		{
			"light", "turn_off", "light.bedroom",
			map[string]interface{}{},
			"success",
		},
		{
			"climate", "set_temperature", "climate.thermostat",
			map[string]interface{}{"temperature": 72},
			"success",
		},
		{
			"lock", "lock", "lock.front_door",
			map[string]interface{}{},
			"success",
		},
		{
			"cover", "open_cover", "cover.garage_door",
			map[string]interface{}{},
			"success",
		},
		{
			"media_player", "media_play", "media_player.living_room_tv",
			map[string]interface{}{},
			"success",
		},
		{
			"switch", "turn_on", "switch.kitchen_outlet",
			map[string]interface{}{},
			"success",
		},
		{
			"light", "toggle", "light.living_room",
			map[string]interface{}{},
			"success",
		},
		{
			"climate", "set_hvac_mode", "climate.thermostat",
			map[string]interface{}{"hvac_mode": "heat"},
			"success",
		},
		{
			"cover", "close_cover", "cover.garage_door",
			map[string]interface{}{},
			"success",
		},
	}

	totalServiceCalls := 0
	for i := 0; i < numServiceCalls && i < len(serviceCalls); i++ {
		call := serviceCalls[i]
		serviceDataJSON, err := json.Marshal(call.serviceData)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to marshal service data: %w", err)
		}

		timestamp := now.Add(-time.Duration(i) * time.Minute * 30)
		err = p.store.RecordServiceCall(instanceIDs[0], call.domain, call.service, string(serviceDataJSON), call.entityID, call.status, timestamp)
		if err != nil {
			return core.SeedData{}, fmt.Errorf("failed to record service call %s.%s: %w", call.domain, call.service, err)
		}
		totalServiceCalls++
	}

	// Print instance tokens for testing
	fmt.Println("\n=== Home Assistant Test Tokens ===")
	for i := 0; i < len(instanceIDs) && i < len(instances); i++ {
		fmt.Printf("Instance: %s (%s)\n", instances[i].name, instances[i].url)
		fmt.Printf("Token: %s\n", instances[i].token)
		fmt.Printf("Test: curl -H 'Authorization: Bearer %s' http://localhost:PORT/api/states\n\n", instances[i].token)
	}

	return core.SeedData{
		Summary: fmt.Sprintf("Created %d instances, %d entities, %d states, %d service calls",
			len(instanceIDs), totalEntities, totalStates, totalServiceCalls),
		Records: map[string]int{
			"instances":     len(instanceIDs),
			"entities":      totalEntities,
			"states":        totalStates,
			"service_calls": totalServiceCalls,
		},
	}, nil
}
